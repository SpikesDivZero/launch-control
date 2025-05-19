package lcerrors

import (
	"context"
	"fmt"
)

type ContextTimeoutError struct {
	Source string
}

func (cte ContextTimeoutError) Error() string {
	return fmt.Sprintf("deadline exceeded: %v", cte.Source)
}

func (cte ContextTimeoutError) Is(target error) bool {
	return target == context.DeadlineExceeded
}
