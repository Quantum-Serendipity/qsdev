package catalog

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/userhome"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// OrgConfigPin is the org overlay a human approved: the path OrgConfigPath
// resolved when a human ran '<app> defaults pin' (PinCommandHint), a
// sensitive command the self-protection hook and the CLI's human gate keep
// the agent from running. Pins are kept in the account's home configuration
// directory (OrgPinsFile), next to the default overlay and protected with it,
// so nothing in a checkout (a `git clean -fdX`, a fresh clone) removes one.
//
// It is what makes the overlay location structural rather than a matter of
// which commands and files the hook recognises: every run reads the pinned
// overlay, or without a pin the account's home overlay, not one that
// <EnvPrefix>ORG_CONFIG points elsewhere however the variable reached the
// process (a command line, an env file, a devenv import, a sourced shell
// fragment, a wrapper that makes an agent's command look like a human's).
type OrgConfigPin struct {
	// Path is the overlay path recorded, already resolved (see
	// fileutil.ResolvePath), or "" when none was resolved.
	Path string
	// Recorded is set when a pin was recorded for the project or the account.
	Recorded bool
	// Global is set when the pin is the account-wide one, not one recorded
	// for the project.
	Global bool
	// Unanchored is set when the user database has no entry for the account
	// (userhome.ErrNoAccount; an arbitrary container or CI uid), so there is
	// no home directory to keep pins in or to find the home overlay below.
	// The overlay OrgConfigPath resolves is then read as it is, unless it
	// lies where the agent can write (untrustedOrgConfigLocation): nothing an
	// agent does makes its account unknown to the system, and such an
	// account has no other way to name its overlay.
	Unanchored bool
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
// project at projectRoot ("" outside a project; see OrgConfigDrift and
// PolicyOrgConfigFile) instead of whichever OrgConfigPath resolves. main
// calls it (via instance.UseProjectDefaults) before any command runs; it has
// no effect once Default has loaded the catalog.
func UseOrgConfigPin(projectRoot string, pin OrgConfigPin) {
	pinMu.Lock()
	defer pinMu.Unlock()
	pinActive, pinRoot, pinned = true, projectRoot, pin
}

// PinCommandHint names the command that records a pin, for messages.
func PinCommandHint() string {
	return "'" + branding.Get().AppName + " defaults pin'"
}

// orgPinsRecord is the content of OrgPinsFile.
type orgPinsRecord struct {
	// Global is the account-wide pin, nil when none is recorded; "" pins no
	// overlay.
	Global *string `yaml:"global,omitempty"`
	// Projects maps a resolved project root to the overlay pinned for it.
	Projects map[string]string `yaml:"projects,omitempty"`
}

// OrgPinsFile returns the file that holds the pins: org-overlay-pins.yaml in
// the account's home configuration directory (branding.Config.OrgConfigDir
// below the home directory the user database records, never HOME), which
// the self-protection hook protects with the default overlay. It fails when
// the account's home directory cannot be resolved.
func OrgPinsFile() (string, error) {
	home, err := pinsHome()
	if err != nil {
		return "", fmt.Errorf("resolving the account's home directory for the org overlay pins: %w", err)
	}
	return filepath.Join(branding.Get().OrgConfigDir(home), "org-overlay-pins.yaml"), nil
}

// pinsHome returns the home directory OrgPinsFile lies below: the account's
// (accountHome). Inside a test binary, which runs the CLI with an isolated
// HOME, it is HOME, so no test reads or writes the developer's pins. A
// variable for tests.
var pinsHome = func() (string, error) {
	if testing.Testing() {
		return envHomeDir()
	}
	return accountHome()
}

// loadOrgPins reads OrgPinsFile; a missing file holds no pins.
func loadOrgPins() (orgPinsRecord, string, error) {
	file, err := OrgPinsFile()
	if err != nil {
		return orgPinsRecord{}, "", err
	}
	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return orgPinsRecord{}, file, nil
	}
	if err != nil {
		return orgPinsRecord{}, file, fmt.Errorf("reading %s: %w", file, err)
	}
	var rec orgPinsRecord
	if err := yaml.Unmarshal(data, &rec); err != nil {
		return orgPinsRecord{}, file, fmt.Errorf("parsing %s: %w", file, err)
	}
	return rec, file, nil
}

// pinKey returns the key a project's pin is stored under: its resolved root.
func pinKey(projectRoot string) (string, error) {
	key, err := fileutil.ResolvePath(projectRoot)
	if err != nil {
		return "", fmt.Errorf("resolving the project root %s: %w", projectRoot, err)
	}
	return key, nil
}

// LoadOrgConfigPin returns the pin that applies to the project at
// projectRoot: the one recorded for it, else the account-wide one, else a
// zero pin. projectRoot "" (outside a project) looks up the account-wide pin
// only.
func LoadOrgConfigPin(projectRoot string) (OrgConfigPin, error) {
	rec, _, err := loadOrgPins()
	if errors.Is(err, userhome.ErrNoAccount) {
		return OrgConfigPin{Unanchored: true}, nil
	}
	if err != nil {
		return OrgConfigPin{}, fmt.Errorf("loading the pinned org overlay: %w", err)
	}
	if projectRoot != "" {
		key, err := pinKey(projectRoot)
		if err != nil {
			return OrgConfigPin{}, err
		}
		if p, ok := rec.Projects[key]; ok {
			return OrgConfigPin{Path: p, Recorded: true}, nil
		}
	}
	if rec.Global != nil {
		return OrgConfigPin{Path: *rec.Global, Recorded: true, Global: true}, nil
	}
	return OrgConfigPin{}, nil
}

