package launch

import (
	"log/slog"

	"github.com/spikesdivzero/launch-control/internal/controller"
)

type ControllerOption func(*controller.Controller)

func WithControllerLogger(log *slog.Logger) ControllerOption {
	if log == nil {
		panic(optionNilArgError{"WithControllerLogger", "log"})
	}

	return func(ic *controller.Controller) {
		ic.Log = log
	}
}
