// Package providers imports all built-in log providers for side-effect registration.
package providers

import (
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers/devenv"
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers/nix"
	_ "github.com/Quantum-Serendipity/qsdev/internal/extlog/providers/npm"
)
