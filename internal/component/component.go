package component

import "context"

// For now, a thin wrapper around the stuff we import from the public interface Options.
type Component struct {
	Name           string
	ImplRun        func(context.Context) error
	ImplStart      func(context.Context) error
	ImplCheckReady func(context.Context) (bool, error)
	ImplStop       func(context.Context) error
}
