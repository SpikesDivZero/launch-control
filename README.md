# Launch Control

The launchcontrol package handles application lifecycle component tracking and shutdown.

Components are started in the order you call Launch, and are shut down in the reverse order. (A component is broadly defined as a blocking function implementing a chunk of your application logic.)

If any of the tracked components return or stop running, the controller considers the component dead, and initiates a shutdown so that a new process can spin up and take our place.

Once all tracked components terminate, Wait unblocks and returns the first non-nil error (presented via either a return or a RequestStop call), or nil if there were no errors.

## Usage

An example usage is available in `example_test.go` (also available via go doc).

## Readiness Checks

Readiness checks are optional and, if missing, default to the component immediately becoming ready.

If a CheckReady function is provided in a Launch, then we'll repeatedly call that function until the component either becomes ready, returns an error, or a shutdown request is received.

Readiness checks support an optional max-attempts, backoff.
