package testutil

import (
	"errors"
	"testing"

	"github.com/shoenig/test"
)

func TestWantPanic(t *testing.T) {
	t.Run("panics, doesn't check message", func(t *testing.T) {
		defer WantPanic(t, "")
		panic("any message")
	})

	t.Run("panics, same message", func(t *testing.T) {
		defer WantPanic(t, "moosey")
		panic("moosey")
	})

	t.Run("panics, error type", func(t *testing.T) {
		defer WantPanic(t, "an error type")
		panic(errors.New("an error type"))
	})

	t.Run("mismatch message", func(t *testing.T) {
		mockT := &testing.T{}
		func() {
			defer WantPanic(mockT, "alpha")
			panic("beta")
		}()
		test.True(t, mockT.Failed())
	})

	t.Run("expected panic, but didn't", func(t *testing.T) {
		mockT := &testing.T{}
		func() { defer WantPanic(mockT, "") }()
		test.True(t, mockT.Failed())
	})
}
