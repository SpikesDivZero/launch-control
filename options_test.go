package launchcontrol

import (
	"context"
	"errors"
	"testing"

	"github.com/shoenig/test"
)

func Test_mergeOptions(t *testing.T) {
	// mergeOptions is a glorified wrapper around the following:
	//   1. newDefaultOptions
	//   2. applyOptions w/ isFinal set to true when it's the last item in the list
	// To that end, this test will be fairly simple.

	// This one will fail because the final element must contain a function, which we're missing.
	t.Run("apply returns error", func(t *testing.T) {
		def := newDefaultOptions()
		_, err := mergeOptions([]Options{def, def, def})
		test.ErrorContains(t, err, "options[2]: the final Options must contain a Stop function")
	})

	t.Run("happy path", func(t *testing.T) {
		o, err := mergeOptions([]Options{
			newDefaultOptions(),
			{
				Start: func(ctx context.Context) error { return errors.New("f1") },
				Stop:  func(ctx context.Context) error { return nil },
			},
		})

		test.Nil(t, err)

		test.NotNil(t, o.Start)
		test.ErrorContains(t, o.Start(nil), "f1")
	})
}

func TestOptions_applyFunctionOptions(t *testing.T) {
	t.Run("mapping: start stop check", func(t *testing.T) {
		o := newDefaultOptions()
		test.NoError(t, o.applyFunctionOptions(Options{
			Start:      func(ctx context.Context) error { return errors.New("e1") },
			Stop:       func(ctx context.Context) error { return errors.New("e2") },
			CheckReady: func(ctx context.Context) (bool, error) { return true, errors.New("e3") },
		}, true))

		test.NotNil(t, o.Start)
		test.ErrorContains(t, o.Start(nil), "e1")

		test.NotNil(t, o.Stop)
		test.ErrorContains(t, o.Stop(nil), "e2")

		test.NotNil(t, o.CheckReady)
		b, e := o.CheckReady(nil)
		test.True(t, b)
		test.ErrorContains(t, e, "e3")
	})

	t.Run("mapping: run stop", func(t *testing.T) {
		o := newDefaultOptions()
		test.NoError(t, o.applyFunctionOptions(Options{
			Run:  func(ctx context.Context) error { return errors.New("e1") },
			Stop: func(ctx context.Context) error { return errors.New("e2") },
		}, true))

		test.NotNil(t, o.Run)
		test.ErrorContains(t, o.Run(nil), "e1")

		test.NotNil(t, o.Stop)
		test.ErrorContains(t, o.Stop(nil), "e2")
	})

	t.Run("mostly errors", func(t *testing.T) {
		f1 := func(context.Context) error { return nil }
		f2 := func(context.Context) (bool, error) { return false, nil }

		tests := []struct {
			name    string
			from    Options
			isFinal bool
			wantErr bool
		}{
			// isFinal == false; the presence of any functions are invalid
			{"not final, no funcs", Options{}, false, false},
			{"not final, has run", Options{Run: f1}, false, true},
			{"not final, has start", Options{Start: f1}, false, true},
			{"not final, has check", Options{CheckReady: f2}, false, true},
			{"not final, has stop", Options{Stop: f1}, false, true},

			// happy paths for the functions
			{"final, run+stop", Options{Run: f1, Stop: f1}, true, false},
			{"final, run+stop", Options{Run: f1, Stop: f1, CheckReady: f2}, true, false},
			{"final, start+stop", Options{Start: f1, Stop: f1}, true, false},
			{"final, start+stop", Options{Start: f1, Stop: f1, CheckReady: f2}, true, false},

			// disallowed configurations
			{"final, missing stop", Options{Start: f1}, true, true},
			{"final, missing run/start", Options{Stop: f1}, true, true},
			{"final, both run+start", Options{Run: f1, Start: f1, Stop: f1}, true, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				o := newDefaultOptions()
				gotErr := o.applyFunctionOptions(tt.from, tt.isFinal)

				if tt.wantErr {
					test.Error(t, gotErr)
				} else {
					test.NoError(t, gotErr)

					test.Eq(t, tt.from.Run != nil, o.Run != nil)
					test.Eq(t, tt.from.Start != nil, o.Start != nil)
					test.Eq(t, tt.from.CheckReady != nil, o.CheckReady != nil)
					test.Eq(t, tt.from.Stop != nil, o.Stop != nil)
				}
			})
		}
	})
}
