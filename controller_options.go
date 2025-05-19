package launch

import (
	"log/slog"

	"github.com/spikesdivzero/launch-control/internal/controller"
)

type internalController = controller.Controller

type ControllerOption func(*internalController)

func WithControllerLogger(log *slog.Logger) ControllerOption {
	if log == nil {
		panic(optionNilArgError{"WithControllerLogger", "log"})
	}

	return func(ic *internalController) {
		ic.Log = log
	}
}
