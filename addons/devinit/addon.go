package devinit

import (
	"log/slog"

	"fastcat.org/go/gdev/addons"
	gdevcmd "fastcat.org/go/gdev/cmd"
	"fastcat.org/go/gdev/instance"

	_ "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	_ "github.com/Quantum-Serendipity/qsdev/addons/devenv"
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

var profileRegistry *ProjectProfileRegistry

func initialize() error {
	profileRegistry = DefaultProjectProfileRegistry()
	for name, p := range addon.Config.Profiles {
		if err := profileRegistry.Register(name, p); err != nil {
			slog.Warn("failed to register profile", "name", name, "error", err)
		}
	}
	gdevcmd.AddConfigCommandBuilder(configShowCmd, migrateCmd)
	instance.AddCommands(
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
	)
	return nil
}
