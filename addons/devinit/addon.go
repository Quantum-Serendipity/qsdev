package devinit

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"fastcat.org/go/gdev/addons"
	gdevcmd "fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"

	_ "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	_ "github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/defaults"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "devinit",
		Description: func() string { return "Development environment initialization wizard" },
	},
	Config: config{},
}

func init() {
	addon.Definition.Initialize = initialize
}

func Configure(opts ...option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	addon.RegisterIfNeeded()
}

var (
	projectProfilesOnce sync.Once
	projectProfilesReg  *ProjectProfileRegistry
	projectProfilesErr  error
)

// projectProfiles returns the project-type profile registry: the catalog's
// built-in profiles, then the embedder's (addon.Config.Profiles). It loads on
// first use rather than when the command tree is built, so the catalog is
// read only after the running command has resolved its project root and the
// root's project defaults layer applies.
func projectProfiles() (*ProjectProfileRegistry, error) {
	projectProfilesOnce.Do(func() {
		projectProfilesReg, projectProfilesErr = newProjectProfiles(addon.Config.Profiles)
	})
	return projectProfilesReg, projectProfilesErr
}

// newProjectProfiles builds a registry of the catalog's built-in profiles and
// the given embedder profiles. A built-in that fails to load or an embedder
// profile that collides with one is an error, not a skipped entry.
func newProjectProfiles(embedder map[string]Profile) (*ProjectProfileRegistry, error) {
	reg, err := loadDefaultProjectProfiles()
	if err != nil {
		return nil, fmt.Errorf("loading built-in project profiles: %w", err)
	}
	if err := registerProfiles(reg, embedder); err != nil {
		return nil, fmt.Errorf("configuring devinit profiles: %w", err)
	}
	return reg, nil
}

// registerProfiles adds the embedder-configured profiles to reg. A profile
// that cannot be registered (for example, one whose name collides with a
// built-in) is a configuration error: skipping it would make --profile silently
// resolve to a different profile.
func registerProfiles(reg *ProjectProfileRegistry, profiles map[string]Profile) error {
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(profiles)) {
		if err := reg.Register(name, profiles[name]); err != nil {
			errs = append(errs, fmt.Errorf("registering profile %q: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// initialize builds the devinit commands. It must not load the catalog (see
// projectProfiles): that waits until a command has resolved its project root.
func initialize() error {
	gdevcmd.AddConfigCommandBuilder(configShowCmd, migrateCmd)
	// instance.Main walks the finished tree; wrapping here as well keeps typo
	// rejection for tools still launched through gdev's cmd.Main.
	instance.AddCommands(cmdutil.RejectUnknownSubcommands(
		initCmd(),
		trialCmd(),
		scaffoldCmd(),
		enableCmd(),
		disableCmd(),
		statusCmd(),
		listCmd(),
		checkCmd(),
		evidenceCmd(),
		teamReportCmd(),
		repairCmd(),
		infoCmd(),
		outdatedCmd(),
		updateCmd(),
		teardownCmd(),
		containerCmd(),
		sandboxCmd(),
		selfprotectCmd(),
		enforceCmd(),
		sessionCmd(),
		policyCmd(),
		defaults.Command(),
	)...)
	return nil
}
