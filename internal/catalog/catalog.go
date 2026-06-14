package catalog

import (
	"fmt"
	"sync"
)

// Catalog holds all loaded configuration data. It is populated once
// at startup and is immutable thereafter.
type Catalog struct {
	tiers           TiersFile
	compliance      ComplianceFile
	profiles        ProfilesFile
	projectProfiles ProjectProfilesFile
	tools           ToolsFile
	security        SecurityFile
	hookTiers       HookTiersFile
	derivations     DerivationsFile
	validation      ValidationFile
	permissionRules PermissionRulesFile
	mcpServers      map[string]MCPServerDef
	docsCorpus      DocsCorpusConfig
}

var (
	mu             sync.Mutex
	defaultOnce    sync.Once
	defaultCat     *Catalog
	defaultErr     error
	projectRootDir string
)

// SetProjectRoot configures the project-level override path.
// Must be called before Default() is first accessed.
func SetProjectRoot(root string) {
	mu.Lock()
	projectRootDir = root
	mu.Unlock()
}

// Default returns the lazily-initialized global catalog loaded from
// embedded defaults, with optional org and project overrides.
func Default() (*Catalog, error) {
	defaultOnce.Do(func() {
		mu.Lock()
		root := projectRootDir
		mu.Unlock()

		var opts []LoadOption

		if orgFile := OrgConfigFile(); orgFile != "" {
			opts = append(opts, WithOrgConfigFile(orgFile))
		}
		if projFile := ProjectConfigFile(root); projFile != "" {
			opts = append(opts, WithProjectConfigFile(projFile))
		}

		defaultCat, defaultErr = Load(opts...)
	})
	return defaultCat, defaultErr
}

// MustDefault returns the lazily-initialized global catalog, panicking
// if loading fails. Use this in init-time accessors where returning an
// error is impractical.
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
	projectRootDir = ""
	mu.Unlock()
}
