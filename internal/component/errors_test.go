package component

import (
	"errors"
	"testing"

	"github.com/shoenig/test"
)

func TestComponentError(t *testing.T) {
	base := errors.New("base error")
	var wrapped error

	// Error() func + implied type casts
	wrapped = ComponentError{"comp1", "stage1", base}
	test.EqError(t, wrapped, "component error: name=\"comp1\" stage=\"stage1\" err=base error")

	// No error doesn't hallucinate
	wrapped = WrapComponentError("comp0", "stage0", nil)
	test.Nil(t, wrapped)

	// String component name works
	wrapped = WrapComponentError("strname", "stage2", base)
	test.EqError(t, wrapped, ComponentError{"strname", "stage2", base}.Error())

	// Object component name also works
	c := &Component{Name: "objname"}
	wrapped = WrapComponentError(c, "stage2", base)
	test.EqError(t, wrapped, ComponentError{"objname", "stage2", base}.Error())

	// Correctly implements error wrapping for the errors package
	test.ErrorIs(t, wrapped, base)
}

func TestPrematureChannelCloseError(t *testing.T) {
	base := errors.New("base")

	// Error() func + implied type casts
	test.EqError(t, PrematureChannelCloseError{"chan1"}, "channel \"chan1\" closed prematurely")

	// When readOk is true, we should always pass through the original value
	test.Nil(t, CheckPrematureChannelClose("foo", nil, true))
	test.EqOp(t, base, CheckPrematureChannelClose("foo", base, true))

	// The actual error case we're interested in
	test.EqError(t,
		CheckPrematureChannelClose("chan2", nil, false),
		PrematureChannelCloseError{"chan2"}.Error())
}
