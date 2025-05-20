package component

import (
	"context"
	"math"
	"time"
)

// Same as in top-level package, but copied here to avoid import
const NoTimeout time.Duration = 50 * (time.Hour * 24 * 365)

const defaultAsyncGracePeriod = 100 * time.Millisecond

type ShutdownOptions struct {
	CallTimeout       time.Duration
	CompletionTimeout time.Duration
}

type CheckReadyOptions struct {
	CallTimeout time.Duration
	Backoff     func() time.Duration
	MaxAttempts int
}

type Component struct {
	Name string

	ImplRun func(context.Context) error

	ImplShutdown    func(context.Context) error
	ShutdownOptions ShutdownOptions

	ImplCheckReady    func(context.Context) (bool, error)
	CheckReadyOptions CheckReadyOptions

	// Values provided by by [ConnectController]
	logError         func(stage string, err error)
	notifyOnExited   func(error)
	asyncGracePeriod time.Duration

	// Lifecycle-related state, created in [Start]
	runCtxCancel context.CancelFunc
	doneCh       <-chan struct{}
}

func New(name string) *Component {
	return &Component{
		Name: name,

		ShutdownOptions: ShutdownOptions{
			CallTimeout:       NoTimeout,
			CompletionTimeout: NoTimeout,
		},
		CheckReadyOptions: CheckReadyOptions{
			CallTimeout: NoTimeout,
			Backoff:     func() time.Duration { return 0 },
			MaxAttempts: math.MaxInt,
		},

		asyncGracePeriod: defaultAsyncGracePeriod,
	}
}

func (c *Component) ConnectController(
	logError func(stage string, err error),
	notifyOnExited func(error),
	asyncGracePeriod time.Duration,
) {
	c.logError = logError
	c.notifyOnExited = notifyOnExited
	c.asyncGracePeriod = asyncGracePeriod
}
