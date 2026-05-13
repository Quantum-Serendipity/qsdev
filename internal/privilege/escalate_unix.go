//go:build !windows

package privilege

import (
	"os"
	"os/exec"
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
	tool := DetectElevationTool()
	switch tool {
	case "sudo":
		return exec.Command("sudo", "-n", "true").Run() == nil
	case "doas":
		return exec.Command("doas", "-n", "true").Run() == nil
	default:
		return false
	}
}
