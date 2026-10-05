package modules

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// The tables below are facts about devenv and nixpkgs that no production code
// consumes, so they live here as their single copy. When the catalog gains
// per-package binary metadata, packageBinaryAliases moves there.

// coreutilsBinaries are the programs pkgs.coreutils puts on PATH.
var coreutilsBinaries = []string{
	"[", "b2sum", "base32", "base64", "basename", "basenc", "cat", "chcon", "chgrp",
	"chmod", "chown", "chroot", "cksum", "comm", "cp", "csplit", "cut", "date", "dd",
	"df", "dir", "dircolors", "dirname", "du", "echo", "env", "expand", "expr",
	"factor", "false", "fmt", "fold", "groups", "head", "hostid", "id", "install",
	"join", "kill", "link", "ln", "logname", "ls", "md5sum", "mkdir", "mkfifo",
	"mknod", "mktemp", "mv", "nice", "nl", "nohup", "nproc", "numfmt", "od", "paste",
	"pathchk", "pinky", "pr", "printenv", "printf", "ptx", "pwd", "readlink",
	"realpath", "rm", "rmdir", "runcon", "seq", "sha1sum", "sha224sum", "sha256sum",
	"sha384sum", "sha512sum", "shred", "shuf", "sleep", "sort", "split", "stat",
	"stdbuf", "stty", "sum", "sync", "tac", "tail", "tee", "test", "timeout", "touch",
	"tr", "true", "truncate", "tsort", "tty", "uname", "unexpand", "uniq", "unlink",
	"users", "vdir", "wc", "who", "whoami", "yes",
}

// stdenvBinaries are the programs of the stdenv initialPath (coreutils,
// findutils, diffutils, gnused, gnugrep, gawk, gnutar, gzip, bzip2, gnumake,
// bash, patch, xz, file), which the devenv shell puts on PATH whatever the
// project configures.
var stdenvBinaries = slices.Concat(coreutilsBinaries, []string{
	"bash", "sh", "find", "xargs", "diff", "cmp", "sdiff", "diff3", "sed", "grep",
	"egrep", "fgrep", "awk", "gawk", "tar", "gzip", "gunzip", "zcat", "bzip2",
	"bunzip2", "make", "patch", "xz", "unxz", "xzcat", "file",
})

// devenvLanguageBinaries maps each devenv language option a module fragment
// enables ("go" for languages.go.enable, "java.maven" for
// languages.java.maven.enable) to the programs it puts on PATH. Every option
// a fragment enables must be listed, nil when it adds no program.
var devenvLanguageBinaries = map[string][]string{
	"clojure":                {"clojure", "clj"},
	"cplusplus":              {"cmake", "ccache", "clangd", "clang-format"},
	"dart":                   {"dart"},
	"dotnet":                 {"dotnet"},
	"elixir":                 {"elixir", "elixirc", "iex", "mix"},
	"go":                     {"go", "gofmt"},
	"haskell":                {"ghc", "ghci", "runghc", "cabal", "hpack"},
	"haskell.stack":          {"stack"},
	"java":                   {"java", "javac", "jar"},
	"java.gradle":            {"gradle"},
	"java.maven":             {"mvn"},
	"javascript":             {"node"},
	"javascript.bun":         {"bun"},
	"javascript.npm":         {"npm", "npx"},
	"javascript.pnpm":        {"pnpm"},
	"javascript.yarn":        {"yarn"},
	"kotlin":                 {"kotlin", "kotlinc"},
	"lua":                    {"lua"},
	"nix":                    nil,
	"opentofu":               {"tofu"},
	"perl":                   {"perl"},
	"php":                    {"php", "composer"},
	"python":                 {"python", "python3"},
	"python.poetry":          {"poetry"},
	"python.poetry.activate": nil,
	"python.poetry.install":  nil,
	"python.uv":              {"uv"},
	"python.venv":            {"pip", "pip3"},
	"r":                      {"R", "Rscript"},
	"ruby":                   {"ruby", "gem", "irb"},
	"ruby.bundler":           {"bundle", "bundler"},
	"rust":                   {"cargo", "rustc", "rustfmt", "cargo-clippy"},
	"scala":                  {"scala", "scalac"},
	"scala.mill":             {"mill"},
	"scala.sbt":              {"sbt"},
	"swift":                  {"swift", "swiftc"},
	"terraform":              {"terraform"},
	"typescript":             {"tsc"},
	"zig":                    {"zig"},
}

