package launch

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/spikesdivzero/launch-control/internal/component"
)

// Truthfully, the constant value is a bit less than 50 years, which isn't the same as saying
// no timeout, but... what're the odds you're going to leave something running for that long?
//
// Why 50 years? No reason, except that it's unreasonably large, and will fit within Go's
// time.Time type for a long time to come. (time.maxWall is the year 2157, 132 years from when
// this was first defined)
const NoTimeout time.Duration = 50 * (time.Hour * 24 * 365)

// Options define how a Component is executed, along with any timeouts that should be applied to it.
//
// At least one Options struct passed to Launch must provide a way to run the component.
// This options struct must contain a `Shutdown` function, along with either `Run` or `Start`.
//
// Everything else is optional.
type Options struct {
	// `Run` is a blocking function, that returns when the component has exited (or has started exiting via `Shutdown`).
	//
	// If `Run` returns an error, then the error will be passed up to the controller and the controller will transition
	// into a failed/shutting down state.
	//
	// Constraints:
	//
	//   - `Run` may only be provided once.
	//   - `Shutdown` must be provided in the same `Options` for completeness.
	//   - `Start` is not compatible with `Run` (they represent different run styles)
	Run func(context.Context) error

	// `Start` is a non-blocking function, that returns once the component has started up.
	//
	// If `Start` returns an error, then the error will be passed up to the controller and the controller will
	// transition into a failed/shutting down state.
	//
	// Constraints:
	//
	//   - `Start` may only be provided once.
	//   - `Shutdown` must be provided in the same `Options` for completeness.
	//   - `Run` is not compatible with `Start` (they represent different run styles)
	Start func(context.Context) error

	// Applies a call-duration timeout to the `Start` functions.
	//
	// Defaults to [NoTimeout].
	// Both zero and negative duration arguments are replaced with [NoTimeout].
	StartCallTimeout *time.Duration

	// `Shutdown` is called when it's time to terminate the component. The shutdown process is not considered complete
	// until both `Run` and `Shutdown` have finished, or their corresponding timeouts have completed.
	//
	// If you've also provided `CheckReady`, it's worth noting that `Shutdown` may be called at any point. The
	// `CheckReady` function may or may not have been called. If the `CheckReady` call timed out, then it may still be
	// running in another coroutine.
	//
	// Constraints:
	//
	//   - `Shutdown` may only be provided once.
	//   - `Shutdown` must be provided in the same `Options` as either `Run` or `Start` (but not both)
	Shutdown func(context.Context) error

	// Applies a call-duration timeout to the `Shutdown` function.
	//
	// In the event that both this and `ShutdownCompletionTimeout` are provided, the call timeout will be the lesser
	// of the two durations.
	//
	// If not provided, it defaults to [NoTimeout].
	ShutdownCallTimeout *time.Duration

	// Applies a timeout to the overall shutdown process. It is expected that within this time, both `Run` and
	// `Shutdown` should return successfully.
	//
	// If not provided, it defaults to [NoTimeout].
	ShutdownCompletionTimeout *time.Duration

	// Defines a function that can check to see if the component is fully started.
	//
	// The returns from `CheckReady` are evaluated in the following order:
	//
	//   - If an error is returned, then the error is passed up to the controller, and the startup is aborted.
	//   - If true is returned, then the component is both started and ready, and we can continue.
	//   - Otherwise (false and no error), we retry as permitted by `CheckReadyMaxAttempts` and an delay from
	//     `CheckReadyBackoff`.
	//
	// Constraint: This option may only be provided once.
	CheckReady func(context.Context) (bool, error)

	// Applies a call-duration timeout to the `CheckReady` function provided to `CheckReady`.
	//
	// If not provided, it defaults to [NoTimeout].
	CheckReadyCallTimeout *time.Duration

	// Defines a function that returns how long to back off after each `CheckReady` attempt.
	//
	// If not provided, it defaults to a function that always returns 0 delay.
	CheckReadyBackoff BackoffFunc

	// Defines the max number of times to attempt a `CheckReady`.
	//
	// If not provided, it defaults to [math.MaxInt]
	CheckReadyMaxAttempts *int
}

// Determines the run style for this configuration.
// If the function set in this Options is invalid, it panics.
// If there's no run configuration in this options, returns empty string.
func (co Options) getRunStyle() string {
	hasRun := co.Run != nil
	hasStart := co.Start != nil
	hasShutdown := co.Shutdown != nil

	if !hasRun && !hasStart && !hasShutdown {
		return ""
	}

	if hasRun && hasStart {
		panic("Options has both Run and Start; can only have one run style, not both (invalid configuration)")
	}

	if hasShutdown != (hasRun || hasStart) {
		if hasRun {
			panic("Options has Run, but doesn't have Shutdown (incomplete configuration)")
		} else if hasStart {
			panic("Options has Start, but doesn't have Shutdown (incomplete configuration)")
		} else if hasShutdown {
			panic("Options has Shutdown, but doesn't have one of Run or Start (incomplete configuration)")
		} else {
			panic("internal error: shouldn't be possible")
		}
	}

	if hasRun {
		return "Run"
	} else if hasStart {
		return "Start"
	} else {
		panic("internal error: shouldn't be possible")
	}
}

func (co *Options) applyOptions(from Options) {
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
			panic(fmt.Sprintf("Options: cannot merge multiple run styles: previous %q, this %q", prevStyle, fromStyle))
		}
	}

	if from.Shutdown != nil {
		co.Run = from.Run
		co.Start = from.Start
		co.Shutdown = from.Shutdown
	}

	applyDuration(&co.ShutdownCallTimeout, from.ShutdownCallTimeout)
	applyDuration(&co.ShutdownCompletionTimeout, from.ShutdownCompletionTimeout)

	applyDuration(&co.StartCallTimeout, from.StartCallTimeout)

	if from.CheckReady != nil {
		if co.CheckReady != nil {
			panic("Options: cannot merge multiple CheckReady funcs")
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

func (co *Options) finalize() error {
	// A valid duration is non-nil, positive, and greater than zero.
	applyDefaultDuration := func(dest **time.Duration, src time.Duration) {
		if *dest == nil || **dest <= 0 {
			*dest = &src
		}
	}

	if co.getRunStyle() == "" {
		return errors.New("must provide (Run or Start) and Shutdown")
	}

	applyDefaultDuration(&co.ShutdownCallTimeout, NoTimeout)
	applyDefaultDuration(&co.ShutdownCompletionTimeout, NoTimeout)

	applyDefaultDuration(&co.StartCallTimeout, NoTimeout)

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

func buildComponent(name string, opts ...Options) (*component.Component, error) {
	if name == "" {
		return nil, errors.New("name must not be empty")
	}

	co := Options{}
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
		ssw.ImplStop = co.Shutdown

		ssw.StartTimeout = *co.StartCallTimeout
		ssw.StopTimeout = *co.ShutdownCallTimeout

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
