package lcerrors

import (
	"errors"
	"fmt"
)

type ComponentError struct {
	Name  string
	Stage string
	Err   error
}

func (ce ComponentError) Error() string {
	return fmt.Sprintf("component %v %v: %v", ce.Name, ce.Stage, ce.Err)
}

func (ce ComponentError) Unwrap() error { return ce.Err }

func (ce ComponentError) Is(target error) bool { return errors.Is(ce.Err, target) }
