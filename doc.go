/*
Package launchcontrol (launch control) provides functionality for managing the lifecycle of a service.

Often, services consist of several components that have a few key critera:

  - They need to be started on one order (ex: A, B, C)
  - They need to be stopped in the reverse order (ex: C, B, A)
  - If any component exits early, for any reason, then the entire service should shut down.
  - We may need to wait for a component to become ready before starting the next one.

The goal of this package is to abstract away some of those concerns.
*/
package launchcontrol
