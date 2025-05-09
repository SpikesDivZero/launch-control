package component

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
)

// Same as in top-level package, but copied here to avoid import
const NoTimeout time.Duration = 50 * (time.Hour * 24 * 365)

func newTestingComponent(*testing.T) *Component {
	return &Component{
		Name: fmt.Sprintf("testing-comp-%d", rand.Int32()),

		ImplRun: func(ctx context.Context) error {
			panic("TestingComponent.ImplRun not defined, but used in test")
		},

		ImplShutdown: func(ctx context.Context) error {
			panic("TestingComponent.ImplShutdown not defined, but used in test")
		},
		ShutdownOptions: ShutdownOptions{
			CallTimeout:       NoTimeout,
			CompletionTimeout: NoTimeout,
		},

		ImplCheckReady: func(ctx context.Context) (bool, error) {
			panic("TestingComponent.ImplCheckReady not defined, but used in test")
		},
		CheckReadyOptions: CheckReadyOptions{
			CallTimeout: NoTimeout,
			Backoff: func() time.Duration {
				panic("TestingComponent.CheckReadyOptions.Backoff not defined, but used in test")
			},
			MaxAttempts: math.MaxInt,
		},

		log: slog.New(slog.DiscardHandler),
		logError: func(stage string, err error) {
			panic("TestingComponent.logError not defined but used in test")
		},
		notifyOnExited: func(err error) {
			panic("TestingComponent.notifyOnExited not defined but used in test")
		},
	}
}

func TestComponent_ConnectController(t *testing.T) {
	c := Component{}

	testLog := slog.New(slog.DiscardHandler)

	testErr := errors.New("fancy")

	calledTestLogError := false
	testLogError := func(stage string, err error) {
		calledTestLogError = true
		test.Eq(t, "in-test", stage)
		test.ErrorIs(t, err, testErr)
	}

	calledTestNotify := false
	testNotify := func(err error) {
		calledTestNotify = true
		test.ErrorIs(t, err, testErr)
	}

	c.ConnectController(testLog, testLogError, testNotify)

	test.EqOp(t, testLog, c.log)

	must.NotNil(t, c.notifyOnExited)
	c.logError("in-test", testErr)
	test.True(t, calledTestLogError)
	c.notifyOnExited(testErr)
	test.True(t, calledTestNotify)
}
