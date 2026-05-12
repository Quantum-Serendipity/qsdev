// Package modules is a convenience import that registers all ecosystem modules
// with the DefaultRegistry via their init() functions. Import this package for
// its side effect to activate all modules:
//
//	import _ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules"
package modules

import (
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/ansible"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/bazel"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/clojure"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/cpp"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dart"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/docker"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dotnet"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/elixir"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/golang"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/haskell"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/helm"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/java"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/javascript"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/lua"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/nixlang"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/perl"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/php"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/powershell"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/python"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rlang"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/ruby"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rust"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/scala"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/shell"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/swift"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/terraform"
	_ "fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/zig"
)
