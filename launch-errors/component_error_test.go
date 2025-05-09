package launchErrors

import (
	"errors"
	"testing"

	"github.com/shoenig/test"
)

func TestComponentError_Basics(t *testing.T) {
	innerErr := errors.New("what went wrong")
	err := error(ComponentError{"cname", "in-this", innerErr})

	test.Eq(t, "component cname in-this: what went wrong", err.Error())
	test.ErrorIs(t, err, innerErr)
	test.EqOp(t, innerErr, errors.Unwrap(err))
}
