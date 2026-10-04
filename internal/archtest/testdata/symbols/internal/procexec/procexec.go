package procexec

import (
	"fmt"
	"os"
	"os/exec"
)

// Run is the owner of exec.Command, may write to os.Stderr and may panic
// from its forbid-exec guard.
func Run() {
	if os.Getenv("FORBID") != "" {
		panic("forbidden")
	}
	_ = exec.Command("a")
	fmt.Fprintln(os.Stderr, "x")
}
