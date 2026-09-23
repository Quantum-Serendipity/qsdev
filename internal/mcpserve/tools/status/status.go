package status

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/toolutil"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// statusChecker implements 2-tier drift detection.
//
// Tier 2 (thorough) runs the canonical drift detector (posture/drift.Detect)
// against the persisted state ledger — generated files modified or deleted
// since generation, hook/marker/lockfile drift, tool availability, version
// drift — and additionally diffs ecosystem detection and the project config
// against the anchor taken at the last generation run. Ledger-derived drift is
// recomputed from the ledger on every run, so it keeps being reported until it
// is reconciled; the detection/config anchor is only rebased when the ledger
// itself changes (a regeneration), never merely because drift was reported.
//
// Tier 1 (fast path) returns the last Tier 2 result while nothing it depends on
// has changed: the state file, the project config, and every tracked generated
// file (a cheap stat loop). Any change falls back to Tier 2. There is no Tier 1
// result before the first Tier 2 run.
type statusChecker struct {
	projectRoot string
	statePath   string
	configPath  string

	// detectFn runs ecosystem detection. Injectable so tests can drive the
	// concurrency behavior; defaults to detect.Detect. It is invoked WITHOUT
	// holding mu so concurrent callers are not serialized behind the walk.
	detectFn func(projectRoot string) types.DetectedProject

	mu sync.Mutex
	// anchor is the reference point for detection/config drift: the project as
	// observed when the state ledger was last seen to change (server start
	// counts as the first observation).
	anchor snapshot
	// cachedFP fingerprints the inputs of cachedStatus; Tier 1 is served only
	// while the current fingerprint still matches it. cachedStatus is nil until
	// the first Tier 2 run.
	cachedFP     fingerprint
	cachedStatus map[string]any
}

// snapshot is the drift anchor: detection, config hash, and state-file identity.
type snapshot struct {
	detection  types.DetectedProject
	configHash string
	stateMtime time.Time
	statePres  bool
}

// fingerprint captures everything a Tier 2 result depends on that can change
// on disk: the state file, the config content, and each tracked file's mtime.
type fingerprint struct {
	stateMtime time.Time
	statePres  bool
	configHash string
	files      map[string]time.Time
}

// newStatusChecker captures the drift anchor (detection, config hash and state
// identity) at construction.
func newStatusChecker(projectRoot string) *statusChecker {
	s := &statusChecker{
		projectRoot: projectRoot,
		statePath:   filepath.Join(projectRoot, state.StateFilePaths()[0]),
		configPath:  filepath.Join(projectRoot, branding.Get().ConfigFile),
		detectFn:    detect.Detect,
	}
	mtime, present := s.stateMtime()
	s.anchor = snapshot{
		detection:  s.detectFn(projectRoot),
		configHash: s.configHash(),
		stateMtime: mtime,
		statePres:  present,
	}
	return s
}

// driftItem is one detected difference between the expected and current project.
type driftItem struct {
	Type     string `json:"type"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"` // info | warning | error | critical
}

// handle resolves the requested tier (or auto-selects) and returns the status.
// The lock is held only briefly to snapshot cache state; fingerprinting and the
// expensive Tier 2 detection run unlocked so concurrent callers are not blocked.
func (s *statusChecker) handle(_ context.Context, _ *spi.ToolCallContext, req *spi.ToolRequest) (*spi.ToolResult, error) {
	tier := toolutil.StringArgOr(req.Arguments, "tier", "auto")

	s.mu.Lock()
	// Copy the cache under the lock so the caller cannot mutate it and a
	// concurrent Tier 2 swap cannot race the read.
	cached := copyStatus(s.cachedStatus)
	cachedFP := s.cachedFP
	s.mu.Unlock()

	useTier1 := cached != nil && (tier == "1" || (tier == "auto" && s.unchangedSince(cachedFP)))
	if useTier1 {
		text := fmt.Sprintf("status (tier 1, cached): %d drift item(s)", driftCount(cached))
		return toolutil.Result(text, cached), nil
	}
	return s.runTier2()
}

