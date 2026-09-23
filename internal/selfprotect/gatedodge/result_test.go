package gatedodge

import (
	"strings"
	"testing"
)

func TestResultRuleFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"/p/.npmrc", "GD-004"},
		{".npmrc", "GD-004"},
		{"/p/.NPMRC", "GD-004"},
		{"/p/.Pre-Commit-Config.yaml", "GD-003"},
		{"/p/.pre-commit-config.yaml", "GD-003"},
		{"/p/pnpm-workspace.yaml", "GD-004"},
		{"/p/.yarnrc.yml", "GD-004"},
		{"/p/.yarnrc", "GD-004"},
		{"/p/bunfig.toml", "GD-004"},
		{"/p/x.npmrc", ""},
		{"/p/.qsdev.yaml", ""},
		{"/p/main.go", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := ""
			if r := ResultRuleFor(tt.path); r != nil {
				got = r.ID
			}
			if got != tt.want {
				t.Errorf("ResultRuleFor(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestResultRule_Npmrc verifies that deleting or neutralizing ignore-scripts —
// which Detect cannot see, since it only scans introduced text — is blocked.
func TestResultRule_Npmrc(t *testing.T) {
	t.Parallel()

	const guarded = "save-exact=true\nignore-scripts=true\n"
	tests := []struct {
		name    string
		before  string
		after   string
		blocked bool
	}{
		{"line deleted", guarded, "save-exact=true\n", true},
		{"file emptied", guarded, "", true},
		{"set to 0", guarded, "ignore-scripts=0\n", true},
		{"set to false with spaces", "", "ignore-scripts = false\n", true},
		{"new file disables", "", "ignore-scripts=0\n", true},
		{"commented out", guarded, "; ignore-scripts=true\n", true},
		{"later assignment wins", guarded, "ignore-scripts=true\nignore-scripts=no\n", true},

		{"unchanged", guarded, guarded, false},
		{"other line edited", guarded, "save-exact=false\nignore-scripts=true\n", false},
		{"quoted true", guarded, "ignore-scripts=\"true\"\n", false},
		{"numeric true", guarded, "ignore-scripts=1\n", false},
		{"bare key is true", guarded, "ignore-scripts\n", false},
		{"min-release-age removed", "ignore-scripts=true\nmin-release-age=3\n", "ignore-scripts=true\n", true},
		{"min-release-age zeroed", "ignore-scripts=true\nmin-release-age=3\n", "ignore-scripts=true\nmin-release-age=0\n", true},
		{"min-release-age raised", "ignore-scripts=true\nmin-release-age=3\n", "ignore-scripts=true\nmin-release-age=7\n", false},
		{"never guarded", "save-exact=true\n", "", false},
		{"adds the guard", "", guarded, false},
		// The registry qsdev set (the organization's proxy) must stay.
		{"registry changed", "registry=https://proxy\n" + guarded, "registry=https://evil\n" + guarded, true},
		{"registry removed", "registry=https://proxy\n" + guarded, guarded, true},
		{"registry requoted", "registry=https://proxy\n" + guarded, "registry=\"https://proxy\"\n" + guarded, false},
		{"registry added", guarded, "registry=https://proxy\n" + guarded, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := ResultRuleFor(".npmrc").Check(tt.before, tt.after)
			if blocked != tt.blocked {
				t.Errorf("Check(%q -> %q) = (%v, %q), want blocked %v", tt.before, tt.after, blocked, reason, tt.blocked)
			}
		})
	}
}

// TestResultRule_Precommit verifies that removing configured hooks is blocked
// however it is expressed (emptying repos, deleting one hook, emptying the
// file), while edits that keep every hook pass.
func TestResultRule_Precommit(t *testing.T) {
	t.Parallel()

	const guarded = `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.8
    hooks:
      - id: ripsecrets
  - repo: local
    hooks:
      - id: golangci-lint
        entry: golangci-lint run
`
	tests := []struct {
		name    string
		before  string
		after   string
		blocked bool
	}{
		{"repos emptied", guarded, "repos: []\n", true},
		{"file emptied", guarded, "", true},
		{"one hook removed", guarded, `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.8
    hooks:
      - id: ripsecrets
`, true},
		{"hook moved to a dummy local repo", guarded, `repos:
  - repo: local
    hooks:
      - id: ripsecrets
      - id: golangci-lint
`, true},
		{"unparseable result", guarded, "repos: [\n", true},

		{"unchanged", guarded, guarded, false},
		{"rev bumped and hook added", guarded, `repos:
  - repo: https://github.com/sirwart/ripsecrets
    rev: v0.1.9
    hooks:
      - id: ripsecrets
  - repo: local
    hooks:
      - id: golangci-lint
        entry: golangci-lint run ./...
      - id: go-vet
`, false},
		{"new config", "", guarded, false},
		{"nothing to remove", "repos: []\n", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := ResultRuleFor(".pre-commit-config.yaml").Check(tt.before, tt.after)
			if blocked != tt.blocked {
				t.Errorf("Check = (%v, %q), want blocked %v", blocked, reason, tt.blocked)
			}
		})
	}
}

// TestResultRule_JSPackageManagerConfigs verifies the pnpm, Yarn and Bun
// hardening files cannot be weakened by removing a setting, flipping it, or
// lowering the release-age gate, while unrelated edits pass.
func TestResultRule_JSPackageManagerConfigs(t *testing.T) {
	t.Parallel()

	const pnpm = "packages:\n  - app\nstrictDepBuilds: true\nminimumReleaseAge: 4320\ntrustPolicy: no-downgrade\nblockExoticSubdeps: true\n"
	const yarn = "enableImmutableInstalls: true\nenableHardenedMode: true\nenableScripts: false\nnpmMinimalAgeGate: 7d\n"
	const yarnClassic = "# hardened\nignore-scripts true\n"
	const bun = "[install]\nminimumReleaseAge = 604800\n"

	tests := []struct {
		name    string
		file    string
		before  string
		after   string
		blocked bool
	}{
		{"pnpm strictDepBuilds false", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "strictDepBuilds: true", "strictDepBuilds: false", 1), true},
		{"pnpm strictDepBuilds removed", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "strictDepBuilds: true\n", "", 1), true},
		{"pnpm allow all builds", "pnpm-workspace.yaml", pnpm, pnpm + "dangerouslyAllowAllBuilds: true\n", true},
		{"pnpm allow all builds in new file", "pnpm-workspace.yaml", "", "dangerouslyAllowAllBuilds: true\n", true},
		{"pnpm age zeroed", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "4320", "0", 1), true},
		{"pnpm age removed", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "minimumReleaseAge: 4320\n", "", 1), true},
		{"pnpm build approved", "pnpm-workspace.yaml", pnpm, pnpm + "onlyBuiltDependencies:\n  - esbuild\n", true},
		{"pnpm allowBuilds approved", "pnpm-workspace.yaml", pnpm, pnpm + "allowBuilds:\n  esbuild: true\n", true},
		{"pnpm file emptied", "pnpm-workspace.yaml", pnpm, "", true},
		{"pnpm unparseable", "pnpm-workspace.yaml", pnpm, "strictDepBuilds: [\n", true},
		{"pnpm package added", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "  - app\n", "  - app\n  - lib\n", 1), false},
		{"pnpm age raised", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "4320", "10080", 1), false},
		{"pnpm build denied explicitly", "pnpm-workspace.yaml", pnpm, pnpm + "allowBuilds:\n  esbuild: false\n", false},
		{"pnpm age exclusion added", "pnpm-workspace.yaml", pnpm, pnpm + "minimumReleaseAgeExclude:\n  - evil\n", true},
		{"pnpm trust policy removed", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "trustPolicy: no-downgrade\n", "", 1), true},
		{"pnpm trust policy changed", "pnpm-workspace.yaml", pnpm, strings.Replace(pnpm, "no-downgrade", "off", 1), true},
		{"pnpm registry changed", "pnpm-workspace.yaml", "npmRegistryServer: https://proxy\n" + pnpm, "npmRegistryServer: https://evil\n" + pnpm, true},

		{"yarn scripts enabled", ".yarnrc.yml", yarn, strings.Replace(yarn, "enableScripts: false", "enableScripts: true", 1), true},
		{"yarn scripts line dropped", ".yarnrc.yml", yarn, strings.Replace(yarn, "enableScripts: false\n", "", 1), true},
		{"yarn age gate zeroed", ".yarnrc.yml", yarn, strings.Replace(yarn, "7d", "0", 1), true},
		{"yarn age gate lowered", ".yarnrc.yml", yarn, strings.Replace(yarn, "7d", "1d", 1), true},
		{"yarn hardened mode off", ".yarnrc.yml", yarn, strings.Replace(yarn, "enableHardenedMode: true", "enableHardenedMode: false", 1), true},
		{"yarn new file enables scripts", ".yarnrc.yml", "", "enableScripts: true\n", true},
		{"yarn age gate raised", ".yarnrc.yml", yarn, strings.Replace(yarn, "7d", "14d", 1), false},
		{"yarn unrelated key", ".yarnrc.yml", yarn, yarn + "nodeLinker: node-modules\n", false},
		{"yarn age gate bypass list grows", ".yarnrc.yml", yarn, yarn + "npmPreapprovedPackages:\n  - evil\n", true},
		{"yarn registry removed", ".yarnrc.yml", "npmRegistryServer: https://proxy\n" + yarn, yarn, true},
		{"yarn registry added", ".yarnrc.yml", yarn, "npmRegistryServer: https://proxy\n" + yarn, false},

		{"yarn classic scripts on", ".yarnrc", yarnClassic, "ignore-scripts false\n", true},
		{"yarn classic line dropped", ".yarnrc", yarnClassic, "# hardened\n", true},
		{"yarn classic quoted true", ".yarnrc", yarnClassic, "ignore-scripts \"true\"\n", false},
		{"yarn classic registry changed", ".yarnrc", "registry \"https://proxy\"\n" + yarnClassic, "registry \"https://evil\"\n" + yarnClassic, true},

		{"bun age zeroed", "bunfig.toml", bun, "[install]\nminimumReleaseAge = 0\n", true},
		{"bun age removed", "bunfig.toml", bun, "[install]\n", true},
		{"bun file emptied", "bunfig.toml", bun, "", true},
		{"bun unparseable", "bunfig.toml", bun, "[install\n", true},
		{"bun unrelated key", "bunfig.toml", bun, bun + "exact = true\n", false},
		{"bun age exclusion added", "bunfig.toml", bun, bun + "minimumReleaseAgeExcludes = [\"evil\"]\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			blocked, reason := ResultRuleFor(tt.file).Check(tt.before, tt.after)
			if blocked != tt.blocked {
				t.Errorf("Check(%q -> %q) = (%v, %q), want blocked %v", tt.before, tt.after, blocked, reason, tt.blocked)
			}
		})
	}
}

