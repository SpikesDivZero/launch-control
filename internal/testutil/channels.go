package testutil

import (
	"reflect"
	"testing"
)

//go:generate go tool stringer -type ChanReadStatus -trimprefix ChanReadStatus
type ChanReadStatus int

const (
	ChanReadStatusClosed ChanReadStatus = iota + 1
	ChanReadStatusBlocked
	ChanReadStatusOk // got a value
)

func Zero[T any]() T {
	var zeroT T
	return zeroT
}

func MaybeReadChan[T any](ch <-chan T) (T, ChanReadStatus) {
	select {
	case v, ok := <-ch:
		if ok {
			return v, ChanReadStatusOk
		} else {
			return v, ChanReadStatusClosed // v is zero value, by spec
		}
	default:
		return Zero[T](), ChanReadStatusBlocked
	}
}

// This test helper, admittedly, feels a bit silly, but in several areas, I specifically want to
// test that a channel is either blocked or closed WITHOUT allowing synctest to advance time.
// As demonstrated in MaybeReadChan, that's kind of verbose.
//
// Wrapping it up like this allows me to check both the status and values in one call, which
// hopefully helps keep it clean and tidy. In my prior sketch, I found testing both values
// inside a synctest to be a bit more verbose than I'd prefer, reducing the clarity.
//
// Maybe this'll be an improvment, maybe it won't.
func ChanReadIs[T any](t *testing.T, ch <-chan T, wantStatus ChanReadStatus, wantValue T) {
	t.Helper()

	value, status := MaybeReadChan(ch)
	if status != wantStatus {
		t.Errorf("ChanReadIs status %v, expected %v", status, wantStatus)
	}
	if !reflect.DeepEqual(value, wantValue) {
		t.Errorf(
			"ChanReadIs value mismatch:\n"+
				"\tGot:  %#v\n"+
				"\tWant: %#v",
			value, wantValue)
	}
}