// runTier2 re-runs detection and the ledger-anchored drift detector, refreshes
// the Tier 1 cache, and returns the thorough status. Detection runs WITHOUT s.mu
// held; the lock is taken only to read and, at the end, update the anchor and
// cache (last write wins under concurrent runs).
func (s *statusChecker) runTier2() (*spi.ToolResult, error) {
	s.mu.Lock()
	anchor := s.anchor
	s.mu.Unlock()

	// Unlocked: detection may walk the filesystem and shell out to a container
	// runtime, so holding the lock here would serialize the fast path.
	current := s.detectFn(s.projectRoot)
	curHash := s.configHash()
	curMtime, curPresent := s.stateMtime()

	var items []driftItem
	if curPresent != anchor.statePres || !curMtime.Equal(anchor.stateMtime) {
		// The ledger changed: a generation run reconciled the project, so rebase
		// the detection/config anchor onto the project as it is now.
		items = append(items, driftItem{
			Type:     "state_changed",
			Detail:   "generated-file state changed (regenerated); drift baseline rebased",
			Severity: "info",
		})
		anchor = snapshot{detection: current, configHash: curHash, stateMtime: curMtime, statePres: curPresent}
	}
	items = append(items, compareDetection(anchor.detection, current)...)
	if curHash != anchor.configHash {
		items = append(items, driftItem{
			Type:     "config_changed",
			Detail:   branding.Get().ConfigFile + " changed since the drift baseline (server start or the last regeneration)",
			Severity: "warning",
		})
	}

	st, stErr := state.LoadStateFromFile(s.statePath)
	stateInfo := map[string]any{"present": curPresent}
	switch {
	case stErr != nil:
		stateInfo["error"] = stErr.Error()
		items = append(items, driftItem{
			Type:     "state_unreadable",
			Detail:   "generated-file state could not be read; generated files were not verified: " + stErr.Error(),
			Severity: "error",
		})
	case curPresent:
		stateInfo["enabled_tools"] = len(st.EnabledTools)
		stateInfo["files"] = len(st.Files)
		stateInfo["last_run"] = st.LastRun
		items = append(items, ledgerDrift(s.projectRoot, st)...)
	}

	status := map[string]any{
		"tier":          2,
		"cached":        false,
		"state_present": curPresent,
		"drift":         items,
		"drift_count":   len(items),
		"state":         stateInfo,
	}
	fp := fingerprint{
		stateMtime: curMtime,
		statePres:  curPresent,
		configHash: curHash,
		files:      s.fileMtimes(st),
	}

	// Re-acquire the lock only to refresh the anchor and cache so the next
	// Tier 1 call reflects this thorough run.
	s.mu.Lock()
	s.anchor = anchor
	s.cachedFP = fp
	s.cachedStatus = copyStatus(status)
	s.cachedStatus["tier"] = 1
	s.cachedStatus["cached"] = true
	s.mu.Unlock()

	text := fmt.Sprintf("status (tier 2): %d drift item(s)", len(items))
	return toolutil.Result(text, status), nil
}

// ledgerDrift runs the canonical drift detector against the state ledger and
// converts its findings into drift items (e.g. a machine-owned generated file
// such as .claude/settings.json modified since generation).
func ledgerDrift(projectRoot string, st types.GeneratedState) []driftItem {
	report := drift.Detect(projectRoot, st, st.EnabledTools)
	var items []driftItem
	for _, cat := range report.Categories {
		kind := strings.ReplaceAll(strings.ToLower(cat.Name), " ", "_")
		for _, f := range cat.Findings {
			items = append(items, driftItem{Type: kind, Detail: f.Description, Severity: string(f.Severity)})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		return items[i].Detail < items[j].Detail
	})
	return items
}

// unchangedSince reports whether the current on-disk inputs still match fp.
func (s *statusChecker) unchangedSince(fp fingerprint) bool {
	mtime, present := s.stateMtime()
	if present != fp.statePres || (present && !mtime.Equal(fp.stateMtime)) {
		return false
	}
	if s.configHash() != fp.configHash {
		return false
	}
	for rel, want := range fp.files {
		if got := fileMtime(filepath.Join(s.projectRoot, rel)); !got.Equal(want) {
			return false
		}
	}
	return true
}

// fileMtimes records the modification time of every file tracked by the ledger
// (the zero time for a missing file).
func (s *statusChecker) fileMtimes(st types.GeneratedState) map[string]time.Time {
	out := make(map[string]time.Time, len(st.Files))
	for rel := range st.Files {
		out[rel] = fileMtime(filepath.Join(s.projectRoot, rel))
	}
	return out
}

// fileMtime returns path's modification time, or the zero time when it cannot
// be stat'ed.
func fileMtime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
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
	var items []driftItem

	for eco, present := range cur.Ecosystems {
		if present && !base.Ecosystems[eco] {
			items = append(items, driftItem{Type: "ecosystem_added", Detail: eco, Severity: "info"})
		}
	}
	for eco, present := range base.Ecosystems {
		if present && !cur.Ecosystems[eco] {
			items = append(items, driftItem{Type: "ecosystem_removed", Detail: eco, Severity: "warning"})
		}
	}

	versionDrift(&items, "go", base.GoVersion, cur.GoVersion)
	versionDrift(&items, "node", base.NodeVersion, cur.NodeVersion)
	versionDrift(&items, "python", base.PythonVersion, cur.PythonVersion)

	sort.Slice(items, func(i, j int) bool {
		if items[i].Type != items[j].Type {
			return items[i].Type < items[j].Type
		}
		return items[i].Detail < items[j].Detail
	})
	return items
}

// versionDrift appends a tool_version_changed item when a language version moved.
func versionDrift(items *[]driftItem, lang, base, cur string) {
	if base != cur {
		*items = append(*items, driftItem{
			Type:     "tool_version_changed",
			Detail:   fmt.Sprintf("%s: %q -> %q", lang, base, cur),
			Severity: "warning",
		})
	}
}

// copyStatus returns a shallow copy of a status map (nil for nil) so cached
// state is not mutated by a returned result.
func copyStatus(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
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
