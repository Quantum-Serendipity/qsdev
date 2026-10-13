package devenv

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
	attrs, err := moduleAttrs(src)
	if err != nil {
		return nil, err
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

// moduleAttrs returns NixModuleAttrs(src), accepting a module without an
// argument set too.
func moduleAttrs(src string) (map[string]string, error) {
	attrs, err := NixModuleAttrs(src)
	if err == nil {
		return attrs, nil
	}
	if attrs, retryErr := NixModuleAttrs("{ ... }: " + src); retryErr == nil {
		return attrs, nil
	}
	return nil, err
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

// gitHooksAttr is the devenv option holding the git hooks; a hook is enabled
// by git-hooks.hooks.<id>.enable.
const (
	gitHooksAttr        = "git-hooks"
	gitHookPrefix       = gitHooksAttr + ".hooks."
	gitHookEnableSuffix = ".enable"
	unsetEnvVarsAttr    = "unsetEnvVars"
)

// ErrUnverifiableModule reports a devenv module whose security settings
// cannot be read statically: it imports or disables other modules, sets
// options through an explicit config attribute, names an attribute through
// interpolation, or computes unsetEnvVars. The generator emits none of these.
var ErrUnverifiableModule = errors.New("cannot verify the module's git hooks and stripped variables")

// unverifiableTopLevel are the module attributes that pull in, drop or
// restate settings the static reader does not see.
var unverifiableTopLevel = []string{"imports", "config", "disabledModules"}

// unsetEnvVarsTokens are the identifiers an unsetEnvVars value may use besides
// string literals, list brackets, parentheses, ++ and integer priorities: the
// generated form (options.unsetEnvVars.default ++ [ ... ]) and the module
// system's priority wrappers. Anything else (builtins.filter, a let binding,
// a variable) could drop a listed variable.
var unsetEnvVarsTokens = []string{
	"options.unsetEnvVars.default",
	"mkForce", "lib.mkForce", "mkOverride", "lib.mkOverride",
	"mkDefault", "lib.mkDefault", "mkOptionDefault", "lib.mkOptionDefault",
}

// DevenvSecurity is what devenv modules declare about the git hooks and the
// variables stripped from the shell.
type DevenvSecurity struct {
	// Hooks are the ids of the enabled git hooks, sorted.
	Hooks []string
	// UnsetVars are the variables unsetEnvVars strips, sorted.
	UnsetVars []string
	// HookSettings maps every other definition under git-hooks (a hook's
	// entry, files or excludes, or a setting for every hook such as
	// git-hooks.excludes) to its value with whitespace collapsed.
	HookSettings map[string]string
}

// SecuritySettings returns the HookSettings that can change what the hooks
// ids check: those of the hooks themselves and those that apply to every
// hook.
func (d DevenvSecurity) SecuritySettings(ids []string) map[string]string {
	out := make(map[string]string)
	for path, value := range d.HookSettings {
		if rest, ok := strings.CutPrefix(path, gitHookPrefix); ok {
			id, _, _ := strings.Cut(rest, ".")
			if !slices.Contains(ids, id) {
				continue
			}
		}
		out[path] = value
	}
	return out
}

// moduleSecurity is what one devenv module declares about the git hooks and
// the variables stripped from the shell.
type moduleSecurity struct {
	// hookEnabled maps the id of each hook whose enable the module sets to
	// whether it sets it to true.
	hookEnabled map[string]bool
	// settings are the module's other git-hooks definitions.
	settings map[string]string
	// unsetVars are the string literals of the module's unsetEnvVars value,
	// and unsetValue that value's source text ("" when it sets none).
	unsetVars  []string
	unsetValue string
}

func newModuleSecurity() moduleSecurity {
	return moduleSecurity{hookEnabled: map[string]bool{}, settings: map[string]string{}}
}

// DeclaredSecurity returns the security settings a devenv module declares:
// the ids of the git hooks it enables (git-hooks.hooks.<id>.enable set to
// true, optionally through lib.mkForce, mkDefault or mkOverride), the
// variables its unsetEnvVars strips (the double-quoted string literals of
// the value; devenv's own defaults, which the value may reference, are not
// included) and its other git-hooks settings. Commented-out definitions and
// list items do not count. A module the reader cannot verify fails with
// ErrUnverifiableModule.
func DeclaredSecurity(src string) (DevenvSecurity, error) {
	m, err := parseModuleSecurity(src)
	if err != nil {
		return DevenvSecurity{}, err
	}
	return DevenvSecurity{Hooks: enabledHooks(m.hookEnabled), UnsetVars: sortedUnique(m.unsetVars), HookSettings: m.settings}, nil
}

// ProjectDeclaredSecurity returns what the project's devenv modules declare,
// as DeclaredSecurity reads them, with devenv.local.nix applied over
// devenv.nix: a hook whose enable it sets to anything but true is disabled,
// each git-hooks setting it makes replaces devenv.nix's, and its
// unsetEnvVars adds to the list, replaces it when set through lib.mkForce or
// mkOverride, and is ignored when set through mkDefault (which devenv.nix's
// definition outranks). A missing file declares nothing, so a deleted
// devenv.nix enables no hook. A file that cannot be read, parsed or verified
// is reported in the returned error, with the settings of the rest.
func ProjectDeclaredSecurity(projectRoot string) (DevenvSecurity, error) {
	base, baseErr := readModuleSecurity(projectRoot, toolreg.DevenvNixFile)
	local, localErr := readModuleSecurity(projectRoot, DevenvLocalNixFile)
	enabled := base.hookEnabled
	for id, on := range local.hookEnabled {
		if _, set := enabled[id]; !on || !set {
			enabled[id] = on
		}
	}
	maps.Copy(base.settings, local.settings)
	unset := base.unsetVars
	switch {
	case local.unsetValue == "", nixHasFunc(local.unsetValue, "mkDefault", "mkOptionDefault"):
	case nixHasFunc(local.unsetValue, "mkForce", "mkOverride"):
		unset = local.unsetVars
	default:
		unset = append(unset, local.unsetVars...)
	}
	return DevenvSecurity{Hooks: enabledHooks(enabled), UnsetVars: sortedUnique(unset), HookSettings: base.settings},
		errors.Join(baseErr, localErr)
}

// readModuleSecurity reads and parses the devenv module name under
// projectRoot. A missing file declares nothing.
func readModuleSecurity(projectRoot, name string) (moduleSecurity, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, name))
	if errors.Is(err, os.ErrNotExist) {
		return newModuleSecurity(), nil
	}
	if err != nil {
		return newModuleSecurity(), fmt.Errorf("reading %s: %w", name, err)
	}
	m, err := parseModuleSecurity(string(data))
	if err != nil {
		return newModuleSecurity(), fmt.Errorf("parsing %s: %w", name, err)
	}
	return m, nil
}

