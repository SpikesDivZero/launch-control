package testutil

import (
	"testing"

	"github.com/shoenig/test"
)

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
