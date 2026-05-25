package testutil

import (
	"sync"
	"testing"

	"github.com/shoenig/test"
)

func Zero[T any]() T {
	var zeroT T
	return zeroT
}

// Returns a channel, along with a closer for that channel.
// The close function is wrapped with [sync.OnceFunc] so make it safe to call multiple times.
func ChanWithCloser[T any](cap int) (chan T, func()) {
	ch := make(chan T, cap)
	return ch, sync.OnceFunc(func() { close(ch) })
}

//go:generate go tool stringer -type=ChanReadStatus -trimprefix ChanReadStatus
type ChanReadStatus int

const (
	// Successful read from the channel.
	// The channel may or may not be closed.
	ChanReadStatusOk ChanReadStatus = iota + 1

	// Channel is open, with no pending messages.
	ChanReadStatusBlocked

	// The channel is closed and no messages remain.
	ChanReadStatusClosed
)

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

func ChanReadIs[T any](t *testing.T, ch <-chan T, expStatus ChanReadStatus, expValue T, settings ...test.Setting) T {
	t.Helper()

	value, status := MaybeReadChan(ch)

	test.Eq(t, expStatus, status, settings...)
	test.Eq(t, expValue, value, settings...)

	return value
}

func ChanReadIsClosed[T any](t *testing.T, ch <-chan T, settings ...test.Setting) {
	t.Helper()
	ChanReadIs(t, ch, ChanReadStatusClosed, Zero[T](), settings...)
}

func ChanReadIsBlocked[T any](t *testing.T, ch <-chan T, settings ...test.Setting) {
	t.Helper()
	ChanReadIs(t, ch, ChanReadStatusBlocked, Zero[T](), settings...)
}

func ChanReadIsOk[T any](t *testing.T, ch <-chan T, wantValue T, settings ...test.Setting) T {
	t.Helper()
	return ChanReadIs(t, ch, ChanReadStatusOk, wantValue, settings...)
}
