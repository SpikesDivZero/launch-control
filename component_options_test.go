package launch

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shoenig/test"
	"github.com/shoenig/test/must"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func Test_optionNilArgError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  optionNilArgError
		want string
	}{
		{
			"WithFoo bar",
			optionNilArgError{"WithFoo", "bar"},
			"WithFoo: bar must not be nil",
		},
		{
			"WithJack jill",
			optionNilArgError{"WithJack", "jill"},
			"WithJack: jill must not be nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.Eq(t, tt.want, tt.err.Error())
		})
	}
}

func Test_optionConflictingCallsError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  optionConflictingCallsError
		want []string
	}{
		{
			"two stack",
			optionConflictingCallsError{
				"my error message",
				[][2]string{
					{"WithFoo", "stack for first call"},
					{"WithBar", "stack for second call"},
				},
			},
			[]string{
				"my error message",
				"",
				"Conflicting calls:",
				"",
				"## Call to WithFoo",
				"stack for first call",
				"",
				"## Call to WithBar",
				"stack for second call",
				"",
				"",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.Eq(t, tt.want, strings.Split(tt.err.Error(), "\n"))
		})
	}
}

func Test_buildComponent(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// normally not allowed, but since we're bypassing the With{Option} protections, we can get away with
		// it to verify that our function was called
		negD := -42 * time.Second
		c, err := buildComponent("happy1", func(cbs *componentBuildState) {
			t.Log("invoked")
			cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"a", "b"})
			cbs.c.ShutdownOptions.CallTimeout = negD
		})
		must.NoError(t, err)
		must.NotNil(t, c)

		test.Eq(t, c.Name, "happy1")                    // build is responsible for setting this
		test.Eq(t, c.ShutdownOptions.CallTimeout, negD) // our function was called
	})

	t.Run("missing run style", func(t *testing.T) {
		c, err := buildComponent("missing run", func(cbs *componentBuildState) {})
		test.ErrorContains(t, err, "must provide either WithRun or WithStartStop")
		test.Nil(t, c)
	})

	t.Run("multiple run style calls", func(t *testing.T) {
		sampleStacks := [][2]string{
			{"WithFoo", "foo"},
			{"WithBar", "bar"},
		}
		c, err := buildComponent("multiple run", func(cbs *componentBuildState) {
			cbs.appliedRunCalls = sampleStacks
		})
		test.Eq(t, err, error(optionConflictingCallsError{
			"multiple calls to WithRun/WithStartStop; must be exactly one",
			sampleStacks,
		}))
		test.Nil(t, c)
	})

	t.Run("multiple check ready calls", func(t *testing.T) {
		sampleStacks := [][2]string{
			{"WithJack", "jack"},
			{"WithJill", "jill"},
		}
		c, err := buildComponent("multiple ready", func(cbs *componentBuildState) {
			cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"a", "b"})
			cbs.appliedCheckReadyCalls = sampleStacks
		})
		test.Eq(t, err, error(optionConflictingCallsError{
			"multiple calls to WithCheckReady; must be zero or one",
			sampleStacks,
		}))
		test.Nil(t, c)
	})
}

func TestWithBundledOptions(t *testing.T) {
	testCbs := newComponentBuildState("bundle test")

	seenCalls := []string{}
	makeTestOptFunc := func(name string) ComponentOption {
		return func(cbs *componentBuildState) {
			seenCalls = append(seenCalls, name)
			test.Eq(t, testCbs, cbs)
		}
	}

	bundle := WithBundledOptions(
		makeTestOptFunc("alfa"),
		makeTestOptFunc("bravo"),
		makeTestOptFunc("charlie"),
	)
	bundle(testCbs)

	test.Eq(t, []string{"alfa", "bravo", "charlie"}, seenCalls)
}

func TestWithRun(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cbs := newComponentBuildState("test")
		cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"test", "abc"})
		WithRun(
			func(ctx context.Context) error { return errors.New("test impl run") },
			func(ctx context.Context) error { return errors.New("test impl shutdown") },
		)(cbs)

		must.NotNil(t, cbs.c.ImplRun)
		test.ErrorContains(t, cbs.c.ImplRun(t.Context()), "test impl run")

		must.NotNil(t, cbs.c.ImplShutdown)
		test.ErrorContains(t, cbs.c.ImplShutdown(t.Context()), "test impl shutdown")

		must.Eq(t, 2, len(cbs.appliedRunCalls)) // append, not overwrite
		test.Eq(t, "WithRun", cbs.appliedRunCalls[1][0])
		test.StrContains(t, cbs.appliedRunCalls[1][1], ".TestWithRun")
	})

	t.Run("nil run", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithRun", "run"}.Error())
		WithRun(nil, func(ctx context.Context) error { return nil })
	})

	t.Run("nil shutdown", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithRun", "shutdown"}.Error())
		WithRun(func(ctx context.Context) error { return nil }, nil)
	})
}

func TestWithShutdownCallTimeout(t *testing.T) {
	tests := []struct {
		argD, wantD time.Duration
	}{
		{-12, NoTimeout},
		{0, NoTimeout},
		{3 * time.Minute, 3 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.argD.String(), func(t *testing.T) {
			cbs := newComponentBuildState("test")
			WithShutdownCallTimeout(tt.argD)(cbs)
			test.Eq(t, tt.wantD, cbs.c.ShutdownOptions.CallTimeout)
		})
	}
}

