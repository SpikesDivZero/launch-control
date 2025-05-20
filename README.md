# Launch Control

## Overview

The launch package handles application lifecycle component tracking and shutdown.

Components are started in the order you call Launch, and are shut down in the reverse order.
(A component is broadly defined as a blocking function implementing a chunk of your application logic.)

If any of the tracked components return or stop running, the controller considers the component dead,
and initiates a shutdown so that a new process can spin up and take our place.

Once all tracked components terminate, Wait unblocks and returns the first non-nil error
(presented via either a return or a RequestStop call), or nil if there were no errors.

## Readiness Checks

Readiness checks are optional and, if missing, default to the component immediately becoming ready.

If a CheckReady function is provided in a Launch, then we'll repeatedly call that function
until the component either becomes ready, returns an error, or a shutdown request is received.

Readiness checks support an optional max-attempts, backoff, and timeout.

## Timeouts

The use of timeouts is optional, and the default timeout for everything is the package `NoTimeout` constant.

Timeouts are implemented by calling the functions you provide in a separate goroutine.

In the event that the timeout is hit, we cancel the context provided to the call, expecting the function to honor that.
If it still doesn't respond after a short time, the goroutine containing the call is abandoned, and an error is recorded.

Accordingly, the use of timeouts may result in leakage.
We mitigate this by regarding timeouts as errors, triggering the shutdown process.

## Usage

```go
ctrl := launch.NewController(context.TODO())

mgmtServer := newHttpManagementServer() // Internal or mgmt facing service
ctrl.Launch("http:mgmt", launch.WithRun(
    mgmtServer.Serve,
    mgmtServer.Shutdown))

// Since there's no WithCheckReady, this first launch will return as soon as Serve() is started.
//
// Our http:mgmt service exposes a `GET /ready` to let our L7 traffic management know if this service is ready
// to accept traffic.
//
// It starts in the not-ready state, and is controlled via `mgmtServer.SetReadyForTraffic(bool)`

kafkaClient := newKafkaClient()
ctrl.Launch("kafka",
    launch.WithStartStop(
        kafkaClient.ConnectProducer,
        kafkaClient.FlushAndDisconnect),
    launch.WithCheckReady(kafkaClient.CheckFullyConnected))

// The WithCheckReady above means we won't proceed to launch the next services until the kafka client is fully
// connected to all partitions on the topics it uses.

pubServer := newHttpPublicServer() // Client-facing service
ctrl.Launch("http:client", launch.WithRun(
    pubServer.Serve,
    pubServer.Shutdown))

ctrl.Launch("mark-ready", launch.WithStartStop(
    func(ctx context.Context) error { // Start
        mgmtServer.SetReadyForTraffic(true)
        return nil
    },
    func(ctx context.Context) error { // Stop
        mgmtServer.SetReadyForTraffic(false)
        time.Sleep(15 * time.Second) // Allow for the L7 traffic routers to get the update
        return nil
    }))

// We can request a shutdown at any time via request stop.
time.AfterFunc(time.Minute, func() {
    ctrl.RequestStop(nil)
})

// We use `Wait()` to block the main coroutine until the application has fully shut down.
if err := ctrl.Wait(); err != nil {
    fmt.Println("controller exit with an error:", err)
}
```
