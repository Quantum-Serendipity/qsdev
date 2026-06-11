// Package exitcode provides an error type that carries a process exit code.
// When returned from a cobra RunE handler, the gdev framework's ExitCodeErr
// interface extracts the code and exits with it instead of the default 1.
package exitcode

import "fmt"

// Error is an error that carries a specific process exit code. It satisfies
// the gdev cmd.ExitCodeErr interface so the framework propagates the code.
type Error struct {
	Code    int
	Message string
}

// New creates an Error with the given exit code and a formatted message.
func New(code int, format string, args ...any) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

func (e *Error) Error() string { return e.Message }

// ExitCode returns the exit code to use when the program exits due to this
// error. This method satisfies the gdev cmd.ExitCodeErr interface.
func (e *Error) ExitCode() int { return e.Code }