func TestWithShutdownCompletionTimeout(t *testing.T) {
	tests := []struct {
		argD, wantD time.Duration
	}{
		{-12, NoTimeout},
		{0, NoTimeout},
		{3 * time.Minute, 3 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.argD.String(), func(t *testing.T) {
			cbs := newComponentBuildState("test")
			WithShutdownCompletionTimeout(tt.argD)(cbs)
			test.Eq(t, tt.wantD, cbs.c.ShutdownOptions.CompletionTimeout)
		})
	}
}

func TestWithStartStop(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cbs := newComponentBuildState("test")
		cbs.appliedRunCalls = append(cbs.appliedRunCalls, [2]string{"test", "abc"})
		WithStartStop(
			func(ctx context.Context) error { return errors.New("test impl start") },
			func(ctx context.Context) error { return errors.New("test impl stop") },
		)(cbs)

		must.NotNil(t, cbs.ssw.ImplStart)
		test.ErrorContains(t, cbs.ssw.ImplStart(t.Context()), "test impl start")

		must.NotNil(t, cbs.ssw.ImplStop)
		test.ErrorContains(t, cbs.ssw.ImplStop(t.Context()), "test impl stop")

		must.Eq(t, 2, len(cbs.appliedRunCalls)) // append, not overwrite
		test.Eq(t, "WithStartStop", cbs.appliedRunCalls[1][0])
		test.StrContains(t, cbs.appliedRunCalls[1][1], ".TestWithStartStop")
	})

	t.Run("nil start", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithStartStop", "start"}.Error())
		WithStartStop(nil, func(ctx context.Context) error { return nil })
	})

	t.Run("nil stop", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithStartStop", "stop"}.Error())
		WithStartStop(func(ctx context.Context) error { return nil }, nil)
	})
}

func TestWithStartStopCallTimeouts(t *testing.T) {
	type dpair struct{ start, stop time.Duration }
	for _, tt := range []struct {
		name string
		args dpair
		want dpair
	}{
		{
			"ok",
			dpair{time.Second, time.Minute},
			dpair{time.Second, time.Minute},
		},
		{
			"zero start",
			dpair{0, time.Hour},
			dpair{NoTimeout, time.Hour},
		},
		{
			"zero stop",
			dpair{time.Minute, 0},
			dpair{time.Minute, NoTimeout},
		},
		{
			"zero both",
			dpair{0, 0},
			dpair{NoTimeout, NoTimeout},
		},
		{
			"neg start",
			dpair{-1, time.Hour},
			dpair{NoTimeout, time.Hour},
		},
		{
			"neg stop",
			dpair{time.Minute, -1},
			dpair{time.Minute, NoTimeout},
		},
		{
			"neg both",
			dpair{-1, -1},
			dpair{NoTimeout, NoTimeout},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cbs := newComponentBuildState("test")
			WithStartStopCallTimeouts(tt.args.start, tt.args.stop)(cbs)
			test.Eq(t, tt.want.start, cbs.ssw.StartTimeout)
			test.Eq(t, tt.want.stop, cbs.ssw.StopTimeout)
		})
	}
}

func TestWithCheckReady(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cbs := newComponentBuildState("test")
		cbs.appliedCheckReadyCalls = append(cbs.appliedCheckReadyCalls, [2]string{"test", "abc"})
		WithCheckReady(
			func(ctx context.Context) (bool, error) { return true, errors.New("test impl ready") },
		)(cbs)

		must.NotNil(t, cbs.c.ImplCheckReady)
		ok, err := cbs.c.ImplCheckReady(t.Context())
		test.True(t, ok)
		test.ErrorContains(t, err, "test impl ready")

		must.Eq(t, 2, len(cbs.appliedCheckReadyCalls)) // append, not overwrite
		test.Eq(t, "WithCheckReady", cbs.appliedCheckReadyCalls[1][0])
		test.StrContains(t, cbs.appliedCheckReadyCalls[1][1], ".TestWithCheckReady")
	})

	t.Run("nil check", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithCheckReady", "checkReady"}.Error())
		WithCheckReady(nil)
	})
}

func TestWithCheckReadyCallTimeout(t *testing.T) {
	tests := []struct {
		argD, wantD time.Duration
	}{
		{-12, NoTimeout},
		{0, NoTimeout},
		{3 * time.Minute, 3 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.argD.String(), func(t *testing.T) {
			cbs := newComponentBuildState("test")
			WithCheckReadyCallTimeout(tt.argD)(cbs)
			test.Eq(t, tt.wantD, cbs.c.CheckReadyOptions.CallTimeout)
		})
	}
}

func TestWithCheckReadyBackoff(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		cbs := newComponentBuildState("test")
		WithCheckReadyBackoff(
			func() time.Duration { return 43 * time.Second },
		)(cbs)
		test.Eq(t, 43*time.Second, cbs.c.CheckReadyOptions.Backoff())
	})

	t.Run("nil backoff", func(t *testing.T) {
		defer testutil.WantPanic(t, optionNilArgError{"WithCheckReadyBackoff", "backoff"}.Error())
		WithCheckReadyBackoff(nil)
	})
}

func TestWithCheckReadyMaxAttempts(t *testing.T) {
	tests := []struct {
		arg, want int
	}{
		{-12, math.MaxInt},
		{0, math.MaxInt},
		{43, 43},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.arg), func(t *testing.T) {
			cbs := newComponentBuildState("test")
			WithCheckReadyMaxAttempts(tt.arg)(cbs)
			test.Eq(t, tt.want, cbs.c.CheckReadyOptions.MaxAttempts)
		})
	}
}
