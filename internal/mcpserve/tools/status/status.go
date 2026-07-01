package status

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// statusChecker implements 2-tier drift detection. Tier 1 is a fast-path
// (≤5ms): it compares the state file's modification time against the cached
// value and, when unchanged, returns the cached status without re-running
// detection. Tier 2 re-runs ecosystem detection and diffs it against the cached
// snapshot to surface drift (added/removed ecosystems, changed tool versions,
// configuration changes).
type statusChecker struct {
	projectRoot string
	statePath   string
	configPath  string

	// detectFn runs ecosystem detection. Injectable so tests can drive the
	// concurrency behavior; defaults to detect.Detect. It is invoked WITHOUT
	// holding mu so concurrent callers are not serialized behind the walk.
	detectFn func(projectRoot string) types.DetectedProject

	mu               sync.Mutex
	baseDetection    types.DetectedProject
	cachedMtime      time.Time
	cachedPresent    bool
	cachedConfigHash string
	cachedStatus     map[string]any
}

// newStatusChecker captures the baseline detection snapshot and state-file
// fingerprint at construction so the first Tier 1 call has a cache to return.
func newStatusChecker(projectRoot string) *statusChecker {
	s := &statusChecker{
		projectRoot: projectRoot,
		statePath:   filepath.Join(projectRoot, state.StateFilePaths()[0]),
		configPath:  filepath.Join(projectRoot, branding.Get().ConfigFile),
		detectFn:    detect.Detect,
	}
	s.baseDetection = s.detectFn(projectRoot)
	s.cachedMtime, s.cachedPresent = s.stateMtime()
	s.cachedConfigHash = s.configHash()
	s.cachedStatus = map[string]any{
		"tier":          1,
		"cached":        true,
		"state_present": s.cachedPresent,
		"drift":         []driftItem{},
		"drift_count":   0,
		"note":          "baseline snapshot captured at server start",
	}
	return s
}

// driftItem is one detected difference between the cached and current project.
type driftItem struct {
	Type     string `json:"type"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"` // info | warning
}

// handle resolves the requested tier (or auto-selects) and returns the status.
// The lock is held only briefly to snapshot cache state; the expensive Tier 2
// detection runs unlocked so concurrent fast-path callers are not blocked.
func (s *statusChecker) handle(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	tier := toolutil.StringArgOr(req.Arguments, "tier", "auto")

	s.mu.Lock()
	unchanged := s.stateUnchangedLocked()
	useTier1 := tier == "1" || (tier == "auto" && unchanged)
	// Copy the cache under the lock (only when we will actually return it) so the
	// caller cannot mutate it and a concurrent Tier 2 swap cannot race the read.
	var cached map[string]any
	if useTier1 {
		cached = copyStatus(s.cachedStatus)
	}
	s.mu.Unlock()

	if useTier1 {
		text := fmt.Sprintf("status (tier 1, cached): %d drift item(s)", driftCount(cached))
		return toolutil.Result(text, cached), nil
	}
	return s.runTier2()
}

