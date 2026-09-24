package devenv

import (
	"fastcat.org/go/gdev/addons"
	"fastcat.org/go/gdev/instance"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
)

var addon = addons.Addon[config]{
	Definition: addons.Definition{
		Name:        "devenv",
		Description: func() string { return "devenv.sh environment setup and management" },
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

func initialize() error {
	// instance.Main walks the finished tree; wrapping here as well keeps typo
	// rejection for tools still launched through gdev's cmd.Main.
	instance.AddCommands(cmdutil.RejectUnknownSubcommands(devenvCmd(), completionCmd())...)
	return nil
}
