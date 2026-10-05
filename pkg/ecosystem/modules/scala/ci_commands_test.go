package scala_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestScalaCICommands checks the sbt CI steps are emitted only for sbt builds
// (F436: they were emitted for Mill projects too): the build.sbt.lock check and
// the osv-scanner scan of it, which replaced the OWASP dependency-check plugin (U10-16).
func TestScalaCICommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buildTool string
		want      []string
	}{
		{buildTool: "sbt", want: []string{"sbt-dependency-lock-check", "sbt-osv-scan"}},
		{buildTool: "", want: []string{"sbt-dependency-lock-check", "sbt-osv-scan"}},
		{buildTool: "mill"},
	}
	for _, tt := range tests {
		t.Run("build_tool="+tt.buildTool, func(t *testing.T) {
			t.Parallel()
			config := ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": tt.buildTool}}
			var got []string
			for _, c := range newModule().CICommands(config) {
				got = append(got, c.Name)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("CI commands = %v, want %v", got, tt.want)
			}
		})
	}
}

// ciCommand returns the sbt project's CI command named name.
func ciCommand(t *testing.T, name string) ecosystem.CICommand {
	t.Helper()
	for _, c := range newModule().CICommands(ecosystem.ModuleConfig{}) {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no CI command %s", name)
	return ecosystem.CICommand{}
}

// TestCICommands_LoadSecurityPlugins runs the lock check with a stub sbt that
// prints the plugin file it is given: the task exists only with the security
// plugins loaded, and sbt never loaded the generated (and gitignored) plugins
// file (F436).
func TestCICommands_LoadSecurityPlugins(t *testing.T) {
	t.Parallel()

	generated := string(newModule().SecurityConfigs(ecosystem.ModuleConfig{})[0].Content)
	var plugins []string
	for _, line := range strings.Split(generated, "\n") {
		if strings.HasPrefix(line, "addSbtPlugin(") {
			plugins = append(plugins, line)
		}
	}
	if len(plugins) != 1 || !strings.Contains(plugins[0], "sbt-dependency-lock") {
		t.Fatalf("generated plugins file addSbtPlugin lines = %q, want only sbt-dependency-lock", plugins)
	}

	c := ciCommand(t, "sbt-dependency-lock-check")
	if c.Phase != ecosystem.CIPhaseInstall {
		t.Errorf("phase = %v, want %v", c.Phase, ecosystem.CIPhaseInstall)
	}
	callRe := regexp.MustCompile(`^sbt --addPluginSbtFile=(/\S+-security-plugins\.sbt)(.*)$`)
	pluginFile := `for a; do case $a in --addPluginSbtFile=*) cat "${a#*=}";; esac; done`
	res := shelltest.Run(t, t.TempDir(), c.Command, map[string]shelltest.Stub{"sbt": {Script: pluginFile}})
	if res.Exit != 0 || len(res.Calls) != 1 {
		t.Fatalf("exit = %d, calls = %q; output:\n%s", res.Exit, res.Calls, res.Output)
	}
	if m := callRe.FindStringSubmatch(res.Calls[0]); m == nil || m[2] != " dependencyLockCheck" {
		t.Errorf("sbt call = %q, want the plugin file and dependencyLockCheck", res.Calls[0])
	}
	if got := strings.TrimSpace(res.Output); got != strings.Join(plugins, "\n") {
		t.Errorf("plugin file sbt loaded =\n%s\nwant\n%s", got, strings.Join(plugins, "\n"))
	}
}

// osvPackage is one entry of osv-scanner's custom lockfile format.
type osvPackage struct {
	Package struct {
		Ecosystem string `json:"ecosystem"`
		Name      string `json:"name"`
		Version   string `json:"version"`
	} `json:"package"`
}

// osvLockfile is osv-scanner's custom lockfile format
// (osv-scanner -L osv-scanner:<path>).
type osvLockfile struct {
	Results []struct {
		Packages []osvPackage `json:"packages"`
	} `json:"results"`
}

// scanStub stands in for osv-scanner: it prints the custom lockfile it was
// given, so the test sees exactly what osv-scanner would scan.
const scanStub = `for a; do case $a in osv-scanner:*) cat "${a#osv-scanner:}";; esac; done`

// runOSVScan runs the sbt-osv-scan step in dir with the real jq and a stub
// osv-scanner, skipping when jq is not installed.
func runOSVScan(t *testing.T, dir string) shelltest.Result {
	t.Helper()
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("jq not on PATH")
	}
	c := ciCommand(t, "sbt-osv-scan")
	if c.Phase != ecosystem.CIPhaseScan {
		t.Errorf("phase = %v, want %v", c.Phase, ecosystem.CIPhaseScan)
	}
	return shelltest.Run(t, dir, c.Command, map[string]shelltest.Stub{"osv-scanner": {Script: scanStub}})
}

