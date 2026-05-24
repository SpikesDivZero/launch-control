package component

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func newTestingSSW(*testing.T) *StartStopWrapper {
	return &StartStopWrapper{
		Comp:      &Component{Name: "testing"},
		ImplStart: func(context.Context) error { panic("test did not set SSW.ImplStart") },
		ImplStop:  func(context.Context) error { panic("test did not set SSW.ImplStop") },
	}
}

func TestStartStopWrapper_Run(t *testing.T) {
	// The first two subtests ("mock control" and "once") cover the case where requestStopCh is not created.
	// The third subtest ("normal flows") makes for a table test to cover the rest of the flow paths.

	t.Run("mock control", func(t *testing.T) {
		ssw := newTestingSSW(t)
		testErr := errors.New("mock")

		ssw.TestControl.MockRun = func(ctx context.Context) error {
			test.EqOp(t, t.Context(), ctx)
			return testErr
		}

		test.ErrorIs(t, ssw.Run(t.Context()), testErr)
	})

	t.Run("once", func(t *testing.T) {
		defer testutil.WantPanic(t, "SSW.Run called more than once?")

		ssw := newTestingSSW(t)
		ssw.requestStopCh = make(chan struct{})
		_ = ssw.Run(t.Context())
	})

	t.Run("normal flows", func(t *testing.T) {
		testErr1 := errors.New("test1")
		testSSWName := newTestingSSW(t).Comp.Name

		type testState struct {
			StartCalled, StopCalled bool

			RunReturned bool
			RunError    error
		}
		type mockReturns struct {
			Name        string // for use in setup
			Start, Stop error
		}
		for _, tt := range []struct {
			name           string
			setup          func(*mockReturns)
			wantAfterStart testState
			wantAfterStop  testState
		}{
			{
				"no errors",
				func(mr *mockReturns) { /* both funcs return nil error */ },
				testState{StartCalled: true},
				testState{StartCalled: true, StopCalled: true, RunReturned: true},
			},
			{
				"start returns error",
				func(mr *mockReturns) { mr.Start = testErr1 },
				testState{StartCalled: true, RunReturned: true,
					RunError: WrapComponentError(testSSWName, "Start", testErr1)},
				testState{StartCalled: true, RunReturned: true,
					RunError: WrapComponentError(testSSWName, "Start", testErr1)},
			},
			{
				"stop returns error",
				func(mr *mockReturns) { mr.Stop = testErr1 },
				testState{StartCalled: true},
				testState{StartCalled: true, StopCalled: true, RunReturned: true,
					RunError: WrapComponentError(testSSWName, "Stop", testErr1)},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ssw := newTestingSSW(t)

					state := testState{}
					mockctl := mockReturns{Name: ssw.Comp.Name}

					ssw.ImplStart = func(ctx context.Context) error {
						state.StartCalled = true
						test.EqOp(t, t.Context(), ctx)
						return mockctl.Start
					}
					ssw.ImplStop = func(ctx context.Context) error {
						state.StopCalled = true
						test.EqOp(t, t.Context(), ctx)
						return mockctl.Stop
					}

					tt.setup(&mockctl)

					must.Nil(t, ssw.requestStopCh)

					// Start Run up.
					go func() {
						state.RunError = ssw.Run(t.Context())
						state.RunReturned = true
					}()
					synctest.Wait()

					must.NotNil(t, ssw.requestStopCh)
					test.Eq(t, tt.wantAfterStart, state)

					// Simulate stop having been called.
					close(ssw.requestStopCh)
					synctest.Wait()

					test.Eq(t, tt.wantAfterStop, state)
				})
			})
		}
	})
}

func TestStartStopWrapper_Stop(t *testing.T) {
	t.Run("mock control", func(t *testing.T) {
		ssw := newTestingSSW(t)
		testErr := errors.New("test")

		ssw.TestControl.MockStop = func(ctx context.Context) error {
			test.EqOp(t, t.Context(), ctx)
			return testErr
		}

		test.ErrorIs(t, testErr, ssw.Stop(t.Context()))
	})

	t.Run("not started", func(t *testing.T) {
		ssw := newTestingSSW(t)
		test.ErrorContains(t, ssw.Stop(t.Context()),
			"StartStopWrapper: Stop called, but Start was never invoked")
	})

	t.Run("once", func(t *testing.T) {
		defer testutil.WantPanic(t, "SSW.Stop called more than once?")

		ssw := newTestingSSW(t)
		ssw.requestStopCh = make(chan struct{})
		close(ssw.requestStopCh)
		_ = ssw.Stop(t.Context())
	})

	t.Run("happy path", func(t *testing.T) {
		ssw := newTestingSSW(t)
		ssw.requestStopCh = make(chan struct{})
		test.NoError(t, ssw.Stop(t.Context()))
		select {
		case <-ssw.requestStopCh:
		default:
			t.Error("Stop did not close requestStopCh")
		}
	})
}
