package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
)

// We're pretending that we're connecting to a data store like Kafka or Mongo, where
// the client connection/open call doesn't fully establish a connection. These drivers
// may instead start up an async connection process.
//
// So this exists to show off how we can use CheckReady to delay startup of the next
// component in the stack until after this component has fully started up.
type DataConnector struct {
	Log *slog.Logger

	ready bool
}

func (c *DataConnector) Connect(ctx context.Context) error {
	// Actually doing the db driver setup call is near instant, so not much to mock
	// here. Instead, we start a timer to eventually mark ourselves as ready.
	time.AfterFunc(3*time.Second, func() {
		c.ready = true
	})
	return nil
}

func (c *DataConnector) Disconnect(context.Context) error {
	time.Sleep(time.Second) // it takes a bit to flush any pending batch writes
	return nil
}

func (c *DataConnector) CheckReady(context.Context) (bool, error) {
	time.Sleep(50 + time.Millisecond*time.Duration(rand.IntN(100))) // 50-150ms
	return c.ready, nil
}