// RecordOrgConfigPin records the overlay path OrgConfigPath resolves as the
// pin for the project at projectRoot, or as the account-wide pin when
// projectRoot is "", and returns it. Only a human may call it (the defaults
// pin command is sensitive). A path the agent could write (see
// untrustedOrgConfigLocation) is refused and nothing is recorded.
func RecordOrgConfigPin(projectRoot string) (string, error) {
	path, err := resolveOrgConfig(OrgConfigPath())
	if err != nil {
		return "", err
	}
	if reason := untrustedOrgConfigLocation(path, projectRoot); reason != "" {
		return "", fmt.Errorf("not pinning the org overlay %s: %s", path, reason)
	}
	rec, file, err := loadOrgPins()
	if errors.Is(err, userhome.ErrNoAccount) {
		return "", fmt.Errorf("not pinning the org overlay: %w; with no home directory to keep a pin in, %s reads the overlay %sORG_CONFIG names without one",
			err, branding.Get().AppName, branding.Get().EnvPrefix)
	}
	if err != nil {
		return "", err
	}
	if projectRoot == "" {
		rec.Global = &path
	} else {
		key, err := pinKey(projectRoot)
		if err != nil {
			return "", err
		}
		if rec.Projects == nil {
			rec.Projects = map[string]string{}
		}
		rec.Projects[key] = path
	}
	data, err := yaml.Marshal(rec)
	if err != nil {
		return "", fmt.Errorf("rendering %s: %w", file, err)
	}
	if err := os.MkdirAll(filepath.Dir(file), fileutil.ModeDirDefault); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(file), err)
	}
	if err := fileutil.WriteFileAtomic(file, data, fileutil.ModeReadWrite); err != nil {
		return "", fmt.Errorf("writing %s: %w", file, err)
	}
	return path, nil
}

// pinnedOverlay returns the overlay pin allows: the pinned path, or without a
// pin the account's home overlay, or "" for an unanchored pin, which has
// neither.
func pinnedOverlay(pin OrgConfigPin) string {
	switch {
	case pin.Recorded:
		return pin.Path
	case pin.Unanchored:
		return ""
	}
	return defaultOrgConfigPath()
}

// describePin names the overlay pin allows and where that comes from, for
// a message.
func describePin(pin OrgConfigPin) string {
	switch {
	case pin.Unanchored:
		return describeOverlay(OrgConfigPath()) + " (named by " + branding.Get().EnvPrefix +
			"ORG_CONFIG; the account has no user database entry, so no overlay can be pinned)"
	case !pin.Recorded:
		return describeOverlay(defaultOrgConfigPath()) + " (the account's home overlay; no overlay is pinned)"
	case pin.Global:
		return describeOverlay(pin.Path) + " (pinned for the account)"
	default:
		return describeOverlay(pin.Path) + " (pinned for the project)"
	}
}

// OrgConfigDrift returns why the overlay OrgConfigPath resolves now is not
// the one the catalog may read for the project at projectRoot, given its
// pin, or "" when it is: it lies below the project or the temporary
// directory, which the agent can write, or it is not the pinned overlay (the
// account's home overlay without a pin). check and doctor report it; the
// catalog reads the pinned overlay instead (PolicyOrgConfigFile).
func OrgConfigDrift(projectRoot string, pin OrgConfigPin) string {
	resolved, err := resolveOrgConfig(OrgConfigPath())
	if err != nil {
		return err.Error()
	}
	if reason := untrustedOrgConfigLocation(resolved, projectRoot); reason != "" {
		return fmt.Sprintf("the org overlay %s %s", resolved, reason)
	}
	if pin.Unanchored {
		return ""
	}
	if !sameOverlay(resolved, pinnedOverlay(pin)) {
		return fmt.Sprintf("the org overlay resolves to %s, not %s", describeOverlay(resolved), describePin(pin))
	}
	return ""
}

// ProjectOrgConfigDrift is OrgConfigDrift for the pin that applies to the
// project at projectRoot; a pin that cannot be read is reported too.
func ProjectOrgConfigDrift(projectRoot string) string {
	pin, err := LoadOrgConfigPin(projectRoot)
	if err != nil {
		return err.Error()
	}
	return OrgConfigDrift(projectRoot, pin)
}

// ProjectOrgConfigSource describes the overlay the catalog reads for the
// project at projectRoot and why, for a report: the pinned one, or the
// account's home overlay when nothing is pinned.
func ProjectOrgConfigSource(projectRoot string) string {
	pin, err := LoadOrgConfigPin(projectRoot)
	if err != nil {
		return err.Error()
	}
	return describePin(pin)
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
	fallback := pinnedOverlay(pin)
	if fallback != "" && untrustedOrgConfigLocation(fallback, root) != "" {
		fallback = ""
	}
	hint := "; to use it, run " + PinCommandHint() + " at your own terminal"
	if pin.Unanchored {
		hint = "; to use it, move it outside the project and the temporary directory"
	}
	warnDrift.Do(func() {
		slog.Warn("ignoring the org overlay this run resolves: "+drift+hint, "using", describeOverlay(fallback))
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
// expected one, which is resolved again in case a symlink on it changed.
func sameOverlay(resolved, expected string) bool {
	again, err := resolveOrgConfig(expected)
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
