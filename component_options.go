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

// FIXME: Document this.
type Options126 struct {
	// WithRun
	Run      func(context.Context) error
	Shutdown func(context.Context) error

	ShutdownCallTimeout       *time.Duration
	ShutdownCompletionTimeout *time.Duration

	// WithStartStop
	Start func(context.Context) error
	Stop  func(context.Context) error

	StartCallTimeout *time.Duration
	StopCallTimeout  *time.Duration

	// WithCheckReady
	CheckReady func(context.Context) (bool, error)

	CheckReadyCallTimeout *time.Duration

	CheckReadyBackoff     BackoffFunc
	CheckReadyMaxAttempts *int
}

// Determines the run style for this configuration.
// If the function set in this Options126 is invalid, it panics.
// If there's no run configuration in this options, returns empty string.
func (co Options126) getRunStyle() string {
	hasRun, hasShutdown := co.Run != nil, co.Shutdown != nil
	if hasRun != hasShutdown {
		panic("Options126 has Run or Shutdown, but not both (incomplete configuration)")
	}

	hasStart, hasStop := co.Start != nil, co.Stop != nil
	if hasStart != hasStop {
		panic("Options126 has Start or Stop, but not both (incomplete configuration)")
	}

	if hasRun && hasStart {
		panic("Options126 has both Run and Start; can only have one run style, not both (invalid configuration)")
	}

	if hasRun {
		return "Run+Shutdown"
	}
	if hasStart {
		return "Start+Stop"
	}
	return ""
}

func (co *Options126) applyOptions(from Options126) {
	applyDuration := func(dest **time.Duration, src *time.Duration) {
		if src != nil {
			*dest = src
		}
	}

	// A run style should only ever be in a single component options (ideally the final one).
	// We regard it as a hard error to provide it more than once, since the meanings would be confounding.
	//
	// NOTE: We always call from.getRunStyle on the outside as it validates the function set configurations,
	// and we rely on those validations later on
	if fromStyle := from.getRunStyle(); fromStyle != "" {
		if prevStyle := co.getRunStyle(); prevStyle != "" {
			panic(fmt.Sprintf("Options126: cannot merge multiple run styles: previous %q, this %q", prevStyle, fromStyle))
		}
	}

	if from.Run != nil {
		co.Run = from.Run
		co.Shutdown = from.Shutdown
	}

	applyDuration(&co.ShutdownCallTimeout, from.ShutdownCallTimeout)
	applyDuration(&co.ShutdownCompletionTimeout, from.ShutdownCompletionTimeout)

	if from.Start != nil {
		co.Start = from.Start
		co.Stop = from.Stop
	}

	applyDuration(&co.StartCallTimeout, from.StartCallTimeout)
	applyDuration(&co.StopCallTimeout, from.StopCallTimeout)

	if from.CheckReady != nil {
		if co.CheckReady != nil {
			panic("Options126: cannot merge multiple CheckReady funcs")
		}
		co.CheckReady = from.CheckReady
	}

	applyDuration(&co.CheckReadyCallTimeout, from.CheckReadyCallTimeout)

	if from.CheckReadyBackoff != nil {
		co.CheckReadyBackoff = from.CheckReadyBackoff
	}
	if from.CheckReadyMaxAttempts != nil {
		co.CheckReadyMaxAttempts = from.CheckReadyMaxAttempts
	}
}

func (co *Options126) finalize() error {
	// A valid duration is non-nil, positive, and greater than zero.
	applyDefaultDuration := func(dest **time.Duration, src time.Duration) {
		if *dest == nil || **dest <= 0 {
			*dest = &src
		}
	}

	if co.getRunStyle() == "" {
		return errors.New("must provide either Run+Shutdown or Start+Stop")
	}

	applyDefaultDuration(&co.ShutdownCallTimeout, NoTimeout)
	applyDefaultDuration(&co.ShutdownCompletionTimeout, NoTimeout)

	applyDefaultDuration(&co.StartCallTimeout, NoTimeout)
	applyDefaultDuration(&co.StopCallTimeout, NoTimeout)

	// CheckReady can be missing.

	applyDefaultDuration(&co.CheckReadyCallTimeout, NoTimeout)

	if co.CheckReadyBackoff == nil {
		co.CheckReadyBackoff = func() time.Duration { return 0 }
	}
	if co.CheckReadyMaxAttempts == nil || *co.CheckReadyMaxAttempts <= 0 {
		co.CheckReadyMaxAttempts = new(math.MaxInt)
	}

	return nil
}

