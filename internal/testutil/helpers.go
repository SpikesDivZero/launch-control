package testutil

import (
	"errors"
	"fmt"
	"testing"
)

func WantPanic(t *testing.T, want string) {
	if e := recover(); e != nil {
		if got := fmt.Sprint(e); got != want {
			t.Errorf("got panic message %q, want %q", got, want)
		}
	} else {
		t.Errorf("got no panic, want one")
	}
}

func WantPanicErrIs(t *testing.T, want error) {
	if e := recover(); e != nil {
		if got, ok := e.(error); !ok {
			t.Errorf("got panic message %q (type %T), expected error type", got, got)
		} else if !errors.Is(got, want) {
			t.Errorf("got panic message %q, want %q", got, want)
		}
	} else {
		t.Errorf("got no panic, want one")
	}
}
