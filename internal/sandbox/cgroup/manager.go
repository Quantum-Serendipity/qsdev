package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// Manager creates and removes cgroup scopes for sandboxed hook execution.
type Manager struct {
	basePath string // e.g., /sys/fs/cgroup/user.slice/user-1000.slice
}

// NewManager returns a Manager rooted at the cgroup user slice for uid.
func NewManager(uid string) *Manager {
	return &Manager{
		basePath: "/sys/fs/cgroup/user.slice/user-" + uid + ".slice",
	}
}

// ScopePath returns the full path to a named cgroup scope directory.
func (m *Manager) ScopePath(name string) string {
	return filepath.Join(m.basePath, "qsdev-hooks.scope", name)
}

// CreateScope creates a cgroup scope directory and writes resource limit files.
// It returns the scope path on success.
func (m *Manager) CreateScope(name string, limits sandbox.ResourceLimits) (string, error) {
	scopePath := m.ScopePath(name)

	if err := os.MkdirAll(scopePath, 0o755); err != nil {
		return "", fmt.Errorf("creating cgroup scope directory: %w", err)
	}

	if err := os.WriteFile(filepath.Join(scopePath, "memory.max"), formatMemoryMax(limits.MemoryBytes), 0o644); err != nil {
		return "", fmt.Errorf("writing memory.max: %w", err)
	}

	if err := os.WriteFile(filepath.Join(scopePath, "pids.max"), formatPIDsMax(limits.MaxPIDs), 0o644); err != nil {
		return "", fmt.Errorf("writing pids.max: %w", err)
	}

	if err := os.WriteFile(filepath.Join(scopePath, "cpu.max"), formatCPUMax(limits.CPUQuotaPercent), 0o644); err != nil {
		return "", fmt.Errorf("writing cpu.max: %w", err)
	}

	return scopePath, nil
}

// Cleanup removes a cgroup scope directory.
func (m *Manager) Cleanup(scopePath string) error {
	if err := os.RemoveAll(scopePath); err != nil {
		return fmt.Errorf("removing cgroup scope: %w", err)
	}
	return nil
}

// formatMemoryMax formats the memory.max cgroup control value.
func formatMemoryMax(bytes int64) []byte {
	return []byte(strconv.FormatInt(bytes, 10))
}

// formatPIDsMax formats the pids.max cgroup control value.
func formatPIDsMax(maxPIDs int) []byte {
	return []byte(strconv.Itoa(maxPIDs))
}

// formatCPUMax formats the cpu.max cgroup control value as "<quota> <period>".
// The quota is CPUQuotaPercent * 1000 and the period is always 100000.
func formatCPUMax(cpuQuotaPercent int) []byte {
	quota := cpuQuotaPercent * 1000
	return []byte(strconv.Itoa(quota) + " 100000")
}
