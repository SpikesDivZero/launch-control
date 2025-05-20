package component

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
)

func newTestingComponent(*testing.T) *Component {
	c := New(fmt.Sprintf("testing-comp-%d", rand.Int32()))

	c.ImplRun = func(ctx context.Context) error {
		panic("TestingComponent.ImplRun not defined, but used in test")
	}
	c.ImplShutdown = func(ctx context.Context) error {
		panic("TestingComponent.ImplShutdown not defined, but used in test")
	}
	c.ImplCheckReady = func(ctx context.Context) (bool, error) {
		panic("TestingComponent.ImplCheckReady not defined, but used in test")
	}
	c.CheckReadyOptions.Backoff = func() time.Duration {
		panic("TestingComponent.CheckReadyOptions.Backoff not defined, but used in test")
	}

	c.logError = func(stage string, err error) {
		panic("TestingComponent.logError not defined but used in test")
	}
	c.notifyOnExited = func(err error) {
		panic("TestingComponent.notifyOnExited not defined but used in test")
	}

	return c
}

func TestComponent_ConnectController(t *testing.T) {
	c := Component{}

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

	c.ConnectController(testLogError, testNotify, 242*time.Millisecond)

	must.NotNil(t, c.notifyOnExited)
	c.logError("in-test", testErr)
	test.True(t, calledTestLogError)

	c.notifyOnExited(testErr)
	test.True(t, calledTestNotify)

	test.Eq(t, 242*time.Millisecond, c.asyncGracePeriod)
}
