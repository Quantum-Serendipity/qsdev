package devenv

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// DevenvLocalNixFile is the untracked devenv module for local customizations
// (per-developer values such as a cloud profile) that devenv merges with
// devenv.nix.
const DevenvLocalNixFile = "devenv.local.nix"

// DeclaredEnv returns the environment variables a devenv module declares,
// whether as `env.NAME = ...;` or inside an `env = { ... };` block. A plain
// double-quoted string value is returned decoded; any other value (an
// interpolated string, a function call, a reference) is returned as its
// source text, since it cannot be evaluated statically. Commented-out
// definitions are not declarations. A module without an argument set
// (`{ env.X = "y"; }`, a common devenv.local.nix shape) is accepted too.
func DeclaredEnv(src string) (map[string]string, error) {
	attrs, err := NixModuleAttrs(src)
	if err != nil {
		var retryErr error
		if attrs, retryErr = NixModuleAttrs("{ ... }: " + src); retryErr != nil {
			return nil, err
		}
	}
	env := make(map[string]string)
	for path, value := range attrs {
		name, ok := strings.CutPrefix(path, "env.")
		if !ok || name == "" || strings.Contains(name, ".") {
			continue
		}
		env[name] = nixStringValue(value)
	}
	return env, nil
}

// nixStringValue decodes value when it is a plain double-quoted Nix string
// (no interpolation) and otherwise returns it unchanged.
func nixStringValue(value string) string {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value
	}
	body := value[1 : len(value)-1]
	if strings.Contains(body, "${") {
		return value
	}
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c == '"' {
			return value // two strings joined by an operator, not one literal
		}
		if c != '\\' || i+1 >= len(body) {
			b.WriteByte(c)
			continue
		}
		i++
		switch body[i] {
		case 'n':
			b.WriteByte('\n')
		case 't':
			b.WriteByte('\t')
		case 'r':
			b.WriteByte('\r')
		default:
			b.WriteByte(body[i])
		}
	}
	return b.String()
}

// ProjectDeclaredEnv returns the environment variables the project's devenv
// modules declare: devenv.nix, then devenv.local.nix, whose definitions win.
// A missing file contributes nothing. A file that cannot be read or parsed is
// reported in the returned error while the variables of the other file are
// still returned.
func ProjectDeclaredEnv(projectRoot string) (map[string]string, error) {
	env := make(map[string]string)
	var errs []error
	for _, name := range []string{toolreg.DevenvNixFile, DevenvLocalNixFile} {
		data, err := os.ReadFile(filepath.Join(projectRoot, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("reading %s: %w", name, err))
			continue
		}
		declared, err := DeclaredEnv(string(data))
		if err != nil {
			errs = append(errs, fmt.Errorf("parsing %s: %w", name, err))
			continue
		}
		maps.Copy(env, declared)
	}
	return env, errors.Join(errs...)
}
