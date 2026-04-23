package component

import (
	"context"
)

// For now, a thin wrapper around the stuff we import from the public interface Options.
type Component struct {
	Name string

	ImplRun        func(context.Context) error
	ImplCheckReady func(context.Context) (bool, error)
	ImplStop       func(context.Context) error

	// We won't use this directly, but we keep a pointer to it here for other test inspection.
	SSW *StartStopWrapper
}
