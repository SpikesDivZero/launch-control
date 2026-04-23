package launchcontrol

import (
	"context"
	"errors"
	"fmt"
)

// Options defines how you'd like this package to run your component.
//
// Most values are optional and, when omitted, will be defined with a sane default as described.
//
// A [Controller.Launch] call allows you to provide multiple Options to a component.
// If more than one Options is provided to a Launch call, then the first ones are treated as general
// settings, and the final one is treated as "what you want to run".
//
// Accordingly, the final Options provided to a Launch call must provide:
//
//   - A Run or Start function (but not both)
//   - A Stop function
//
// It is a fatal error to provide these functions inside any non-final Options.
type Options struct {
	// Run defines a blocking startup style.
	//
	// When the component is launched, the Run function will be invoked in a separate coroutine.
	//
	// When this function returns, Controller.RequestStop is called with the returned error value
	// as the reason for the stop request. A non-nil error may be wrapped.
	//
	// A Component must provide one of Run or Start (but not both) in it's final Options.
	Run func(context.Context) error

	// Start defines a non-blocking startup style.
	//
	// When the component is launched, the Start function will be invoked, and the controller's
	// startup process will block until this function returns.
	//
	// If Start returns a non-nil error, then Controller.RequestStop will be called with the returned
	// error value. The error may be wrapped.
	//
	// A Component must provide one of Run or Start (but not both) in it's final Options.
	Start func(context.Context) error

	// CheckReady defines how the controller should check to see if your component has finished starting up.
	//
	// If CheckReady is not provided, then we assume the component is ready as soon as it's started.
	//
	// Component readiness checks happen after either:
	//
	//   - Run is invoked
	//   - Start returns
	//
	// CheckReady's return values are evaluated as follows:
	//
	//   - non-nil error: Controller.RequestStop will be called with the returned error value.
	//     The error may be wrapped.
	//   - true: The component has fully started up, and the controller may proceed to the next step.
	//   - false: The component is not ready. We'll retry the check.
	CheckReady func(context.Context) (bool, error)

	// Stop defines how the controller should terminate your component.
	//
	// In the event that you're using Run, this should signal to the blocking Run function that it's
	// time for it to close down and return. The shutdown process will block until BOTH Run and Stop
	// return.
	//
	// In the event that you're using Start, then the shutdown process will block until Stop returns.
	//
	// Any non-nil error will be logged and returned in Controller.AllErrors. The error may be wrapped.
	Stop func(context.Context) error
}

func newDefaultOptions() Options {
	return Options{}
}

func mergeOptions(opts []Options) (Options, error) {
	ret := newDefaultOptions()
	for i, o := range opts {
		isFinal := len(opts) == i+1
		if err := ret.applyOptions(o, isFinal); err != nil {
			return ret, fmt.Errorf("options[%d]: %w", i, err)
		}
	}
	return ret, nil
}

func (o *Options) applyFunctionOptions(from Options, isFinal bool) error {
	hasRun, hasStart := from.Run != nil, from.Start != nil
	hasCheckReady := from.CheckReady != nil
	hasStop := from.Stop != nil

	// Non-final Options must contain no functions (in which case, there's little for us to do)
	if !isFinal {
		if hasAny := hasRun || hasStart || hasCheckReady || hasStop; hasAny {
			return errors.New("only the final Options may contain implementation functions")
		} else {
			return nil
		}
	}

	// The final Options must contain the correct function configuration

	if !hasStop {
		return errors.New("the final Options must contain a Stop function")
	}

	if hasRun && hasStart {
		return errors.New("the final Options must not contain both of Run and Start")
	} else if !hasRun && !hasStart {
		return errors.New("the final Options must contain either Run or Start")
	}

	o.Run = from.Run
	o.Start = from.Start
	o.CheckReady = from.CheckReady
	o.Stop = from.Stop

	return nil
}

func (o *Options) applyOptions(from Options, isFinal bool) error {
	if err := o.applyFunctionOptions(from, isFinal); err != nil {
		return err
	}

	return nil
}

func (o *Options) validate() error {
	// The run style was validated over in applyFunctionOptions

	// Here, we should validate all other remaining values, and assign them defaults as necessary.

	return errors.New("NYI: Options.validate")
}
