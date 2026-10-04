//go:build windows

package writer

import "os/exec"

func look() { _, _ = exec.LookPath("x") }