func buildComponent126(name string, opts ...Options126) (*component.Component, error) {
	if name == "" {
		return nil, errors.New("name must not be empty")
	}

	co := Options126{}
	for _, opt := range opts {
		co.applyOptions(opt)
	}
	if err := co.finalize(); err != nil {
		fmt.Printf("%#v\n", co)
		panic(fmt.Errorf("TODO: handle error: %w", err))
	}

	c := component.New(name)

	// Run style is validated over in finalize().
	// If we're using SSW, then set it up and assign Run/Shutdown with the SSW provided ones.
	if co.Start != nil {
		ssw := component.NewStartStopWrapperFor(c)

		ssw.ImplStart = co.Start
		ssw.ImplStop = co.Stop

		ssw.StartTimeout = *co.StartCallTimeout
		ssw.StopTimeout = *co.StopCallTimeout

		co.Run = ssw.Run
		co.Shutdown = ssw.Shutdown
	}

	c.ImplRun = co.Run
	c.ImplShutdown = co.Shutdown
	c.ImplCheckReady = co.CheckReady

	c.ShutdownOptions.CallTimeout = *co.ShutdownCallTimeout
	c.ShutdownOptions.CompletionTimeout = *co.ShutdownCompletionTimeout

	c.CheckReadyOptions.CallTimeout = *co.CheckReadyCallTimeout
	c.CheckReadyOptions.Backoff = co.CheckReadyBackoff
	c.CheckReadyOptions.MaxAttempts = *co.CheckReadyMaxAttempts

	return c, nil
}

//
//
//
//
//
//
//
//
//
//
//
//
//
//
// FIXME: Most everything below here will go away...
//
//
//
//
//
//
//
//
//
//
//
//
//
//

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
	c := component.New(name)
	return &componentBuildState{
		c:   c,
		ssw: component.NewStartStopWrapperFor(c),
	}
}

func buildComponent(name string, opts ...ComponentOption) (*component.Component, error) {
	if name == "" {
		return nil, errors.New("name must not be empty")
	}

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

// Combines a set of options into a single ComponentOption.
//
// Great for use in defining your application's default options:
//
//	myDefaultOptions := WithBundledOptions( /* ...all your defaults */ )
//	ctrl.Launch("name", myDefaultOptions, /* ...other options */ )
func WithBundledOptions(opts ...ComponentOption) ComponentOption {
	return func(cbs *componentBuildState) {
		for _, opt := range opts {
			opt(cbs)
		}
	}
}

// Defines the main `Run` and `Shutdown` functions that control the component's lifecycle.
//
// `Run` is a blocking function, that returns when the component's has exited (or has started exiting via `Shutdown`).
//
// If `Run` returns an error, then the error will be passed up to the controller and the controller will transition
// into a failed/shutting down state.
//
// `Shutdown` is called when it's time to terminate the component. The shutdown process is not considered complete
// until both `Run` and `Shutdown` have finished, or their corresponding timeouts have completed.
//
// If you've also provided [WithCheckReady], it's worth noting that `Shutdown` may be called at any point. The
// `CheckReady` function may or may not have been called. If the `CheckReady` call timed out, then it may still be
// running in another coroutine.
//
// Constraints: [WithRun] may only be provided once, and is mutually exclusive with [WithStartStop].
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

// Applies a call-duration timeout to the `Shutdown` function provided to [WithRun].
//
// In the event that both this and [WithShutdownCompletionTimeout] are provided, the call timeout
// will be the lesser of the two durations.
//
// If not provided, it defaults to [NoTimeout].
func WithShutdownCallTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.ShutdownOptions.CallTimeout = d
	}
}

// Applies a timeout to the overall shutdown process. It is expected that within this time, both `Run` and `Shutdown`
// should return successfully.
//
// If not provided, it defaults to [NoTimeout].
func WithShutdownCompletionTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.ShutdownOptions.CompletionTimeout = d
	}
}

// Wraps the provided `Start` and `Stop` functions, making them compatible with the controllers Run-Shutdown model.
//
// Both `Start` and `Stop` are expected to return once their respective step is completed.
//
// If `Start` returns an error, then the error will be passed up to the controller and the controller will transition
// into a failed/shutting down state.
//
// Constraints: [WithStartStop] may only be provided once, and is mutually exclusive with [WithRun].
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

// Applies a call-duration timeout to the `Start` and `Stop` functions provided to [WithStartStop].
//
// These values default to [NoTimeout].
//
// Both zero and negative duration arguments are replaced with [NoTimeout].
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

// Defines a function that can check to see if the component is fully started.
//
// The returns from `CheckReady` are evaluated in the following order:
//
//   - If an error is returned, then the error is passed up to the controller, and the startup is aborted.
//   - If true is returned, then the component is both started and ready, and we can continue.
//   - Otherwise (false and no error), we retry as permitted by [WithCheckReadyMaxAttempts] and an delay from
//     [WithCheckReadyBackoff].
//
// Constraint: This option may only be provided once.
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

// Applies a call-duration timeout to the `CheckReady` function provided to [WithCheckReady].
//
// If not provided, it defaults to [NoTimeout].
func WithCheckReadyCallTimeout(d time.Duration) ComponentOption {
	if d <= 0 {
		d = NoTimeout
	}

	return func(cbs *componentBuildState) {
		cbs.c.CheckReadyOptions.CallTimeout = d
	}
}

// Defines a function that returns how long to back off after each `CheckReady` attempt.
//
// If not provided, it defaults to a function that always returns 0 delay.
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

// Defines the max number of times to attempt a `CheckReady`.
//
// If not provided, it defaults to [math.MaxInt]
func WithCheckReadyMaxAttempts(n int) ComponentOption {
	if n <= 0 {
		n = math.MaxInt
	}

	return func(cbs *componentBuildState) {
		cbs.c.CheckReadyOptions.MaxAttempts = n
	}
}