// packageBinaryAliases lists the programs of the nixpkgs attributes whose
// binary is not their last attribute segment, lowercased (see
// packageBinaries). A full attribute path wins over its last segment.
var packageBinaryAliases = map[string][]string{
	"ansible":                {"ansible", "ansible-galaxy", "ansible-playbook"},
	"awscli2":                {"aws"},
	"azure-cli":              {"az"},
	"coreutils":              coreutilsBinaries,
	"gnumake":                {"make"},
	"go-tools":               {"staticcheck"},
	"google-cloud-sdk":       {"gcloud"},
	"kubernetes-helm":        {"helm"},
	"leiningen":              {"lein"},
	"perlPackages.CPANAudit": {"cpan-audit"},
	"powershell":             {"pwsh"},
}

// versionSuffix matches a nixpkgs version suffix such as bazel_8's.
var versionSuffix = regexp.MustCompile(`_[0-9]+$`)

// packageBinaries returns the programs the nixpkgs attribute attr (with or
// without a pkgs. prefix) puts on PATH.
func packageBinaries(attr string) []string {
	attr = strings.TrimPrefix(attr, "pkgs.")
	if bins, ok := packageBinaryAliases[attr]; ok {
		return bins
	}
	last := attr[strings.LastIndex(attr, ".")+1:]
	if bins, ok := packageBinaryAliases[last]; ok {
		return bins
	}
	return []string{versionSuffix.ReplaceAllString(strings.ToLower(last), "")}
}

// pkgsAttrRef matches a pkgs.<attr> reference in a Nix expression.
var pkgsAttrRef = regexp.MustCompile(`\bpkgs\.([A-Za-z_][A-Za-z0-9_'-]*(?:\.[A-Za-z_][A-Za-z0-9_'-]*)*)`)

