package catalog

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Catalog holds all loaded configuration data. It is populated once
// at startup and is immutable thereafter.
type Catalog struct {
	tiers           TiersFile
	compliance      ComplianceFile
	projectProfiles ProjectProfilesFile
	tools           ToolsFile
	security        SecurityFile
	hookTiers       HookTiersFile
	derivations     DerivationsFile
	validation      ValidationFile
	permissionRules PermissionRulesFile
	mcpServers      map[string]MCPServerDef
	bootstrapTools  map[string]BootstrapToolDef
	mcpServe        MCPServeOptIns
	docsCorpus      DocsCorpusConfig

	// projectHooks lists the pre-commit hooks the project defaults file
	// added (see ProjectOverlayHooks); nil when no project file applied.
	projectHooks []ProjectHook

	// entryNodes holds, for a catalog parsed from a unified defaults file,
	// the source YAML node of every entry of each top-level mapping section
	// (section name -> entry name -> node). MergeCatalogs uses it to
	// deep-merge a partial overlay entry onto the base entry.
	entryNodes map[string]map[string]*yaml.Node
}

var (
	mu             sync.Mutex
	defaultOnce    sync.Once
	defaultCat     *Catalog
	defaultErr     error
	orgOverlayErr  error
	projectRootDir string
	// loaded is set when Default reads projectRootDir to load the catalog;
	// from then on the root can no longer change (see SetProjectRoot).
	loaded bool
	// rootConflictErr, once set by SetProjectRoot, is what every later
	// Default returns instead of the cached catalog.
	rootConflictErr error
)

// ErrCatalogAlreadyLoaded reports a SetProjectRoot for a different project
// after Default has loaded the catalog for another one.
var ErrCatalogAlreadyLoaded = errors.New("catalog already loaded")

// SetProjectRoot sets the project whose defaults file
// (<root>/.qsdev/defaults.yaml, see ProjectConfigFile) Default applies. The
// runtime's cobra initializer calls it with the executing command's resolved
// root (instance.Runtime.initCommand); nothing loads the catalog before that.
// Once Default has loaded, restating the same root is a no-op, and a
// different root fails closed: it returns an error wrapping
// ErrCatalogAlreadyLoaded and poisons the cache, so every later Default
// returns that error rather than serving one project's policy to another.
// ResetDefault clears it.
func SetProjectRoot(root string) error {
	mu.Lock()
	defer mu.Unlock()
	if !loaded {
		projectRootDir = root
		return nil
	}
	if root == projectRootDir {
		return nil
	}
	rootConflictErr = fmt.Errorf("setting project root %q: %w for project root %q", root, ErrCatalogAlreadyLoaded, projectRootDir)
	return rootConflictErr
}

// ProjectRoot returns the root set by SetProjectRoot, or "" when none is set.
func ProjectRoot() string {
	mu.Lock()
	defer mu.Unlock()
	return projectRootDir
}

// Default returns the lazily-initialized global catalog loaded from
// embedded defaults, with optional project and org overrides (see Load for
// their order and the project file's add-or-tighten restriction).
//
// The user-level org overlay ($QSDEV_ORG_CONFIG or
// ~/.config/qsdev/defaults.yaml) is the developer's own file. If it fails to
// parse or validate, Default does not fail: a broken overlay would otherwise
// take down every command, including the `defaults validate/edit/reset`
// commands that exist to repair it. The overlay is skipped with a warning
// instead, and the error is kept for OrgOverlayError; the CLI's root catalog
// gate then refuses every command that would generate or change something
// without it, and check fails config_catalog. Errors in the embedded
// or project-level catalog are still returned: the project file is policy
// the repository declares, so one that fails to parse, tries to loosen the
// defaults, or fails the trust rule (see ProjectConfigFile) stops the
// command rather than being dropped. After a SetProjectRoot conflict it
// returns that error (see SetProjectRoot).
func Default() (*Catalog, error) {
	defaultOnce.Do(func() {
		mu.Lock()
		root := projectRootDir
		loaded = true
		mu.Unlock()

		cat, err := loadDefault(root)
		mu.Lock()
		defaultCat, defaultErr = cat, err
		mu.Unlock()
	})
	mu.Lock()
	defer mu.Unlock()
	if rootConflictErr != nil {
		return nil, rootConflictErr
	}
	return defaultCat, defaultErr
}

// LoadError wraps an error from Default in the wording every command uses to
// tell a person the defaults catalog did not load, so the root gate and
// `check` report the failure identically.
func LoadError(err error) error {
	return fmt.Errorf("loading %s defaults: %w", branding.Get().AppName, err)
}

// ValidateCommand returns the command line that diagnoses a defaults file
// that does not load, for repair hints.
func ValidateCommand() string {
	return branding.Get().AppName + " defaults validate"
}

// loadDefault loads the catalog Default caches for the project at root: the
// embedded defaults, the project layer, and the pinned org overlay, which is
// skipped with a warning (recorded for OrgOverlayError) when it alone fails.
func loadDefault(root string) (*Catalog, error) {
	projFile, err := ProjectConfigFile(root)
	if err != nil {
		return nil, err
	}
	var projOpts []LoadOption
	if projFile != "" {
		projOpts = append(projOpts, WithProjectConfigFile(projFile))
	}

	orgFile := PolicyOrgConfigFile()
	if orgFile == "" {
		return Load(projOpts...)
	}

	cat, err := Load(append([]LoadOption{WithOrgConfigFile(orgFile)}, projOpts...)...)
	if err == nil {
		return cat, nil
	}

	// Attribute the failure: when the catalog loads without the org
	// overlay, the overlay is at fault and is skipped.
	fallback, fbErr := Load(projOpts...)
	if fbErr != nil {
		return nil, err
	}
	mu.Lock()
	orgOverlayErr = fmt.Errorf("user defaults %s: %w", orgFile, err)
	mu.Unlock()
	slog.Warn("ignoring invalid user defaults file; using built-in defaults",
		"path", orgFile, "error", err,
		"fix", fmt.Sprintf("run '%s', then edit or reset the file", ValidateCommand()))
	return fallback, nil
}

// OrgOverlayError returns the error that made Default skip the user-level
// org defaults file, or nil when that file loaded or is absent. It is only
// meaningful after Default has run.
func OrgOverlayError() error {
	mu.Lock()
	defer mu.Unlock()
	return orgOverlayErr
}

// MustDefault returns the lazily-initialized global catalog, panicking
// if loading fails. Use this in init-time accessors where returning an
// error is impractical. An invalid user-level org overlay does not panic
// (see Default); only a broken embedded or project catalog does.
func MustDefault() *Catalog {
	cat, err := Default()
	if err != nil {
		panic(fmt.Sprintf("catalog: failed to load: %v", err))
	}
	return cat
}

// ResetDefault clears the cached default catalog, forcing the next
// call to Default() to reload. Intended for testing only.
func ResetDefault() {
	mu.Lock()
	defaultOnce = sync.Once{}
	defaultCat = nil
	defaultErr = nil
	orgOverlayErr = nil
	projectRootDir = ""
	loaded = false
	rootConflictErr = nil
	mu.Unlock()
}
