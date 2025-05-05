package debug

import (
	"strings"
	"testing"

	"github.com/spikesdivzero/launch-control/internal/testutil"
)

func TestTidyStack(t *testing.T) {
	f1 := func() string { return TidyStack(2) } // don't include f1 or f2
	f2 := func() string { return f1() }
	f3 := func() string { return f2() }
	f4 := func() string { return f3() }

	stackLines := strings.Split(f4(), "\n")

	// Our tests here tend to be a bit brittle, so I'm being generous with the FailNow calls.
	failOutWithStack := func() {
		for _, line := range stackLines {
			t.Logf("\tgot stack: %v", line)
		}
		t.FailNow()
	}

	// We expect there to be 4 function calls in the stack, with each call containing two lines.
	if len(stackLines) < 8 {
		t.Error("stack should have at least 8 lines")
		failOutWithStack()
	}

	// The first line should not be a goroutine line
	if strings.HasPrefix(stackLines[0], "goroutine") {
		t.Error("stack should not have the `goroutine N [running]` line")
		failOutWithStack()
	}

	// The first and last lines should not be empty
	if stackLines[0] == "" {
		t.Error("stack should not begin with a blank line")
		failOutWithStack()
	}

	if stackLines[len(stackLines)-1] == "" {
		t.Error("stack should not end with a blank line")
		failOutWithStack()
	}

	// How many times did this function show up? It should be 3 (itself as well as two non-skipped inner funcs: f3, f4)
	//
	// The above tests set us up to ensure that even lines are function calls, and odd lines are filenames.
	needle, count := "/debug.TestTidyStack", 0
	for i, line := range stackLines {
		if i%2 == 0 && strings.Contains(line, needle) {
			count++
		}
	}
	if count != 3 {
		t.Errorf("stack contains %q %v times, wanted 3", needle, count)
		failOutWithStack()
	}

	// And one sin of a test...
	t.Run("panic on too large skip", func(t *testing.T) {
		defer testutil.WantPanic(t, "internal error: chompLine failed: got idx = -1 while looking for newline")
		TidyStack(10000)
	})
}