func parseModuleSecurity(src string) (moduleSecurity, error) {
	m := newModuleSecurity()
	attrs, err := moduleAttrs(src)
	if err != nil {
		return m, err
	}
	for _, path := range slices.Sorted(maps.Keys(attrs)) {
		if err := verifiableAttrPath(path); err != nil {
			return newModuleSecurity(), err
		}
		value := attrs[path]
		if path != gitHooksAttr && !strings.HasPrefix(path, gitHooksAttr+".") {
			continue
		}
		rest, _ := strings.CutPrefix(path, gitHookPrefix)
		if id, isEnable := strings.CutSuffix(rest, gitHookEnableSuffix); rest != path && isEnable && id != "" && !strings.Contains(id, ".") {
			m.hookEnabled[id] = nixTrue(value)
			continue
		}
		if norm := strings.Join(strings.Fields(value), " "); strings.ReplaceAll(norm, " ", "") != "{}" {
			m.settings[path] = norm
		}
	}
	if value, ok := attrs[unsetEnvVarsAttr]; ok {
		if err := checkUnsetEnvVarsExpr(value); err != nil {
			return newModuleSecurity(), err
		}
		m.unsetValue = value
		if m.unsetVars, err = nixStringLiterals(value); err != nil {
			return newModuleSecurity(), err
		}
	}
	return m, nil
}

