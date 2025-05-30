package lcerrors

import "errors"

var (
	ErrMonitorExitedWhileStillAlive = errors.New("monitor exiting while the component is still alive")
)

var (
	ErrWaitReadyComponentExited     = errors.New("component exited")
	ErrWaitReadyExceededMaxAttempts = errors.New("did not become ready within MaxAttempts")
	ErrWaitReadyAbortChClosed       = errors.New("abort requested")
)

var (
	ErrShutdownAbandonedNonResponsive = errors.New("failed to respond to both ImplShutdown and ctx cancellation; abandoning it")
)
