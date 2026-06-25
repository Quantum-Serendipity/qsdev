package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeKind classifies a graph mutation the watcher applied.
type ChangeKind int

const (
	// ChangeNone indicates an event the watcher ignored.
	ChangeNone ChangeKind = iota
	// ChangeRescan indicates a membership-config change that triggered a full
	// DetectWorkspaces re-scan.
	ChangeRescan
	// ChangeSinglePackage indicates a member-manifest change that triggered a
	// single-package re-parse.
	ChangeSinglePackage
)

// String renders a ChangeKind for logs.
func (k ChangeKind) String() string {
	switch k {
	case ChangeRescan:
		return "rescan"
	case ChangeSinglePackage:
		return "single_package"
	default:
		return "none"
	}
}

// watcherState is the watcher's coalescing state machine:
// Idle -> Debouncing (events arriving) -> Rescanning (applying) -> Idle.
type watcherState int

const (
	stateIdle watcherState = iota
	stateDebouncing
	stateRescanning
)

// Default debounce windows: membership configs coalesce over a longer window
// because a full re-scan is heavier; member manifests use a short window.
const (
	defaultMembershipDebounce = 500 * time.Millisecond
	defaultMemberDebounce     = 50 * time.Millisecond
)

// membershipConfigs are the root configuration filenames whose change triggers a
// full workspace re-scan.
var membershipConfigs = map[string]struct{}{
	fileNpmManifest:    {},
	filePnpmWorkspace:  {},
	filePnpmWorkspace2: {},
	fileCargoManifest:  {},
	fileGoWork:         {},
	filePyprojectToml:  {},
}

// memberManifests are the per-member manifest filenames whose change triggers a
// single-package re-parse.
var memberManifests = map[string]struct{}{
	fileNpmManifest:   {},
	fileCargoManifest: {},
	fileGoModManifest: {},
	filePyprojectToml: {},
}

// Watcher observes a workspace's configuration with fsnotify and keeps a
// WorkspaceGraph in sync. It implements the two-category strategy: a change to a
// root membership config triggers a full re-scan (DetectWorkspaces), while a
// change to a member manifest triggers a single-package re-parse of just that
// entry. Rapid changes are coalesced through per-category debounce timers and
// the Idle -> Debouncing -> Rescanning -> Idle state machine.
//
// The watcher signals catalog changes through an injected onChange callback (the
// server's NotifyToolsListChanged), keeping the package decoupled from mcpserve.
type Watcher struct {
	root  string
	graph *WorkspaceGraph
	fsw   *fsnotify.Watcher

	membershipDebounce time.Duration
	memberDebounce     time.Duration

	// onChange is the decoupled seam invoked after every applied graph change;
	// the server supplies NotifyToolsListChanged. May be nil.
	onChange func()

	// onChangeKind is an unexported test hook invoked with the applied change
	// kind after a mutation. Always nil in production.
	onChangeKind func(ChangeKind)

	mu    sync.Mutex
	state watcherState

	closeOnce sync.Once
	done      chan struct{}
}

// WatcherOption configures a Watcher at construction.
type WatcherOption func(*Watcher)

// WithOnChange sets the callback invoked after each applied graph change. The
// server passes its NotifyToolsListChanged method here.
func WithOnChange(fn func()) WatcherOption {
	return func(w *Watcher) { w.onChange = fn }
}

// WithDebounce overrides the membership and member debounce windows. A
// non-positive value leaves the corresponding default in place.
func WithDebounce(membership, member time.Duration) WatcherOption {
	return func(w *Watcher) {
		if membership > 0 {
			w.membershipDebounce = membership
		}
		if member > 0 {
			w.memberDebounce = member
		}
	}
}

// NewWatcher constructs a Watcher bound to graph. It creates the underlying
// fsnotify watcher but does not begin watching until Start is called.
func NewWatcher(graph *WorkspaceGraph, opts ...WatcherOption) (*Watcher, error) {
	if graph == nil {
		return nil, fmt.Errorf("workspace watcher: graph is required")
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}
	w := &Watcher{
		root:               graph.Root(),
		graph:              graph,
		fsw:                fsw,
		membershipDebounce: defaultMembershipDebounce,
		memberDebounce:     defaultMemberDebounce,
		state:              stateIdle,
		done:               make(chan struct{}),
	}
	for _, o := range opts {
		o(w)
	}
	return w, nil
}

// Start adds the initial watches (the workspace root plus every member
// directory) and launches the watch loop. The loop exits cleanly when ctx is
// cancelled or Close is called.
func (w *Watcher) Start(ctx context.Context) error {
	w.addWatches()
	go w.loop(ctx)
	return nil
}

// Close stops the watcher and releases the fsnotify resources. It is safe to
// call multiple times.
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.done)
		err = w.fsw.Close()
	})
	return err
}

// State returns the watcher's current coalescing state (primarily for tests).
func (w *Watcher) State() watcherState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.state
}

func (w *Watcher) setState(s watcherState) {
	w.mu.Lock()
	w.state = s
	w.mu.Unlock()
}

// addWatches watches the workspace root (catching membership configs) and each
// member directory (catching member manifests). Re-adding an already watched
// path is harmless, so it doubles as a refresh after a re-scan adds members.
func (w *Watcher) addWatches() {
	if err := w.fsw.Add(w.root); err != nil {
		slog.Debug("workspace watcher: failed to watch root", "root", w.root, "error", err)
	}
	for _, p := range w.graph.Packages() {
		if p.RelDir == "." {
			continue // the root is already watched
		}
		dir := filepath.Join(w.root, filepath.FromSlash(p.RelDir))
		if err := w.fsw.Add(dir); err != nil {
			slog.Debug("workspace watcher: failed to watch member dir", "dir", dir, "error", err)
		}
	}
}