// verifiableAttrPath rejects an attribute path the static reader cannot
// account for: one of unverifiableTopLevel, or a name built by interpolation
// or a quoted name with escapes, which could spell git-hooks or
// unsetEnvVars.
func verifiableAttrPath(path string) error {
	top, _, _ := strings.Cut(path, ".")
	switch {
	case slices.Contains(unverifiableTopLevel, top):
		return fmt.Errorf("%w: it sets %s", ErrUnverifiableModule, top)
	case strings.Contains(path, "${"), strings.HasPrefix(path, `"`), strings.Contains(path, gitHooksAttr+`."`):
		return fmt.Errorf("%w: it names an attribute through interpolation or escapes", ErrUnverifiableModule)
	}
	return nil
}

// checkUnsetEnvVarsExpr rejects an unsetEnvVars value built from anything
// but string literals without interpolation, list brackets, parentheses,
// ++, integer priorities and unsetEnvVarsTokens: any other expression could
// drop a variable it lists.
func checkUnsetEnvVarsExpr(expr string) error {
	s := newNixSource(expr)
	for p := 0; ; {
		q, err := s.skipTrivia(p)
		if err != nil {
			return err
		}
		if q >= len(expr) {
			return nil
		}
		c := expr[q]
		switch {
		case c == '"':
			if p, err = s.lexString(q); err != nil {
				return err
			}
			if strings.Contains(expr[q:p], "${") {
				return fmt.Errorf("%w: unsetEnvVars interpolates a string", ErrUnverifiableModule)
			}
		case strings.HasPrefix(expr[q:], "++"):
			p = q + 2
		case strings.IndexByte("[]()", c) >= 0:
			p = q + 1
		case isNixIdentStart(c) || (c >= '0' && c <= '9'):
			p = q
			for p < len(expr) && (isNixIdentChar(expr[p]) || expr[p] == '.') {
				p++
			}
			tok := expr[q:p]
			if _, err := strconv.Atoi(tok); err != nil && !slices.Contains(unsetEnvVarsTokens, tok) {
				return fmt.Errorf("%w: unsetEnvVars uses %s", ErrUnverifiableModule, tok)
			}
		default:
			return fmt.Errorf("%w: unsetEnvVars uses %q", ErrUnverifiableModule, string(c))
		}
	}
}

// nixTrue reports whether value is the literal true, optionally wrapped in
// lib.mkForce, mkDefault or mkOverride <n>. Anything else (false, a
// condition, a reference) is not known to be true.
func nixTrue(value string) bool {
	fields := strings.Fields(strings.NewReplacer("(", " ", ")", " ").Replace(value))
	if len(fields) == 0 || fields[len(fields)-1] != "true" {
		return false
	}
	for _, f := range fields[:len(fields)-1] {
		switch strings.TrimPrefix(f, "lib.") {
		case "mkForce", "mkDefault", "mkOverride":
		default:
			if _, err := strconv.Atoi(f); err != nil {
				return false
			}
		}
	}
	return true
}

// nixHasFunc reports whether the expression value calls one of the module
// system functions names (bare or as lib.<name>).
func nixHasFunc(value string, names ...string) bool {
	for _, f := range strings.FieldsFunc(value, func(r rune) bool { return r > 0x7f || (r != '.' && !isNixIdentChar(byte(r))) }) {
		if slices.Contains(names, strings.TrimPrefix(f, "lib.")) {
			return true
		}
	}
	return false
}

// nixStringLiterals returns the decoded double-quoted string literals of the
// expression expr, skipping comments and indented strings.
func nixStringLiterals(expr string) ([]string, error) {
	s := newNixSource(expr)
	var out []string
	for p := 0; p < len(expr); {
		q, err := s.skipTrivia(p)
		if err != nil {
			return nil, err
		}
		switch {
		case q >= len(expr):
			return out, nil
		case expr[q] == '"':
			end, err := s.lexString(q)
			if err != nil {
				return nil, err
			}
			out = append(out, nixStringValue(expr[q:end]))
			p = end
		case strings.HasPrefix(expr[q:], "''"):
			if p, err = s.lexIndentedString(q); err != nil {
				return nil, err
			}
		default:
			p = q + 1
		}
	}
	return out, nil
}

// enabledHooks returns the sorted ids enabled maps to true.
func enabledHooks(enabled map[string]bool) []string {
	ids := []string{}
	for id, on := range enabled {
		if on {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

// sortedUnique returns a sorted copy of list without duplicates (never nil).
func sortedUnique(list []string) []string {
	out := append([]string{}, list...)
	slices.Sort(out)
	return slices.Compact(out)
}
