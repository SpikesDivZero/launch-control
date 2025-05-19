package lcerrors

import (
	"context"
	"testing"

	"github.com/shoenig/test"
)

func TestContextTimeoutError_Basics(t *testing.T) {
	err := error(ContextTimeoutError{"in-this"})

	test.Eq(t, "deadline exceeded: in-this", err.Error())
	test.ErrorIs(t, err, context.DeadlineExceeded)
}