// loop is the watch goroutine. It debounces membership and member changes on
// independent timers and applies them through the state machine.
func (w *Watcher) loop(ctx context.Context) {
	membershipTimer := time.NewTimer(time.Hour)
	memberTimer := time.NewTimer(time.Hour)
	membershipTimer.Stop()
	memberTimer.Stop()
	defer membershipTimer.Stop()
	defer memberTimer.Stop()

	pending := map[string]struct{}{}

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue // ignore chmod-only events
			}
			kind, relDir := w.categorize(ev.Name)
			switch kind {
			case ChangeRescan:
				w.setState(stateDebouncing)
				armTimer(membershipTimer, w.membershipDebounce)
			case ChangeSinglePackage:
				w.setState(stateDebouncing)
				pending[relDir] = struct{}{}
				armTimer(memberTimer, w.memberDebounce)
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			if err != nil {
				slog.Warn("workspace watcher error", "error", err)
			}

		case <-membershipTimer.C:
			w.setState(stateRescanning)
			w.rescan()
			// A full re-scan supersedes any queued single-package work.
			pending = map[string]struct{}{}
			drainTimer(memberTimer)
			w.setState(stateIdle)

		case <-memberTimer.C:
			w.setState(stateRescanning)
			members := pending
			pending = map[string]struct{}{}
			w.reparseMembers(members)
			w.setState(stateIdle)
		}
	}
}

// categorize maps a changed path to a change kind. A membership config at the
// root yields a full re-scan; a member manifest in a subtree yields a
// single-package re-parse keyed by the package's relative directory. The root
// membership configs (package.json, Cargo.toml, pyproject.toml) take precedence
// over their member-manifest spelling.
func (w *Watcher) categorize(path string) (ChangeKind, string) {
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	relParent, err := filepath.Rel(w.root, dir)
	if err != nil {
		return ChangeNone, ""
	}
	relParent = filepath.ToSlash(relParent)
	if relParent == ".." || strings.HasPrefix(relParent, "../") {
		return ChangeNone, ""
	}
	atRoot := relParent == "."

	if atRoot {
		if _, ok := membershipConfigs[base]; ok {
			return ChangeRescan, ""
		}
	}
	if _, ok := memberManifests[base]; ok {
		return ChangeSinglePackage, normalizeRelDir(relParent)
	}
	return ChangeNone, ""
}

// rescan performs a full re-scan and atomically replaces the graph's contents,
// preserving the graph pointer callers (e.g. projectctx) hold.
func (w *Watcher) rescan() {
	newGraph, err := DetectWorkspaces(w.root)
	if err != nil {
		slog.Warn("workspace re-scan failed", "error", err)
		return
	}
	pkgs := newGraph.Packages()
	m := make(map[string]*Package, len(pkgs))
	for _, p := range pkgs {
		m[p.RelDir] = p
	}
	w.graph.ReplaceAll(m)
	w.addWatches() // pick up any newly added member directories
	w.notify(ChangeRescan)
}

// reparseMembers re-parses each queued member directory and notifies once if any
// entry changed.
func (w *Watcher) reparseMembers(members map[string]struct{}) {
	changed := false
	for relDir := range members {
		if w.reparseOne(relDir) {
			changed = true
		}
	}
	if changed {
		w.addWatches()
		w.notify(ChangeSinglePackage)
	}
}

// reparseOne re-reads the manifest for relDir and updates just that graph entry.
// An existing package is rebuilt with its own ecosystem; a removed manifest
// drops the package; a directory that has become a new member is added under the
// first ecosystem whose root config and member manifest both apply. It reports
// whether the graph changed.
func (w *Watcher) reparseOne(relDir string) bool {
	if existing := w.graph.ByRelDir(relDir); existing != nil {
		spec, ok := specByID(existing.Ecosystem)
		if !ok {
			return false
		}
		pkg, err := buildPackage(w.root, spec, relDir)
		if err != nil {
			return w.graph.Remove(relDir) // manifest gone or unreadable
		}
		w.graph.Upsert(pkg)
		return true
	}

	for _, spec := range allEcosystems() {
		if !fileExists(filepath.Join(w.root, spec.configFile)) {
			continue
		}
		pkg, err := buildPackage(w.root, spec, relDir)
		if err != nil {
			continue
		}
		w.graph.Upsert(pkg)
		return true
	}
	return false
}

// notify invokes the decoupled change seam and the test hook.
func (w *Watcher) notify(kind ChangeKind) {
	if w.onChange != nil {
		w.onChange()
	}
	if w.onChangeKind != nil {
		w.onChangeKind(kind)
	}
}

// specByID returns the ecosystem spec with the given id.
func specByID(id string) (ecosystemSpec, bool) {
	for _, s := range allEcosystems() {
		if s.id == id {
			return s, true
		}
	}
	return ecosystemSpec{}, false
}

// armTimer (re)arms t to fire after d, safely draining any pending value so
// Reset is only applied to a stopped, drained timer.
func armTimer(t *time.Timer, d time.Duration) {
	drainTimer(t)
	t.Reset(d)
}

// drainTimer stops t and removes any value already delivered to its channel.
func drainTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}