// exprPackageAttrs returns the nixpkgs attributes a package expression such
// as `(pkgs.leiningen.override { ... })` builds, without the override call.
func exprPackageAttrs(expr string) []string {
	var attrs []string
	for _, m := range pkgsAttrRef.FindAllStringSubmatch(expr, -1) {
		attr := m[1]
		for _, fn := range []string{".overrideAttrs", ".override"} {
			attr = strings.TrimSuffix(attr, fn)
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

var (
	nixBlockOpen  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*)\s*=\s*\{$`)
	nixBlockClose = regexp.MustCompile(`^\}\s*;?$`)
	nixEnableTrue = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*)\s*=\s*true\s*;$`)
)

// devenvLanguageOptions returns the devenv language options a devenv.nix
// fragment enables, as paths below languages ("go", "java.maven"). It
// accepts the inline form (`languages.java.maven.enable = true;`) and the
// block form (`languages.javascript = { npm.enable = true; };`), nested
// to any depth. Braces of any other construct are tracked so a block outside
// languages contributes nothing.
func devenvLanguageOptions(fragment string) []string {
	var (
		stack []string // attribute path segments; "" marks a non-attribute block
		opts  []string
	)
	for line := range strings.SplitSeq(fragment, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := nixBlockOpen.FindStringSubmatch(line); m != nil {
			stack = append(stack, m[1])
			continue
		}
		if nixBlockClose.MatchString(line) {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if m := nixEnableTrue.FindStringSubmatch(line); m != nil && !slices.Contains(stack, "") {
			path := strings.Join(append(slices.Clone(stack), m[1]), ".")
			if opt, ok := strings.CutPrefix(path, "languages."); ok {
				if opt, ok = strings.CutSuffix(opt, ".enable"); ok {
					opts = append(opts, opt)
				}
			}
			continue
		}
		for range strings.Count(line, "{") - strings.Count(line, "}") {
			stack = append(stack, "")
		}
		for range strings.Count(line, "}") - strings.Count(line, "{") {
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return opts
}

// languageBinaries returns the programs of every devenv language option the
// fragment enables. An option devenvLanguageBinaries does not list is an
// error, so a new language cannot pass the invariant unexamined.
func languageBinaries(fragment string) ([]string, error) {
	var bins []string
	for _, opt := range devenvLanguageOptions(fragment) {
		b, ok := devenvLanguageBinaries[opt]
		if !ok {
			return nil, fmt.Errorf("devenv language option languages.%s is not in devenvLanguageBinaries: add its binaries", opt)
		}
		bins = append(bins, b...)
	}
	return bins, nil
}

// baseProvidedBinaries returns the programs every devenv shell has: the
// stdenv initialPath and the catalog base packages the generator adds.
func baseProvidedBinaries(t *testing.T) map[string]bool {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("loading the catalog: %v", err)
	}
	provided := make(map[string]bool)
	for _, b := range stdenvBinaries {
		provided[b] = true
	}
	for _, attr := range cat.BasePackages() {
		for _, b := range packageBinaries(attr) {
			provided[b] = true
		}
	}
	return provided
}

// moduleProvidedBinaries returns base plus the programs mod provisions for
// cfg whatever the hook tier: its devenv language options, DevenvPackages
// and DevenvPackageExprs. Hook NixPackages are deliberately left out: the
// generator adds them only when the hook runs at the project's tier, and a
// built-in hook's package never reaches PATH.
func moduleProvidedBinaries(base map[string]bool, mod ecosystem.EcosystemModule, cfg ecosystem.ModuleConfig) (map[string]bool, error) {
	provided := make(map[string]bool, len(base))
	for b := range base {
		provided[b] = true
	}
	frag, err := mod.DevenvNixFragment(cfg)
	if err != nil {
		return nil, fmt.Errorf("DevenvNixFragment: %w", err)
	}
	bins, err := languageBinaries(frag)
	if err != nil {
		return nil, err
	}
	if pp, ok := mod.(ecosystem.PackageProvider); ok {
		for _, attr := range pp.DevenvPackages(cfg) {
			bins = append(bins, packageBinaries(attr)...)
		}
	}
	if ep, ok := mod.(ecosystem.PackageExprProvider); ok {
		for _, expr := range ep.DevenvPackageExprs(cfg) {
			for _, attr := range exprPackageAttrs(expr) {
				bins = append(bins, packageBinaries(attr)...)
			}
		}
	}
	for _, b := range bins {
		provided[b] = true
	}
	return provided, nil
}

// ciPrograms returns the programs a CI command looks up on PATH, using the
// shared shell parser: every simple command, loop bodies and command
// substitutions included, its program found through env/timeout-style
// wrappers, `sh -c` scripts and find -exec commands. Builtins, functions the
// script defines and programs named by a path are skipped. A command that
// does not parse, or whose program word is an expansion, is an error.
func ciPrograms(script string) ([]string, error) {
	cmds, err := cmdscan.Parse(script)
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", script, err)
	}
	functions := make(map[string]bool)
	for _, c := range cmds {
		if c.Defines != "" {
			functions[c.Defines] = true
		}
	}
	var progs []string
	for _, c := range cmds {
		if c.NameHasExpansion {
			return nil, fmt.Errorf("a command word in %q is an expansion, so its program cannot be checked", script)
		}
		if c.Name == "" {
			continue
		}
		p, err := wordsPrograms(append([]string{c.Name}, c.Args...), functions)
		if err != nil {
			return nil, err
		}
		progs = append(progs, p...)
	}
	return progs, nil
}

// wordsPrograms returns the programs one simple command's words run (see
// ciPrograms).
func wordsPrograms(words []string, functions map[string]bool) ([]string, error) {
	var progs []string
	add := func(name string, shellRuns bool) {
		switch {
		case shellRuns && (cmdscan.IsShellBuiltin(name) || functions[name]):
		case strings.Contains(name, "/"): // a path is not looked up on PATH
		default:
			progs = append(progs, name)
		}
	}
	add(words[0], true)
	run := cmdscan.Program(words)
	if run.CommandString {
		return nil, fmt.Errorf("%q runs a command given as one string, which cannot be checked", strings.Join(words, " "))
	}
	if run.Index > 0 {
		add(words[run.Index], run.ShellRuns)
	}
	if script, ok := cmdscan.Script(words); ok {
		inner, err := ciPrograms(script.Script)
		if err != nil {
			return nil, err
		}
		progs = append(progs, inner...)
	}
	if run.Index >= 0 && words[run.Index] == "find" {
		inner, err := findExecPrograms(words[run.Index+1:])
		if err != nil {
			return nil, err
		}
		progs = append(progs, inner...)
	}
	return progs, nil
}

// findExecPrograms returns the programs find's -exec, -execdir, -ok and
// -okdir actions run.
func findExecPrograms(args []string) ([]string, error) {
	var progs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-exec", "-execdir", "-ok", "-okdir":
		default:
			continue
		}
		end := i + 1
		for end < len(args) && args[end] != ";" && args[end] != "+" {
			end++
		}
		if end > i+1 {
			p, err := wordsPrograms(args[i+1:end], nil)
			if err != nil {
				return nil, err
			}
			progs = append(progs, p...)
		}
		i = end
	}
	return progs, nil
}

