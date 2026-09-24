package catalog

import (
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
	docsCorpus      DocsCorpusConfig

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
)

// SetProjectRoot sets the project whose defaults file
// (<root>/.qsdev/defaults.yaml, see ProjectConfigPath) Default applies. main
// calls it (via instance.UseProjectDefaults) before any command runs; it has
// no effect once Default has loaded the catalog.
func SetProjectRoot(root string) {
	mu.Lock()
	projectRootDir = root
	mu.Unlock()
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
// instead, and the error is kept for OrgOverlayError. Errors in the embedded
// or project-level catalog are still returned: the project file is policy
// the repository declares, so one that fails to parse or tries to loosen the
// defaults stops the command rather than being dropped.
func Default() (*Catalog, error) {
	defaultOnce.Do(func() {
		mu.Lock()
		root := projectRootDir
		mu.Unlock()

		var projOpts []LoadOption
		if projFile := ProjectConfigFile(root); projFile != "" {
			projOpts = append(projOpts, WithProjectConfigFile(projFile))
		}

		orgFile := OrgConfigFile()
		if orgFile == "" {
			defaultCat, defaultErr = Load(projOpts...)
			return
		}

		cat, err := Load(append([]LoadOption{WithOrgConfigFile(orgFile)}, projOpts...)...)
		if err == nil {
			defaultCat = cat
			return
		}

		// Attribute the failure: when the catalog loads without the org
		// overlay, the overlay is at fault and is skipped.
		fallback, fbErr := Load(projOpts...)
		if fbErr != nil {
			defaultErr = err
			return
		}
		mu.Lock()
		orgOverlayErr = fmt.Errorf("user defaults %s: %w", orgFile, err)
		mu.Unlock()
		slog.Warn("ignoring invalid user defaults file; using built-in defaults",
			"path", orgFile, "error", err,
			"fix", fmt.Sprintf("run '%s defaults validate', then edit or reset the file", branding.Get().AppName))
		defaultCat = fallback
	})
	return defaultCat, defaultErr
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
	mu.Unlock()
}