// runTier2 re-runs detection, diffs against the cached snapshot, refreshes the
// cache, and returns the thorough status. Detection runs WITHOUT s.mu held; the
// lock is taken only to snapshot the baseline and, at the end, to swap in the
// refreshed cache (last write wins under concurrent detection).
func (s *statusChecker) runTier2() (*spi.ToolResult, error) {
	s.mu.Lock()
	base := s.baseDetection
	cachedConfigHash := s.cachedConfigHash
	cachedMtime := s.cachedMtime
	cachedPresent := s.cachedPresent
	s.mu.Unlock()

	// Unlocked: detection may walk the filesystem and shell out to a container
	// runtime, so holding the lock here would serialize the fast path.
	current := s.detectFn(s.projectRoot)
	drift := compareDetection(base, current)

	curHash := s.configHash()
	if curHash != cachedConfigHash {
		drift = append(drift, driftItem{
			Type:     "config_changed",
			Detail:   branding.Get().ConfigFile + " changed since the cached snapshot",
			Severity: "warning",
		})
	}

	curMtime, curPresent := s.stateMtime()
	if curPresent != cachedPresent || !curMtime.Equal(cachedMtime) {
		drift = append(drift, driftItem{
			Type:     "state_changed",
			Detail:   "generated-file state changed since the cached snapshot",
			Severity: "info",
		})
	}

	st, stErr := state.LoadStateFromFile(s.statePath)
	stateInfo := map[string]any{"present": curPresent}
	if stErr == nil && curPresent {
		stateInfo["enabled_tools"] = len(st.EnabledTools)
		stateInfo["files"] = len(st.Files)
		stateInfo["last_run"] = st.LastRun
	}

	status := map[string]any{
		"tier":          2,
		"cached":        false,
		"state_present": curPresent,
		"drift":         drift,
		"drift_count":   len(drift),
		"state":         stateInfo,
	}

	// Re-acquire the lock only to refresh the cache so the next Tier 1 call
	// reflects this thorough run.
	s.mu.Lock()
	s.baseDetection = current
	s.cachedMtime = curMtime
	s.cachedPresent = curPresent
	s.cachedConfigHash = curHash
	s.cachedStatus = copyStatus(status)
	s.cachedStatus["tier"] = 1
	s.cachedStatus["cached"] = true
	s.mu.Unlock()

	text := fmt.Sprintf("status (tier 2): %d drift item(s)", len(drift))
	return toolutil.Result(text, status), nil
}

// stateUnchangedLocked reports whether the state file is in the same
// present/mtime condition as the cache. The caller holds s.mu.
func (s *statusChecker) stateUnchangedLocked() bool {
	mtime, present := s.stateMtime()
	if present != s.cachedPresent {
		return false
	}
	if !present {
		return true // both absent
	}
	return mtime.Equal(s.cachedMtime)
}

// stateMtime returns the state file's modification time and whether it exists.
func (s *statusChecker) stateMtime() (time.Time, bool) {
	info, err := os.Stat(s.statePath)
	if err != nil {
		return time.Time{}, false
	}
	return info.ModTime(), true
}

// configHash returns a content hash of the project config file, or "" when it is
// absent or unreadable.
func (s *statusChecker) configHash() string {
	h, err := state.ComputeFileHash(s.configPath)
	if err != nil {
		return ""
	}
	return h
}

// compareDetection diffs two detection snapshots into drift items: ecosystems
// added or removed and changed language tool versions.
func compareDetection(base, cur types.DetectedProject) []driftItem {
	var drift []driftItem

	for eco, present := range cur.Ecosystems {
		if present && !base.Ecosystems[eco] {
			drift = append(drift, driftItem{Type: "ecosystem_added", Detail: eco, Severity: "info"})
		}
	}
	for eco, present := range base.Ecosystems {
		if present && !cur.Ecosystems[eco] {
			drift = append(drift, driftItem{Type: "ecosystem_removed", Detail: eco, Severity: "warning"})
		}
	}

	versionDrift(&drift, "go", base.GoVersion, cur.GoVersion)
	versionDrift(&drift, "node", base.NodeVersion, cur.NodeVersion)
	versionDrift(&drift, "python", base.PythonVersion, cur.PythonVersion)

	sort.Slice(drift, func(i, j int) bool {
		if drift[i].Type != drift[j].Type {
			return drift[i].Type < drift[j].Type
		}
		return drift[i].Detail < drift[j].Detail
	})
	return drift
}

// versionDrift appends a tool_version_changed item when a language version moved.
func versionDrift(drift *[]driftItem, lang, base, cur string) {
	if base != cur {
		*drift = append(*drift, driftItem{
			Type:     "tool_version_changed",
			Detail:   fmt.Sprintf("%s: %q -> %q", lang, base, cur),
			Severity: "warning",
		})
	}
}

// copyStatus returns a shallow copy of a status map so cached state is not
// mutated by a returned result.
func copyStatus(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// driftCount reads the drift_count field defensively for text formatting.
func driftCount(m map[string]any) int {
	if v, ok := m["drift_count"].(int); ok {
		return v
	}
	return 0
}