// unprovisionedCIBinaries returns the programs command runs that provided
// lacks, sorted and without duplicates.
func unprovisionedCIBinaries(command string, provided map[string]bool) ([]string, error) {
	progs, err := ciPrograms(command)
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, p := range progs {
		if !provided[p] {
			missing = append(missing, p)
		}
	}
	sort.Strings(missing)
	return slices.Compact(missing), nil
}

// ciConfig is one configuration a module is checked under.
type ciConfig struct {
	label string
	cfg   ecosystem.ModuleConfig
}

// ciProvisioningConfigs derives the configurations to check mod under from
// the module itself: the zero config, each package manager it lists, and each
// option of each select wizard field, recorded the way the wizard records it.
func ciProvisioningConfigs(mod ecosystem.EcosystemModule) []ciConfig {
	configs := []ciConfig{{label: "zero config"}}
	for _, pm := range mod.PackageManagers() {
		configs = append(configs, ciConfig{
			label: "package manager " + pm.Name,
			cfg:   ecosystem.ModuleConfig{PackageManager: pm.Name},
		})
	}
	wp, ok := mod.(ecosystem.WizardFieldProvider)
	if !ok {
		return configs
	}
	for _, field := range wp.WizardFields() {
		if field.Type != ecosystem.FieldTypeSelect {
			continue
		}
		for _, o := range field.Options {
			lc := types.LanguageChoice{Name: mod.Name()}.WithSetting(field.Key, o.Value)
			configs = append(configs, ciConfig{label: field.Key + "=" + o.Value, cfg: ecosystem.ToModuleConfig(lc)})
		}
	}
	return configs
}

// ciDetectionFixtures are projects whose CI depends on files detection
// reads; they run through the --yes path (detection, then FillDefaults).
var ciDetectionFixtures = []struct {
	name  string
	lang  string
	files map[string]string
}{
	{name: "perl cpanfile only", lang: "perl", files: map[string]string{"cpanfile": "requires 'Moo';\n"}},
	{name: "perl with snapshot", lang: "perl", files: map[string]string{"cpanfile": "requires 'Moo';\n", "cpanfile.snapshot": "# carton snapshot format: version 1.0\n"}},
	{name: "conan", lang: "cpp", files: map[string]string{"conanfile.txt": "[requires]\nzlib/1.3.1\n", "CMakeLists.txt": "project(x)\n"}},
	{name: "deps.edn without clj-watson", lang: "clojure", files: map[string]string{"deps.edn": "{:deps {org.clojure/clojure {:mvn/version \"1.12.0\"}}}\n"}},
	{name: "deps.edn with clj-watson", lang: "clojure", files: map[string]string{"deps.edn": "{:aliases {:clj-watson {:replace-deps {io.github.clj-holmes/clj-watson {:git/tag \"v6.0.0\" :git/sha \"cb02879\"}} :main-opts [\"-m\" \"clj-watson.cli\"]}}}\n"}},
	{name: "project.clj without lein-nvd", lang: "clojure", files: map[string]string{"project.clj": "(defproject x \"0.1.0\")\n"}},
	{name: "project.clj with lein-nvd", lang: "clojure", files: map[string]string{"project.clj": "(defproject x \"0.1.0\" :plugins [[lein-nvd \"2.0.0\"]])\n"}},
	{name: "mix.exs without mix_audit", lang: "elixir", files: map[string]string{"mix.exs": "defmodule X.MixProject do\nend\n", "mix.lock": "%{}\n"}},
	{name: "mix.exs with mix_audit", lang: "elixir", files: map[string]string{"mix.exs": "defmodule X.MixProject do\n  defp deps, do: [{:mix_audit, \"~> 2.1\", only: [:dev, :test], runtime: false}]\nend\n", "mix.lock": "%{}\n"}},
	{name: "cabal with freeze file", lang: "haskell", files: map[string]string{"x.cabal": "cabal-version: 3.0\nname: x\nversion: 0.1\n", "cabal.project.freeze": "active-repositories: hackage.haskell.org\nindex-state: hackage.haskell.org 2026-01-01T00:00:00Z\n"}},
}

