// Package modules is a convenience import that registers all ecosystem modules
// with the DefaultRegistry via their init() functions. Import this package for
// its side effect to activate all modules:
//
//	import _ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
package modules

import (
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/ansible"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/bazel"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/clojure"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/container"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cpp"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/dart"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/dotnet"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/elixir"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/haskell"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/helm"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/java"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/lua"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/nixlang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/perl"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/php"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/powershell"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/rlang"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/ruby"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/rust"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/scala"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/shell"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/swift"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/zig"
)
