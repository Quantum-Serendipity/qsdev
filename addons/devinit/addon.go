package devinit

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"

	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
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
		_ = profileRegistry.Register(name, p)
	}
	instance.AddCommands(
		initCmd(),
		enableCmd(),
		disableCmd(),
		statusCmd(),
		listCmd(),
		configCmd(),
		checkCmd(),
	)
	return nil
}
