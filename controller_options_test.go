package launch

import (
	"log/slog"
	"testing"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/controller"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestWithControllerLogger(t *testing.T) {
	c := controller.New(t.Context())

	log := slog.New(slog.Default().Handler())
	WithControllerLogger(log)(c)
	test.Eq(t, log, c.Log)

	t.Run("panics on nil", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithControllerLogger", "log"}.Error())
		WithControllerLogger(nil)(c)
	})
}
