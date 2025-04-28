package testutil

import (
	"fmt"
	"testing"
)

// defer WantPanic(t, "wanted reason")
//
// If want is empty string, the specific cause message is not checked.
func WantPanic(t *testing.T, want string) {
	if e := recover(); e != nil {
		if want != "" {
			if got := fmt.Sprint(e); got != want {
				t.Errorf("got panic message %q, want %q", got, want)
			}
		}
	} else {
		t.Errorf("got no panic, want one")
	}
}