// checkCIProvisioned reports every program of mod's CI commands under cfg
// that nothing provisions, and returns how many programs it checked.
func checkCIProvisioned(t *testing.T, base map[string]bool, mod ecosystem.EcosystemModule, c ciConfig) int {
	t.Helper()
	steps := mod.CICommands(c.cfg)
	if len(steps) == 0 {
		return 0
	}
	provided, err := moduleProvidedBinaries(base, mod, c.cfg)
	if err != nil {
		t.Errorf("%s [%s]: %v", mod.Name(), c.label, err)
		return 0
	}
	checked := 0
	for _, step := range steps {
		progs, err := ciPrograms(step.Command)
		if err != nil {
			t.Errorf("%s [%s] step %q: %v", mod.Name(), c.label, step.Name, err)
			continue
		}
		checked += len(progs)
		missing, _ := unprovisionedCIBinaries(step.Command, provided)
		for _, bin := range missing {
			t.Errorf("%s [%s] step %q runs %q, which neither its devenv language options, its DevenvPackages/DevenvPackageExprs, the base packages nor stdenv provide",
				mod.Name(), c.label, step.Name, bin)
		}
	}
	return checked
}

// TestCICommandBinariesProvisioned verifies that every program a module's CI
// commands run is on the devenv shell's PATH on a clean runner at any hook
// tier: the security-scan workflow runs them in that shell, and a missing
// program fails the step (U10-05).
func TestCICommandBinariesProvisioned(t *testing.T) {
	t.Parallel()
	base := baseProvidedBinaries(t)
	for _, mod := range ecosystem.DefaultRegistry().All() {
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			hasCI, checked := false, 0
			for _, c := range ciProvisioningConfigs(mod) {
				hasCI = hasCI || len(mod.CICommands(c.cfg)) > 0
				checked += checkCIProvisioned(t, base, mod, c)
			}
			if hasCI && checked == 0 {
				t.Errorf("%s has CI commands but no program in them was checked", mod.Name())
			}
		})
	}
	for _, fx := range ciDetectionFixtures {
		t.Run("fixture/"+fx.name, func(t *testing.T) {
			t.Parallel()
			lc, ok := languageByName(fillFromDetection(t, writeTree(t, fx.files)), fx.lang)
			if !ok {
				t.Fatalf("language %q not selected", fx.lang)
			}
			mod, ok := ecosystem.DefaultRegistry().ByName(fx.lang)
			if !ok {
				t.Fatalf("module %q not registered", fx.lang)
			}
			cfg := ecosystem.ToModuleConfig(lc)
			if len(mod.CICommands(cfg)) > 0 && checkCIProvisioned(t, base, mod, ciConfig{label: fx.name, cfg: cfg}) == 0 {
				t.Errorf("fixture %q has CI commands but no program in them was checked", fx.name)
			}
		})
	}
}

