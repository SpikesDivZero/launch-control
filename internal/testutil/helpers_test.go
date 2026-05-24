package testutil_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/shoenig/test"
	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestWantPanic(t *testing.T) {
	tests := []struct {
		name   string
		arg    string
		f      func()
		wantOk bool
	}{
		{
			"got expected panic",
			"something",
			func() { panic("something") },
			true,
		},
		{
			"wrong panic message",
			"something",
			func() { panic("different") },
			false,
		},
		{
			"did not panic",
			"something",
			func() {},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := &testing.T{}
			(func() {
				defer testutil.WantPanic(mockT, tt.arg)
				tt.f()
			})()
			test.EqOp(t, tt.wantOk, !mockT.Failed())
		})
	}
}

func TestWantPanicErrIs(t *testing.T) {
	testErr1 := errors.New("err1")
	wrapped1 := fmt.Errorf("wrapped: %w", testErr1)

	tests := []struct {
		name   string
		arg    error
		f      func()
		wantOk bool
	}{
		{
			"got expected panic",
			testErr1,
			func() { panic(testErr1) },
			true,
		},
		{
			"got expected panic, wrapped",
			testErr1,
			func() { panic(wrapped1) },
			true,
		},
		{
			"panic with string",
			testErr1,
			func() { panic("wrong type") },
			false,
		},
		{
			"wrong error",
			testErr1,
			func() { panic(errors.New("different")) },
			false,
		},
		{
			"did not panic",
			testErr1,
			func() {},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockT := &testing.T{}
			(func() {
				defer testutil.WantPanicErrIs(mockT, tt.arg)
				tt.f()
			})()
			test.EqOp(t, tt.wantOk, !mockT.Failed())
		})
	}
}
