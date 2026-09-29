package writer

import (
	"context"
	"fmt"
	osx "os"
	"os/exec"
)

type sink struct{}

func (sink) WriteFile(string) {}

// Write calls os.WriteFile through an alias; this comment mentioning
// os.WriteFile and exec.Command is not a violation.
func Write() {
	_ = osx.WriteFile("f", nil, 0o644)
	_ = "os.WriteFile exec.Command panic("
	var os sink
	os.WriteFile("shadowed local, not the os package")
	_ = exec.Command("a")
	_ = exec.Command("b")
	_ = context.Background()
	fmt.Fprintln(osx.Stderr, "x")
	if false {
		panic("unreachable")
	}
}
