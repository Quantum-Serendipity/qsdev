package catalog

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// OrgConfigPin is the org overlay a human approved for a project: the path
// OrgConfigPath resolved when a human ran init at their own terminal, kept in
// the project's state directory (state.OrgOverlayFile), which the
// self-protection hook keeps the agent from changing.
//
// It is what makes the overlay location structural rather than a matter of
// which commands and files the hook recognises. A run no human started (an
// agent's tool call, a script) reads the pinned overlay, not one that
// <EnvPrefix>ORG_CONFIG points elsewhere however the variable reached the
// process: a command line, an env file, a devenv import or a sourced shell
// fragment.
type OrgConfigPin struct {
	// Path is the overlay path recorded, already resolved (see
	// fileutil.ResolvePath), or "" when none was resolved.
	Path string
	// Recorded is set when a pin was recorded for the project.
	Recorded bool
}

var (
	pinMu     sync.Mutex
	pinActive bool
	pinned    OrgConfigPin
	pinRoot   string
	// warnDrift reports, once per process, an overlay that is not read.
	warnDrift sync.Once
)

// UseOrgConfigPin makes the catalog read the overlay pin allows for the
// project at projectRoot (see OrgConfigDrift and PolicyOrgConfigFile) instead
// of whichever OrgConfigPath resolves. main calls it (via
// instance.UseProjectDefaults) for a run no human started, before any command
// runs; it has no effect once Default has loaded the catalog.
func UseOrgConfigPin(projectRoot string, pin OrgConfigPin) {
	pinMu.Lock()
	defer pinMu.Unlock()
	pinActive, pinRoot, pinned = true, projectRoot, pin
}

// LoadOrgConfigPin returns the pin recorded for the project at projectRoot,
// or a zero pin when none is recorded.
func LoadOrgConfigPin(projectRoot string) (OrgConfigPin, error) {
	path, ok, err := state.LoadOrgOverlay(projectRoot)
	if err != nil {
		return OrgConfigPin{}, fmt.Errorf("loading the recorded org overlay: %w", err)
	}
	return OrgConfigPin{Path: path, Recorded: ok}, nil
}

// RecordOrgConfigPin records the overlay path OrgConfigPath resolves as the
// project's pin, for a human's init at their own terminal, and returns it.
// A path the agent could write (see untrustedOrgConfigLocation) is refused
// and nothing is recorded.
func RecordOrgConfigPin(projectRoot string) (string, error) {
	path, err := resolveOrgConfig(OrgConfigPath())
	if err != nil {
		return "", err
	}
	if reason := untrustedOrgConfigLocation(path, projectRoot); reason != "" {
		return "", fmt.Errorf("not recording the org overlay %s: %s", path, reason)
	}
	if err := state.SaveOrgOverlay(projectRoot, path); err != nil {
		return "", fmt.Errorf("recording the org overlay: %w", err)
	}
	return path, nil
}

// OrgConfigDrift returns why the overlay OrgConfigPath resolves now is not
// one a run no human started may read for the project at projectRoot, given
// its pin, or "" when it may: it lies below the project or the temporary
// directory, which the agent can write, or it is not the overlay a human
// recorded. check and doctor report it; such a run reads the pinned overlay
// instead (PolicyOrgConfigFile).
func OrgConfigDrift(projectRoot string, pin OrgConfigPin) string {
	resolved, err := resolveOrgConfig(OrgConfigPath())
	if err != nil {
		return err.Error()
	}
	if reason := untrustedOrgConfigLocation(resolved, projectRoot); reason != "" {
		return fmt.Sprintf("the org overlay %s %s", resolved, reason)
	}
	if pin.Recorded && !sameOverlay(resolved, pin.Path) {
		return fmt.Sprintf("the org overlay resolves to %s, not %s, which a human recorded at init (%s)",
			describeOverlay(resolved), describeOverlay(pin.Path), state.OrgOverlayFile())
	}
	return ""
}

// ProjectOrgConfigDrift is OrgConfigDrift for the pin recorded for the
// project at projectRoot; a pin that cannot be read is reported too.
func ProjectOrgConfigDrift(projectRoot string) string {
	pin, err := LoadOrgConfigPin(projectRoot)
	if err != nil {
		return err.Error()
	}
	return OrgConfigDrift(projectRoot, pin)
}

// PolicyOrgConfigFile returns the org overlay file the catalog applies: the
// one OrgConfigFile names, or, when UseOrgConfigPin is in force and that
// overlay drifts (OrgConfigDrift), the pinned overlay (the account's home
// overlay when none is pinned), with a warning. It returns "" when the
// overlay it settles on does not exist.
func PolicyOrgConfigFile() string {
	pinMu.Lock()
	active, root, pin := pinActive, pinRoot, pinned
	pinMu.Unlock()
	if !active {
		return OrgConfigFile()
	}
	drift := OrgConfigDrift(root, pin)
	if drift == "" {
		return OrgConfigFile()
	}
	fallback := pin.Path
	if !pin.Recorded {
		fallback = homeOrgConfigPath()
	}
	if fallback != "" && untrustedOrgConfigLocation(fallback, root) != "" {
		fallback = ""
	}
	warnDrift.Do(func() {
		slog.Warn("ignoring the org overlay this run resolves: "+drift+
			"; a human can record another one by running '"+branding.Get().AppName+" init' at their own terminal",
			"using", describeOverlay(fallback))
	})
	if fallback == "" || !fileExists(fallback) {
		return ""
	}
	return fallback
}

// resolveOrgConfig resolves an overlay path for comparison (see
// fileutil.ResolvePath); "" stays "".
func resolveOrgConfig(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	resolved, err := fileutil.ResolvePath(path)
	if err != nil {
		return "", fmt.Errorf("resolving the org overlay %s: %w", path, err)
	}
	return resolved, nil
}

// sameOverlay reports whether the resolved overlay path resolved names the
// recorded one, recorded, which is resolved again in case a symlink on it
// changed.
func sameOverlay(resolved, recorded string) bool {
	again, err := resolveOrgConfig(recorded)
	return err == nil && again == resolved
}

// untrustedOrgConfigLocation returns why path lies where the agent can write
// it, or "": below the project at projectRoot, or below the temporary
// directory.
func untrustedOrgConfigLocation(path, projectRoot string) string {
	if path == "" {
		return ""
	}
	if projectRoot != "" && fileutil.PathWithin(projectRoot, path) {
		return "lies below the project " + projectRoot
	}
	if tmp := os.TempDir(); tmp != "" && fileutil.PathWithin(tmp, path) {
		return "lies below the temporary directory " + tmp
	}
	return ""
}

// describeOverlay names an overlay path for a message: the path, or "no
// overlay".
func describeOverlay(path string) string {
	if path == "" {
		return "no overlay"
	}
	return path
}
