package component

import (
	"errors"
	"fmt"
)

var (
	// Used in context cancellation. The user-provided Run function has exited.
	ErrRunExited = errors.New("run exited")
)

// When passing errors up to the controller, we wrap them up with sufficient information to diagnose
// the problem. We may get rid of this later on.
type ComponentError struct {
	Name  string
	Stage string
	Err   error
}

func (ce ComponentError) Error() string {
	return fmt.Sprintf("component error: name=%q stage=%q err=%v", ce.Name, ce.Stage, ce.Err)
}

func (ce ComponentError) Unwrap() error {
	return ce.Err
}

func WrapComponentError[T *Component | string](comp T, stage string, err error) error {
	if err == nil {
		return nil
	}

	var name string
	switch v := any(comp).(type) {
	case *Component:
		name = v.Name
	case string:
		name = v
	default:
		// It shouldn't be possible to get here, but we'll panic to be safe
		panic("internal error: unsupported type in WrapComponentError")
	}

	return ComponentError{name, stage, err}
}

// A channel was closed when we were expecting a vlaue.
type PrematureChannelCloseError struct {
	Name string
}

func (pce PrematureChannelCloseError) Error() string {
	return fmt.Sprintf("channel %q closed prematurely", pce.Name)
}

func CheckPrematureChannelClose(chanName string, err error, readOk bool) error {
	if !readOk {
		return PrematureChannelCloseError{chanName}
	}
	return err
}
