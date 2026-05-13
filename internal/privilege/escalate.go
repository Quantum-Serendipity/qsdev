package privilege

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// IsElevated reports whether the current process has admin privileges.
func IsElevated() bool {
	return !NeedsElevation()
}

// ElevatedExec runs a command with elevation via the detected tool.
func ElevatedExec(ctx context.Context, name string, args ...string) error {
	tool := DetectElevationTool()
	if tool == "" {
		return fmt.Errorf("no elevation tool available (sudo, doas, pkexec)")
	}

	// Special handling for PowerShell Start-Process on Windows
	if tool == "powershell" {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = "'" + a + "'"
		}
		psCmd := fmt.Sprintf("Start-Process -Verb RunAs -Wait -FilePath '%s' -ArgumentList %s",
			name, strings.Join(quoted, ","))
		c := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	allArgs := make([]string, 0, 1+len(args))
	allArgs = append(allArgs, name)
	allArgs = append(allArgs, args...)
	c := exec.CommandContext(ctx, tool, allArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// BatchElevatedInstall runs a single elevated package install command
// for multiple packages. pm is the package manager binary (e.g., "apt-get"),
// pmArgs is the subcommand (e.g., ["install", "-y"]), and packages are appended.
func BatchElevatedInstall(ctx context.Context, pm string, pmArgs []string, packages []string) error {
	if len(packages) == 0 {
		return nil
	}
	allArgs := make([]string, 0, len(pmArgs)+len(packages))
	allArgs = append(allArgs, pmArgs...)
	allArgs = append(allArgs, packages...)
	return ElevatedExec(ctx, pm, allArgs...)
}
