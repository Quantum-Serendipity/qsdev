package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// maxCPUQuotaPercent caps CPUQuota= at 100 cores; larger values are treated as
// misconfiguration and omitted.
const maxCPUQuotaPercent = 10000

// userBusVars are the variables systemd-run --user uses to reach the user's
// service manager over D-Bus.
var userBusVars = []string{"XDG_RUNTIME_DIR", "DBUS_SESSION_BUS_ADDRESS"}

// SystemdScopeArgs returns the systemd-run arguments that start a command in a
// transient --user scope carrying the limits in res. The result ends with
// "--"; the caller appends the command. --quiet keeps systemd-run's "Running
// as unit" banner out of the hook's stderr.
func SystemdScopeArgs(res ResourceLimits) []string {
	args := []string{"--user", "--scope", "--quiet"}
	if res.MemoryBytes > 0 {
		args = append(args, "-p", "MemoryMax="+strconv.FormatInt(res.MemoryBytes, 10))
	}
	if res.MaxPIDs > 0 {
		args = append(args, "-p", "TasksMax="+strconv.Itoa(res.MaxPIDs))
	}
	if res.CPUQuotaPercent > 0 && res.CPUQuotaPercent <= maxCPUQuotaPercent {
		args = append(args, "-p", "CPUQuota="+strconv.Itoa(res.CPUQuotaPercent)+"%")
	}
	return append(args, "--")
}

// UserBusEnv returns the user-bus variables of the current process that
// systemd-run --user needs. They are not in the hook environment allowlist, so
// a backend that filters the environment must add them back for systemd-run
// itself.
func UserBusEnv() map[string]string {
	env := make(map[string]string, len(userBusVars))
	for _, k := range userBusVars {
		if v := os.Getenv(k); v != "" {
			env[k] = v
		}
	}
	return env
}

// UserScopeUsable reports whether systemd-run at path can start --user scopes:
// the binary must exist and the user's bus must be reachable. Without a user
// session (SSH without lingering, containers, CI) systemd-run fails every run
// with "Failed to connect to user scope bus", so statting the binary alone is
// not enough.
func UserScopeUsable(path string) error {
	if path == "" {
		return errors.New("systemd-run binary path not set")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("systemd-run binary not found at %s: %w", path, err)
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return nil
	}
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		return errors.New("no systemd user session: neither DBUS_SESSION_BUS_ADDRESS nor XDG_RUNTIME_DIR is set")
	}
	bus := filepath.Join(runtimeDir, "bus")
	if _, err := os.Stat(bus); err != nil {
		return fmt.Errorf("no systemd user session bus at %s: %w", bus, err)
	}
	return nil
}