// TestSbtLockToOSV runs the sbt-osv-scan step over the build.sbt.lock
// fixture, in the root project and a subproject (sbt-dependency-lock writes
// one per project), and checks the custom lockfile osv-scanner is given
// lists every locked dependency once as a Maven "org:name" package.
func TestSbtLockToOSV(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile(filepath.Join("testdata", "build.sbt.lock"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, p := range []string{"build.sbt.lock", filepath.Join("core", "build.sbt.lock")} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, p), fixture, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	res := runOSVScan(t, dir)
	if res.Exit != 0 || len(res.Calls) != 1 {
		t.Fatalf("exit = %d, calls = %q; output:\n%s", res.Exit, res.Calls, res.Output)
	}
	if !regexp.MustCompile(`^osv-scanner scan source -L osv-scanner:/\S+$`).MatchString(res.Calls[0]) {
		t.Errorf("osv-scanner call = %q, want scan source -L osv-scanner:<file>", res.Calls[0])
	}
	var lock osvLockfile
	if err := json.Unmarshal([]byte(res.Output), &lock); err != nil {
		t.Fatalf("custom lockfile is not JSON: %v\n%s", err, res.Output)
	}
	if len(lock.Results) != 1 {
		t.Fatalf("custom lockfile has %d results, want 1", len(lock.Results))
	}
	var got []string
	for _, p := range lock.Results[0].Packages {
		got = append(got, p.Package.Ecosystem+" "+p.Package.Name+" "+p.Package.Version)
	}
	slices.Sort(got)
	want := []string{
		"Maven org.apache.commons:commons-lang3 3.9",
		"Maven org.scala-lang:scala-library 2.13.12",
		"Maven org.typelevel:cats-core_2.13 2.10.0",
	}
	if !slices.Equal(got, want) {
		t.Errorf("packages =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestSbtOSVScan_NoLockFails checks a project with no build.sbt.lock fails the
// scan instead of passing with nothing scanned.
func TestSbtOSVScan_NoLockFails(t *testing.T) {
	t.Parallel()

	res := runOSVScan(t, t.TempDir())
	if res.Exit == 0 || len(res.Calls) != 0 {
		t.Errorf("exit = %d, calls = %q, want a failure before osv-scanner runs; output:\n%s", res.Exit, res.Calls, res.Output)
	}
	if !strings.Contains(res.Output, "build.sbt.lock") {
		t.Errorf("output does not name build.sbt.lock:\n%s", res.Output)
	}
}

// TestNoSbtDependencyCheck checks neither the generated plugin file nor any
// CI step uses the OWASP dependency-check sbt plugin (U10-16).
func TestNoSbtDependencyCheck(t *testing.T) {
	t.Parallel()

	var texts []string
	for _, f := range newModule().SecurityConfigs(ecosystem.ModuleConfig{RegistryProxy: "https://proxy.example.com"}) {
		texts = append(texts, string(f.Content))
	}
	for _, c := range newModule().CICommands(ecosystem.ModuleConfig{}) {
		texts = append(texts, c.Name, c.Command, c.Description)
	}
	for _, s := range texts {
		if strings.Contains(s, "dependency-check") || strings.Contains(s, "dependencyCheck") {
			t.Errorf("generated text mentions the dependency-check plugin:\n%s", s)
		}
	}
}

// TestScalaProvisionsOSVScanner checks sbt projects get osv-scanner, which
// the sbt-osv-scan step runs, and Mill projects (with no scan) do not.
func TestScalaProvisionsOSVScanner(t *testing.T) {
	t.Parallel()

	var p ecosystem.PackageProvider = newModule()
	tests := []struct {
		buildTool string
		want      bool
	}{
		{buildTool: "sbt", want: true},
		{buildTool: "", want: true},
		{buildTool: "mill", want: false},
	}
	for _, tt := range tests {
		config := ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": tt.buildTool}}
		if got := slices.Contains(p.DevenvPackages(config), "osv-scanner"); got != tt.want {
			t.Errorf("build_tool=%q: provisions osv-scanner = %v, want %v", tt.buildTool, got, tt.want)
		}
	}
}
