//go:build !windows

package privilege

import (
	"context"
	"os"
	"os/exec"
	"time"
)

func NeedsElevation() bool {
	return os.Getuid() != 0
}

func DetectElevationTool() string {
	for _, tool := range []string{"sudo", "doas", "pkexec"} {
		if _, err := exec.LookPath(tool); err == nil {
			return tool
		}
	}
	return ""
}

func HasCachedCredentials() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tool := DetectElevationTool()
	switch tool {
	case "sudo":
		return exec.CommandContext(ctx, "sudo", "-n", "true").Run() == nil
	case "doas":
		return exec.CommandContext(ctx, "doas", "-n", "true").Run() == nil
	default:
		return false
	}
}
