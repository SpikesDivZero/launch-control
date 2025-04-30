package component

import (
	"errors"
	"testing"

	"github.com/shoenig/test"
)

func Test_checkForPrematureClose(t *testing.T) {
	test.ErrorIs(t, checkForPrematureClose(nil, true), nil)
	test.ErrorIs(t, checkForPrematureClose(nil, false), errPrematureChannelClose)

	testErr := errors.New("test error")
	test.ErrorIs(t, checkForPrematureClose(testErr, true), testErr)
	test.ErrorIs(t, checkForPrematureClose(testErr, false), errPrematureChannelClose)
}