// TestUnprovisionedCIBinaries_Checker proves the checker reports the programs
// it must, so the invariant cannot pass vacuously.
func TestUnprovisionedCIBinaries_Checker(t *testing.T) {
	t.Parallel()
	provided := make(map[string]bool)
	for _, b := range stdenvBinaries {
		provided[b] = true
	}
	provided["present"] = true
	tests := []struct {
		name    string
		command string
		want    []string
	}{
		{name: "unprovisioned", command: "foo --check", want: []string{"foo"}},
		{name: "provisioned", command: "present --check"},
		{name: "env-wrapped", command: "env A=1 foo", want: []string{"foo"}},
		{name: "timeout-wrapped", command: "timeout 30 foo .", want: []string{"foo"}},
		{name: "loop body", command: "for f in ./*.x; do\n  foo \"$f\" || exit 1\ndone", want: []string{"foo"}},
		{name: "command substitution", command: "x=\"$(foo)\" && present \"$x\"", want: []string{"foo"}},
		{name: "builtins skipped", command: "cd sub && test -f x || { echo missing >&2; exit 1; }"},
		{name: "and-list", command: "present && foo && bar", want: []string{"bar", "foo"}},
		{name: "sh -c script", command: "bash -c 'foo x'", want: []string{"foo"}},
		{name: "find -exec", command: "find . -name '*.x' -exec foo {} +", want: []string{"foo"}},
		{name: "defined function skipped", command: "f() { foo; }; f", want: []string{"foo"}},
		{name: "path skipped", command: "./gradlew build"},
		{name: "pipeline", command: "present | foo --strict", want: []string{"foo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := unprovisionedCIBinaries(tt.command, provided)
			if err != nil {
				t.Fatalf("unprovisionedCIBinaries(%q): %v", tt.command, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("unprovisionedCIBinaries(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
	for _, bad := range []string{"foo &&", "$TOOL --check"} {
		if _, err := unprovisionedCIBinaries(bad, provided); err == nil {
			t.Errorf("unprovisionedCIBinaries(%q) = nil error, want one", bad)
		}
	}
}

func TestDevenvLanguageOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		fragment string
		want     []string
	}{
		{name: "inline", fragment: "  languages.go.enable = true;\n", want: []string{"go"}},
		{name: "inline sub-option", fragment: "  languages.java.enable = true;\n  languages.java.maven.enable = true;\n  languages.java.jdk.package = pkgs.jdk21;\n", want: []string{"java", "java.maven"}},
		{name: "block", fragment: "  languages.javascript = {\n    enable = true;\n    package = pkgs.nodejs_24;\n    npm.enable = true;\n  };\n", want: []string{"javascript", "javascript.npm"}},
		{name: "nested block", fragment: "  languages.python = {\n    enable = true;\n    poetry = {\n      enable = true;\n    };\n  };\n", want: []string{"python", "python.poetry"}},
		{name: "disabled", fragment: "  languages.go.enable = false;\n"},
		{name: "commented", fragment: "  # languages.go.enable = true;\n"},
		{name: "other blocks ignored", fragment: "  tasks.\"x\" = lib.mkIf config.languages.python.enable {\n    enable = true;\n  };\n  imports = [ { languages.java.jdk.package = pkgs.jdk21; } ];\n  languages.opentofu = {\n    enable = true;\n  };\n", want: []string{"opentofu"}},
		{name: "haskell stack", fragment: "  languages.haskell.enable = true;\n  languages.haskell.stack.enable = true;\n", want: []string{"haskell", "haskell.stack"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := devenvLanguageOptions(tt.fragment); !slices.Equal(got, tt.want) {
				t.Errorf("devenvLanguageOptions = %q, want %q", got, tt.want)
			}
		})
	}

	_, err := languageBinaries("  languages.cobol.enable = true;\n")
	if err == nil || !strings.Contains(err.Error(), "add its binaries") {
		t.Errorf("languageBinaries(unknown language) error = %v, want one saying to add its binaries", err)
	}
}

func TestPackageBinaries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		attr string
		want []string
	}{
		{"cppcheck", []string{"cppcheck"}},
		{"pkgs.tflint", []string{"tflint"}},
		{"perlPackages.Carton", []string{"carton"}},
		{"perlPackages.CPANAudit", []string{"cpan-audit"}},
		{"luaPackages.luarocks", []string{"luarocks"}},
		{"kubernetes-helm", []string{"helm"}},
		{"bazel_8", []string{"bazel"}},
	}
	for _, tt := range tests {
		if got := packageBinaries(tt.attr); !slices.Equal(got, tt.want) {
			t.Errorf("packageBinaries(%q) = %q, want %q", tt.attr, got, tt.want)
		}
	}
	if got, want := exprPackageAttrs("(pkgs.leiningen.override { jdk = config.languages.java.jdk.package; })"), []string{"leiningen"}; !slices.Equal(got, want) {
		t.Errorf("exprPackageAttrs = %q, want %q", got, want)
	}
}
