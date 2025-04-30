package component

import "errors"

var errPrematureChannelClose = errors.New("channel closed without sending a result value")

func checkForPrematureClose(err error, ok bool) error {
	if ok {
		return err
	}
	return errPrematureChannelClose
}