// TestConfigCommandTarget verifies that package-manager commands which rewrite
// a guarded project config without naming it are recognised, while reads and
// user-level writes are not.
func TestConfigCommandTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"yarn", []string{"config", "set", "enableScripts", "true"}, ".yarnrc.yml"},
		{"yarn", []string{"config", "unset", "npmMinimalAgeGate"}, ".yarnrc.yml"},
		{"yarn", []string{"config", "set", "-H", "enableTelemetry", "false"}, ""},
		{"yarn", []string{"config", "get", "enableScripts"}, ""},
		{"npm", []string{"config", "set", "ignore-scripts", "false", "--location=project"}, ".npmrc"},
		{"npm", []string{"--location", "project", "config", "delete", "min-release-age"}, ".npmrc"},
		{"npm", []string{"set", "ignore-scripts=false", "-L", "project"}, ".npmrc"},
		{"npm", []string{"config", "set", "fund", "false"}, ""},
		{"npm", []string{"config", "get", "registry", "--location=project"}, ""},
		{"pnpm", []string{"config", "set", "--location", "project", "dangerouslyAllowAllBuilds", "true"}, "pnpm-workspace.yaml"},
		{"pnpm", []string{"approve-builds", "esbuild"}, "pnpm-workspace.yaml"},
		{"pnpm", []string{"config", "list"}, ""},
		{"pnpm", []string{"install"}, ""},
		{"bun", []string{"install"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name+" "+strings.Join(tt.args, " "), func(t *testing.T) {
			t.Parallel()
			if got := ConfigCommandTarget(tt.name, tt.args); got != tt.want {
				t.Errorf("ConfigCommandTarget(%q, %q) = %q, want %q", tt.name, tt.args, got, tt.want)
			}
		})
	}
}
