package claudecode

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"

	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

var addon = addons.Addon[Config]{
	Definition: addons.Definition{
		Name:        "claudecode",
		Description: func() string { return "Claude Code project configuration" },
	},
	Config: Config{},
}

func init() {
	addon.Definition.Initialize = initialize
}

func Configure(opts ...Option) {
	addon.CheckNotInitialized()
	for _, o := range opts {
		o(&addon.Config)
	}
	addon.RegisterIfNeeded()
}

func initialize() error {
	instance.AddCommands(claudeCmd())
	return nil
}
