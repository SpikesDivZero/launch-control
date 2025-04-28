package debug

import (
	"bytes"
	"runtime/debug"
)

// Wraps [runtime/debug.Stack()], removing cruft that's not desired in this project.
func TidyStack(skip int) string {
	stack := debug.Stack()

	// goroutine 1 [running]:
	// runtime/debug.Stack()
	//     .../1.24.2/libexec/src/runtime/debug/stack.go:26 +0x5e
	// main.captureRunCallStack()
	//     .../main.go:12 +0x17
	// [...]
	// main.main()
	//     .../main.go:39 +0x4d

	// Skip both this and runtime/debug.Stack() call
	skip += 2

	// Removes the leading line from the stack slice
	chompLine := func() {
		idx := bytes.IndexRune(stack, '\n')
		if idx == -1 {
			panic("internal error: chompLine failed: got idx = -1 while looking for newline")
		}
		stack = stack[idx+1:]
	}

	// Remove the leading "goroutine N [running]:" line, if present
	if bytes.HasPrefix(stack, []byte("goroutine")) {
		chompLine()
	}

	for range skip {
		chompLine() // "function()"
		chompLine() // "file:line +offset"
	}

	// As well as the trailing newline so it can be formatted easily
	stack = bytes.TrimSuffix(stack, []byte("\n"))

	return string(stack)
}
