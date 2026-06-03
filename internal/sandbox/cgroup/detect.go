package cgroup

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// DetectCgroupV2 reports whether the system uses the cgroup v2 unified
// hierarchy by checking for /sys/fs/cgroup/cgroup.controllers.
func DetectCgroupV2(prober sandbox.SandboxProber) bool {
	_, err := prober.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil
}

// DetectDelegation reports whether cgroup delegation is available for the
// given user. It checks that the user's cgroup slice has controllers enabled.
func DetectDelegation(prober sandbox.SandboxProber, uid string) bool {
	if uid == "" {
		return false
	}

	path := "/sys/fs/cgroup/user.slice/user-" + uid + ".slice/cgroup.controllers"
	data, err := prober.ReadFile(path)
	if err != nil {
		return false
	}

	return len(strings.TrimSpace(string(data))) > 0
}
