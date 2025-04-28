package launch

import (
	"errors"

	"github.com/spikesdivzero/launch-control/internal/component"
)

type componentBuildState struct {
	c *component.Component
}

func buildComponent(name string, opts ...ComponentOption) (*component.Component, error) {
	cbs := componentBuildState{
		c: &component.Component{
			Name: name,
			// TODO: add in all defaults
		},
	}

	for _, opt := range opts {
		opt(cbs)
	}

	// TODO: validate that exactly one of WithRun or WithStartStop was provided

	return cbs.c, errors.New("NYI: buildComponent")
}

type ComponentOption func(componentBuildState)

func WithBundledOptions(opts ...ComponentOption) ComponentOption {
	return func(cbs componentBuildState) {
		for _, opt := range opts {
			opt(cbs)
		}
	}
}
