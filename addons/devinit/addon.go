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
	profileRegistry     *ProjectProfileRegistry
	profileRegistryOnce sync.Once
)

func ensureProfileRegistry() *ProjectProfileRegistry {
	profileRegistryOnce.Do(func() {
		profileRegistry = DefaultProjectProfileRegistry()
	})
	return profileRegistry
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

func initialize() error {
	if err := registerProfiles(ensureProfileRegistry(), addon.Config.Profiles); err != nil {
		return fmt.Errorf("configuring devinit profiles: %w", err)
	}
	gdevcmd.AddConfigCommandBuilder(configShowCmd, migrateCmd)
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
