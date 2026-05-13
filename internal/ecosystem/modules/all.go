// Package modules is a convenience import that registers all ecosystem modules
// with the DefaultRegistry via their init() functions. Import this package for
// its side effect to activate all modules:
//
//	import _ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules"
package modules

import (
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/ansible"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/bazel"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/clojure"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/cpp"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dart"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/docker"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dotnet"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/elixir"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/haskell"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/helm"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/java"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/lua"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/nixlang"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/perl"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/php"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/powershell"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rlang"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/ruby"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rust"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/scala"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/shell"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/swift"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/terraform"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/zig"
)
