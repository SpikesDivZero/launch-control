package controller

import (
	"context"
	"log/slog"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/component"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestController_worker_main(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// This ends up being a bit of an end-to-end test, but that's OK with me.
		c := NewController(t.Context())

		exitedCh := make(chan struct{})
		go func() {
			defer close(exitedCh)
			c.state = runStateAlive
			c.worker_main()
		}()
		synctest.Wait()

		// In the Alive state.
		test.EqOp(t, runStateAlive, c.state)
		testutil.ChanReadIsBlocked(t, exitedCh)
		testutil.ChanReadIsBlocked(t, c.deadCh)

		// Launch a job
		doneCh := make(chan struct{})
		stubStopGate := make(chan struct{}) // close to allow ImplStop to finish
		c.requestLaunchCh <- launchRequest{
			Name: "testing",
			Comp: &component.StubComponent{
				ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {},
				ImplStart:    func(ctx context.Context) {},
				ImplStop:     func() { <-stubStopGate },
			},
			DoneCh: doneCh,
		}
		<-doneCh

		// Transition into the Dying state.
		close(c.requestStopCh)
		synctest.Wait()

		// Our stub component is currently blocked in ImplStop, keeping us in the dying state

		test.EqOp(t, runStateDying, c.state)
		testutil.ChanReadIsBlocked(t, exitedCh)
		testutil.ChanReadIsBlocked(t, c.deadCh)

		// While we're in the dying state, we should discard launch requests
		// No component => will panic if we try to access it instead of just silently discarding.
		doneCh = make(chan struct{})
		c.requestLaunchCh <- launchRequest{DoneCh: doneCh}
		<-doneCh

		// Okay, let's allow us to progress out of the dying state.
		close(stubStopGate)
		synctest.Wait()

		// We should now be fully dead, with the exception that the launch request discarder is taking care
		// of any potential stragglers.
		test.EqOp(t, runStateDead, c.state)
		testutil.ChanReadIsClosed(t, exitedCh)
		testutil.ChanReadIsClosed(t, c.deadCh)

		// Verify the instant discard behavior; same as above
		doneCh = make(chan struct{})
		c.requestLaunchCh <- launchRequest{DoneCh: doneCh}
		<-doneCh

		// Fantastic. Last thing: advance time far enough that the discarder exits (5s of inactivity)
		time.Sleep(10 * time.Second)
	})
}

func TestController_worker_changeState(t *testing.T) {
	// The first change works
	c := NewController(t.Context())
	c.worker_changeState(runStateNew, runStateAlive)

	// This second change will panic, since the current state doesn't match
	defer testutil.WantPanic(t, "Controller worker state mismatch; got Alive != expected New")
	c.worker_changeState(runStateNew, runStateAlive)
}

func TestController_worker_alive(t *testing.T) {
	c := NewController(t.Context())

	starts := []int{}
	doStart := func(i int) {
		req := launchRequest{
			Name: "testing",
			Comp: &component.StubComponent{
				ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {},
				ImplStart: func(ctx context.Context) {
					starts = append(starts, i)
				},
			},
			DoneCh: make(chan struct{}),
		}
		c.requestLaunchCh <- req
		<-req.DoneCh
	}

	workerExitedCh := make(chan struct{})
	go func() {
		defer close(workerExitedCh)
		c.worker_alive()
	}()

	for i := range 3 {
		doStart(i)
	}
	close(c.requestStopCh)
	test.SliceEqOp(t, []int{0, 1, 2}, starts)

	select {
	case c.requestLaunchCh <- launchRequest{}:
		t.Error("shouldn't be able to write into requestLaunchCh after stop requested")
	default:
	}
}

func TestController_worker_doLaunch(t *testing.T) {
	makeReq := func() (*component.StubComponent, launchRequest) {
		comp := &component.StubComponent{}
		return comp, launchRequest{
			Name:   "testing",
			Comp:   comp,
			DoneCh: make(chan struct{}),
		}
	}

	t.Run("happy path", func(t *testing.T) {
		c := NewController(t.Context())
		comp, req := makeReq()

		var registerCalled, startCalled bool
		comp.ImplRegister = func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {
			registerCalled = true
		}
		comp.ImplStart = func(ctx context.Context) {
			startCalled = true
		}

		c.worker_doLaunch(req)
		testutil.ChanReadIsClosed(t, req.DoneCh)
		test.True(t, registerCalled)
		test.True(t, startCalled)
	})

	t.Run("stop request has priority", func(t *testing.T) {
		c := NewController(t.Context())
		_, req := makeReq()

		// No Stub impl funcs provided, so if it attempts to register or launch the component, it'll panic.

		close(c.requestStopCh)

		c.worker_doLaunch(req)
		testutil.ChanReadIsClosed(t, req.DoneCh)
	})

	t.Run("ctx cancel to interrupt startup", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c := NewController(t.Context())
			comp, req := makeReq()

			comp.ImplRegister = func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {}

			startEnteredCh := make(chan struct{})
			comp.ImplStart = func(ctx context.Context) {
				close(startEnteredCh)
				<-ctx.Done() // close(requestStopCh) should trigger context cancel
			}

			callReturnedCh := make(chan struct{})
			go func() {
				defer close(callReturnedCh)
				c.worker_doLaunch(req)
			}()
			synctest.Wait()

			// We entered start, and our doLaunch call is still blocked
			testutil.ChanReadIsClosed(t, startEnteredCh)
			testutil.ChanReadIsBlocked(t, callReturnedCh)

			// We request a stop
			close(c.requestStopCh)
			synctest.Wait()

			// doLaunch propegated the requestStopCh closure into a context cancellation.
			testutil.ChanReadIsClosed(t, callReturnedCh)
		})
	})
}

func TestController_worker_dying(t *testing.T) {
	c := NewController(t.Context())

	stopOrder := []int{}
	for i := range 3 {
		c.registerComponent(strconv.Itoa(i), &component.StubComponent{
			ImplRegister: func(ctx context.Context, l *slog.Logger, cc component.ControllerCallbacks) {},
			ImplStop: func() {
				stopOrder = append(stopOrder, i)
			},
		})
	}

	c.worker_dying()
	test.SliceEqOp(t, []int{2, 1, 0}, stopOrder) // Stopped in reverse order

	// The discarder should be running.
	doneCh := make(chan struct{})
	c.requestLaunchCh <- launchRequest{DoneCh: doneCh}
	<-doneCh
}

func TestController_worker_discardLaunchRequests(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c := NewController(t.Context())

		sendReq := func() {
			doneCh := make(chan struct{})
			c.requestLaunchCh <- launchRequest{DoneCh: doneCh}
			<-doneCh
		}

		timingCh := make(chan time.Duration)
		go func() {
			t0 := time.Now()
			c.worker_discardLaunchRequests()
			timingCh <- time.Since(t0)
		}()

		for range 3 {
			time.Sleep(time.Second)
			sendReq()
		}

		// discardLaunchRequests should return 5 seconds after the last request it voids.
		test.Eq(t, 8*time.Second, <-timingCh)
	})
}
