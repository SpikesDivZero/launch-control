package testutil

import (
	"testing"

	"github.com/shoenig/test"
)

// Silly bit to provide "coverage" for a stringer branch.
func init() { _ = ChanReadStatus(-1).String() }

func wantFail(t *testing.T, f func(*testing.T)) {
	t.Helper()

	mockT := &testing.T{}
	f(mockT)
	test.True(t, mockT.Failed())
}

func TestChanReadIs(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		ch := make(chan int, 3)
		ChanReadIs(t, ch, ChanReadStatusBlocked, 0)

		ch <- 8
		ch <- 3
		ch <- 12
		ChanReadIs(t, ch, ChanReadStatusOk, 8)
		ChanReadIs(t, ch, ChanReadStatusOk, 3)
		ChanReadIs(t, ch, ChanReadStatusOk, 12)

		ChanReadIs(t, ch, ChanReadStatusBlocked, 0)

		close(ch)
		ChanReadIs(t, ch, ChanReadStatusClosed, 0)
	})

	t.Run("mismatches", func(t *testing.T) {
		ch := make(chan string, 3)
		ch <- "hello"
		ch <- "world"
		ch <- "again"
		close(ch)

		expectFail := func(n string, s ChanReadStatus, v string) {
			t.Run(n, func(t *testing.T) {
				mockT := &testing.T{}
				ChanReadIs(mockT, ch, s, v)
				test.True(t, mockT.Failed())
			})
		}

		expectFail("status", ChanReadStatusBlocked, "hello") // actual Ok+"hello"
		expectFail("value", ChanReadStatusOk, "nope")        // actual Ok+"world"
		expectFail("both", ChanReadStatusClosed, "wrogn")    // actual Ok+"again"
		expectFail("closed", ChanReadStatusOk, "boop")       // actual Closed+""
	})
}

func TestChanReadIsClosed(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "boop"
	close(ch)

	wantFail(t, func(t *testing.T) { ChanReadIsClosed(t, ch) })
	ChanReadIsClosed(t, ch)
}

func TestChanReadIsBlocked(t *testing.T) {
	ch := make(chan string, 1)
	ch <- "boop"
	defer close(ch)

	wantFail(t, func(t *testing.T) { ChanReadIsBlocked(t, ch) })
	ChanReadIsBlocked(t, ch)
}

func TestChanReadIsOk(t *testing.T) {
	ch := make(chan string, 2)
	ch <- "boop"
	ch <- "again"
	close(ch)

	wantFail(t, func(t *testing.T) { ChanReadIsOk(t, ch, "wrong") })
	ChanReadIsOk(t, ch, "again")
}

func TestChanWithCloser(t *testing.T) {
	ch, closeCh := ChanWithCloser[int](1)

	// Works as normal
	ch <- 12
	closeCh()
	test.Eq(t, 12, <-ch)

	// It's actually closed (trying to close it directly results in a panic)
	func() {
		defer WantPanic(t, "close of closed channel")
		close(ch)
	}()

	// We can use our closer it a second time without a panic
	closeCh()
}
