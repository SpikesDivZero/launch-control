package launch

import (
	"testing"
	"time"

	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestConstBackoff(t *testing.T) {
	for _, delay := range []time.Duration{
		3 * time.Second,
		4 * time.Minute,
	} {
		t.Run(delay.String(), func(t *testing.T) {
			bf := ConstBackoff(delay)
			for range 6 {
				if got := bf(); got != delay {
					t.Errorf("ConstBackoff() = %v, want %v", got, delay)
				}
			}
		})
	}
}

func TestExpBackoff(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		type args struct {
			minDelay time.Duration
			maxDelay time.Duration
			exp      float64
		}
		tests := []struct {
			name string
			args args
			want []time.Duration
		}{
			{
				"doubled",
				args{time.Second, time.Minute, 2.0},
				[]time.Duration{
					1 * time.Second,
					2 * time.Second,
					4 * time.Second,
					8 * time.Second,
					16 * time.Second,
					32 * time.Second,
					60 * time.Second, // maxDuration hit
					60 * time.Second, // maxDuration hit
				},
			},
			{
				"tripled, 3s",
				args{4 * time.Second, 120 * time.Second, 3.0},
				[]time.Duration{
					4 * time.Second,
					12 * time.Second,
					36 * time.Second,
					108 * time.Second,
					120 * time.Second, // maxDuration hit
					120 * time.Second, // maxDuration hit
				},
			},
			{
				"50pct, 8s",
				args{8 * time.Second, 48 * time.Second, 1.5},
				[]time.Duration{
					8 * time.Second,
					12 * time.Second,
					18 * time.Second,
					27 * time.Second,
					40*time.Second + time.Second/2, // 40.5
					48 * time.Second,               // maxDuration hit
					48 * time.Second,               // maxDuration hit
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bf := ExpBackoff(tt.args.minDelay, tt.args.maxDelay, tt.args.exp, false)

				for i, want := range tt.want {
					if got := bf(); got != want {
						t.Errorf("ExpBackoff() call %v = %v, want %v", i, got, want)
					}
				}
			})
		}
	})

	// Can't particularly test jitter behaviors, so we'll just approximate it by calling it a bunch of times and seeing that it
	// always returns in range. This is far from perfect, but works well enough for now as a first pass.
	t.Run("jitter", func(t *testing.T) {
		minD, maxD := time.Second, 5*time.Second
		bf := ExpBackoff(minD, maxD, 1.5, true)
		for i := range 50 {
			if got := bf(); got < minD || got > maxD {
				t.Errorf("ExpBackoff() call %v = %v, wanted in range %v to %v", i, got, minD, maxD)
			}
		}
	})

	t.Run("zero min", func(t *testing.T) {
		defer testutil.WantPanic(t, "")
		ExpBackoff(0, time.Second, 2.0, true)
	})

	t.Run("zero max", func(t *testing.T) {
		defer testutil.WantPanic(t, "")
		ExpBackoff(time.Second, 0, 2.0, true)
	})

	t.Run("min gt max", func(t *testing.T) {
		defer testutil.WantPanic(t, "")
		ExpBackoff(time.Minute, time.Second, 2.0, true)
	})
}
