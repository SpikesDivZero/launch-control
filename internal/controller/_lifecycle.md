# Controller Lifecycle

Mentally, I'm thinking of the controller as a narrowly defined state machine.

There's only a select few states, and they transition in a strictly linear order.

1) New
2) Alive
3) Dying
4) Dead

## State Shortcuts

The following state transitions consist of the "normal" flow of operations:

1) New -> Alive
2) Alive -> Dying
3) Dying -> Dead

The following non-normal state transitions are also acceptable:

1) New -> Dead: when RequestStop is called, and Launch was never called

## New (Initial State)

When a controller is created, and before any services are launched, the controller starts off in the New state.

In this state, it's pretty bare bones, and the internal control loop isn't yet started.

## Alive

Upon the first Launch request, the controller transitions into this state.
It'll remain in this state until such time that something calls RequestStop (for any reason).

In this state, the controller's main responsibility is twofold:

1) Listen for incoming Launch requests, and execute them.
2) Listen for the Shutdown signal.

## Dying

Upon the Shutdown signal being received, the controller transitions into this state.

In this state, the controller's main responsiblity is:

1) Listen for and silently reject incoming Launch requests.
2) Run the graceful shutdown procedure (stopping all components in the reverse order of when they were started).

## Dead

When the Dying state has finished processing all component shutdowns, we transition into this state.

There's nothing more to be done here, and the control coroutine should be terminated.
