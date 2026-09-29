package procexec

import (
	"fmt"
	"os"
	"os/exec"
)

// Run is the owner of exec.Command and may write to os.Stderr.
func Run() {
	_ = exec.Command("a")
	fmt.Fprintln(os.Stderr, "x")
}
