package launch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/spikesdivzero/launch-control/internal/component"
	"github.com/spikesdivzero/launch-control/internal/debug"
)

type optionNilArgError struct{ funcName, argName string }

func (e optionNilArgError) Error() string {
	return fmt.Sprintf("%s: %s must not be nil", e.funcName, e.argName)
}

type optionConflictingCallsError struct {
	errStr string
	stacks [][2]string
}

func (e optionConflictingCallsError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.errStr)
	sb.WriteString("\n\n")
	sb.WriteString("Conflicting calls:\n\n")
	for _, c := range e.stacks {
		funcName, stack := c[0], c[1]
		sb.WriteString("## Call to ")
		sb.WriteString(funcName)
		sb.WriteString("\n")
		sb.WriteString(stack)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

type componentBuildState struct {
	c   *component.Component
	ssw *component.StartStopWrapper

	// The options pattern leads to the potential for runtime panics. We'll collect the stacks and, if a violation is found,
	// report them with the call stacks to make it a bit easier for the dev to trace down where it happened.
	appliedRunCalls        [][2]string // {funcName, stack}
	appliedCheckReadyCalls [][2]string // the funcName is always the same here, but using the same time lets us share an error
}

func newComponentBuildState(name string) *componentBuildState {
	return &componentBuildState{
		c: &component.Component{
			Name: name,

			ShutdownOptions: component.ShutdownOptions{
				CallTimeout:       NoTimeout,
				CompletionTimeout: NoTimeout,
			},
			CheckReadyOptions: component.CheckReadyOptions{
				CallTimeout: NoTimeout,
				Backoff:     ConstBackoff(0),
				MaxAttempts: math.MaxInt,
			},
		},
		ssw: &component.StartStopWrapper{
			StartTimeout: NoTimeout,
			StopTimeout:  NoTimeout,
		},
	}
}

func buildComponent(name string, opts ...ComponentOption) (*component.Component, error) {
	cbs := newComponentBuildState(name)
	for _, opt := range opts {
		opt(cbs)
	}

	if len(cbs.appliedRunCalls) == 0 {
		return nil, errors.New("must provide either WithRun or WithStartStop")
	}

	if len(cbs.appliedRunCalls) > 1 {
		return nil, optionConflictingCallsError{
			"multiple calls to WithRun/WithStartStop; must be exactly one",
			cbs.appliedRunCalls,
		}
	}

	if cbs.c.ImplRun == nil {
		cbs.c.ImplRun = cbs.ssw.Run
		cbs.c.ImplShutdown = cbs.ssw.Shutdown
	}

	if len(cbs.appliedCheckReadyCalls) > 1 {
		return nil, optionConflictingCallsError{
			"multiple calls to WithCheckReady; must be zero or one",
			cbs.appliedCheckReadyCalls,
		}
	}

	return cbs.c, nil
}

// If you don't want something to have a timeout, you can use this as a convenience.
//
// Truthfully, the constant value is a bit less than 50 years, which isn't the same as saying
// no timeout, but... what're the odds you're going to leave something running for that long?
//
// Why 50 years? No reason, except that it's unreasonably large, and will fit within Go's
// time.Time type for a long time to come. (time.maxWall is the year 2157, 132 years from now)
const NoTimeout time.Duration = 50 * (time.Hour * 24 * 365)

type ComponentOption func(*componentBuildState)

func WithBundledOptions(opts ...ComponentOption) ComponentOption {
	return func(cbs *componentBuildState) {
		for _, opt := range opts {
			opt(cbs)
		}
	}
}

func WithRun(
	run func(context.Context) error,
	shutdown func(context.Context) error,
) ComponentOption {
	if run == nil {
		panic(optionNilArgError{"WithRun", "run"})
	}
	if shutdown == nil {
		panic(optionNilArgError{"WithRun", "shutdown"})
	}

	stack := debug.TidyStack(1)
	return func(cbs *componentBuildState) {
		cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"WithRun", stack})
		cbs.c.ImplRun = run
		cbs.c.ImplShutdown = shutdown
	}
}

func WithShutdownCallTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.ShutdownOptions.CallTimeout = d
	}
}

func WithShutdownCompletionTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.ShutdownOptions.CompletionTimeout = d
	}
}

func WithStartStop(
	start func(context.Context) error,
	stop func(context.Context) error,
) ComponentOption {
	if start == nil {
		panic(optionNilArgError{"WithStartStop", "start"})
	}
	if stop == nil {
		panic(optionNilArgError{"WithStartStop", "stop"})
	}

	stack := debug.TidyStack(1)
	return func(cbs *componentBuildState) {
		cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"WithStartStop", stack})
		cbs.ssw.ImplStart = start
		cbs.ssw.ImplStop = stop
	}
}

// FIXME: not a big fan of this name, or having both timeouts in one call.
// I didn't like the idea of "WithStartStopStartTimeout".
// And for world history reasons, I didn't like the idea of abbreviating StartStop to a double-S.
func WithStartStopCallTimeouts(startD, stopD time.Duration) ComponentOption {
	if startD <= 0 {
		startD = NoTimeout
	}
	if stopD <= 0 {
		stopD = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.ssw.StartTimeout = startD
		cbs.ssw.StopTimeout = stopD
	}
}

func WithCheckReady(
	checkReady func(context.Context) (bool, error),
) ComponentOption {
	if checkReady == nil {
		panic(optionNilArgError{"WithCheckReady", "checkReady"})
	}

	stack := debug.TidyStack(1)
	return func(cbs *componentBuildState) {
		cbs.appliedCheckReadyCalls = append(cbs.appliedCheckReadyCalls, [2]string{"WithCheckReady", stack})
		cbs.c.ImplCheckReady = checkReady
	}
}

func WithCheckReadyCallTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.CheckReadyOptions.CallTimeout = d
	}
}

func WithCheckReadyBackoff(
	backoff BackoffFunc,
) ComponentOption {
	if backoff == nil {
		panic(optionNilArgError{"WithCheckReadyBackoff", "backoff"})
	}

	return func(cbs *componentBuildState) {
		cbs.c.CheckReadyOptions.Backoff = backoff
	}
}

func WithCheckReadyMaxAttempts(n int) ComponentOption {
	if n <= 0 {
		n = math.MaxInt
	}

	return func(cbs *componentBuildState) {
		cbs.c.CheckReadyOptions.MaxAttempts = n
	}
}
