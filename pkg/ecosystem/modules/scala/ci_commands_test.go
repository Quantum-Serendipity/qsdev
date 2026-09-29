package scala_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCICommands_GatedOnSbt checks the sbt plugin tasks are emitted only for
// sbt builds (F436: they were emitted for Mill projects too).
func TestCICommands_GatedOnSbt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buildTool string
		want      []string
	}{
		{buildTool: "sbt", want: []string{"sbt-dependency-lock-check", "sbt-dependency-check"}},
		{buildTool: "", want: []string{"sbt-dependency-lock-check", "sbt-dependency-check"}},
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

// TestCICommands_LoadSecurityPlugins runs each sbt CI command with a stub sbt
// that prints the plugin file it is given: the tasks exist only with the
// security plugins loaded, and sbt never loaded the generated (and
// gitignored) plugins file (F436). The scan must also fail on findings, which
// sbt-dependency-check never does by default.
func TestCICommands_LoadSecurityPlugins(t *testing.T) {
	t.Parallel()

	generated := string(newModule().SecurityConfigs(ecosystem.ModuleConfig{})[0].Content)
	var plugins []string
	for _, line := range strings.Split(generated, "\n") {
		if strings.HasPrefix(line, "addSbtPlugin(") {
			plugins = append(plugins, line)
		}
	}
	if len(plugins) != 2 {
		t.Fatalf("generated plugins file has %d addSbtPlugin lines, want 2", len(plugins))
	}

	want := map[string]struct {
		phase ecosystem.CIPhase
		args  string
	}{
		"sbt-dependency-lock-check": {ecosystem.CIPhaseInstall, " dependencyLockCheck"},
		"sbt-dependency-check":      {ecosystem.CIPhaseScan, " set Global / dependencyCheckFailBuildOnCVSS := 7 dependencyCheck"},
	}
	callRe := regexp.MustCompile(`^sbt --addPluginSbtFile=(/\S+-security-plugins\.sbt)(.*)$`)
	pluginFile := `for a; do case $a in --addPluginSbtFile=*) cat "${a#*=}";; esac; done`

	for _, c := range newModule().CICommands(ecosystem.ModuleConfig{}) {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			w, ok := want[c.Name]
			if !ok {
				t.Fatalf("unexpected CI command %s", c.Name)
			}
			if c.Phase != w.phase {
				t.Errorf("phase = %v, want %v", c.Phase, w.phase)
			}
			res := shelltest.Run(t, t.TempDir(), c.Command, map[string]shelltest.Stub{"sbt": {Script: pluginFile}})
			if res.Exit != 0 || len(res.Calls) != 1 {
				t.Fatalf("exit = %d, calls = %q; output:\n%s", res.Exit, res.Calls, res.Output)
			}
			m := callRe.FindStringSubmatch(res.Calls[0])
			if m == nil || m[2] != w.args {
				t.Errorf("sbt call = %q, want the plugin file and args %q", res.Calls[0], w.args)
			}
			if got := strings.TrimSpace(res.Output); got != strings.Join(plugins, "\n") {
				t.Errorf("plugin file sbt loaded =\n%s\nwant\n%s", got, strings.Join(plugins, "\n"))
			}
		})
	}
}
