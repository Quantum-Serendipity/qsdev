package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// fillValue sets every field reachable from v to a non-zero value derived from
// its path, so distinct fields never share a value (a blocked MCP server name
// never matches a configured one, and no level names a real compliance level).
func fillValue(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	switch v.Kind() {
	case reflect.String:
		v.SetString(path)
	case reflect.Int:
		v.SetInt(int64(types.ConfigVersionCurrent))
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillValue(t, v.Elem(), path)
	case reflect.Struct:
		for i := range v.NumField() {
			fillValue(t, v.Field(i), path+"."+v.Type().Field(i).Name)
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fillValue(t, s.Index(0), path+"[0]")
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		key := reflect.New(v.Type().Key()).Elem()
		fillValue(t, key, path+".key")
		val := reflect.New(v.Type().Elem()).Elem()
		if val.Kind() == reflect.Interface {
			val = reflect.ValueOf(path + ".value")
		} else {
			fillValue(t, val, path+".value")
		}
		m.SetMapIndex(key, val)
		v.Set(m)
	default:
		t.Fatalf("fillValue: unhandled kind %s at %s", v.Kind(), path)
	}
}

// TestResolveConfig_KeepsEveryProjectField guards the resolver against
// dropping a .qsdev.yaml key: init, join and update now read the project
// through ResolveConfig, so a field that cloneQsdevConfig or deepMerge does
// not carry would silently vanish from generation. A new QsdevConfig field
// fails this test until the resolver merges it.
func TestResolveConfig_KeepsEveryProjectField(t *testing.T) {
	t.Parallel()
	var project types.QsdevConfig
	fillValue(t, reflect.ValueOf(&project).Elem(), "cfg")

	resolved, err := ResolveConfig(nil, nil, &project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Violations) > 0 {
		t.Fatalf("unexpected floor violations: %v", resolved.Violations)
	}
	got, want := reflect.ValueOf(*resolved.Config), reflect.ValueOf(project)
	for i := range want.NumField() {
		name := want.Type().Field(i).Name
		if !reflect.DeepEqual(got.Field(i).Interface(), want.Field(i).Interface()) {
			t.Errorf("%s not carried through ResolveConfig:\n got  %#v\n want %#v", name, got.Field(i).Interface(), want.Field(i).Interface())
		}
	}
}

func writeProjectFiles(t *testing.T, project, local string) string {
	t.Helper()
	dir := t.TempDir()
	b := branding.Get()
	if project != "" {
		if err := os.WriteFile(filepath.Join(dir, b.ConfigFile), []byte(project), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if local != "" {
		if err := os.WriteFile(filepath.Join(dir, b.LocalConfig), []byte(local), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadProjectPolicy(t *testing.T) {
	t.Parallel()

	t.Run("missing config wraps ErrNotExist", func(t *testing.T) {
		t.Parallel()
		_, err := LoadProjectPolicy(t.TempDir())
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("err = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("invalid local config is an error", func(t *testing.T) {
		t.Parallel()
		dir := writeProjectFiles(t, "version: 2\n", "securty:\n  level: baseline\n")
		if _, err := LoadProjectPolicy(dir); err == nil || !strings.Contains(err.Error(), branding.Get().LocalConfig) {
			t.Fatalf("err = %v, want a local config parse error", err)
		}
	})

	t.Run("empty org layer does not raise the project", func(t *testing.T) {
		t.Parallel()
		dir := writeProjectFiles(t, "version: 2\ntier: standard\n", "")
		p, err := LoadProjectPolicy(dir)
		if err != nil {
			t.Fatal(err)
		}
		if lvl := p.Effective.Config.Security.Level; lvl != "" {
			t.Errorf("security.level = %q, want unset (no org defaults layer)", lvl)
		}
		if w := p.Warnings(); len(w) > 0 {
			t.Errorf("unexpected warnings: %v", w)
		}
	})

	t.Run("declared level with unset bools is not a violation", func(t *testing.T) {
		t.Parallel()
		dir := writeProjectFiles(t, "version: 2\nsecurity:\n  level: strict\n", "")
		p, err := LoadProjectPolicy(dir)
		if err != nil {
			t.Fatal(err)
		}
		if w := p.Warnings(); len(w) > 0 {
			t.Errorf("unexpected warnings: %v", w)
		}
		if ag := p.Effective.Config.Security.AgeGating; ag == nil || !*ag {
			t.Errorf("age_gating = %v, want raised to true by the strict floor", ag)
		}
	})

	t.Run("local override below the floor is reported and raised", func(t *testing.T) {
		t.Parallel()
		dir := writeProjectFiles(t,
			"version: 2\nsecurity:\n  level: strict\n",
			"security:\n  level: baseline\n  script_blocking: false\nextra_packages: [neovim]\n")
		p, err := LoadProjectPolicy(dir)
		if err != nil {
			t.Fatal(err)
		}
		if lvl := p.Effective.Config.Security.Level; lvl != "strict" {
			t.Errorf("effective level = %q, want strict", lvl)
		}
		warnings := strings.Join(p.Warnings(), "\n")
		for _, want := range []string{"security.level: baseline raised to strict", "security.script_blocking: false raised to true"} {
			if !strings.Contains(warnings, want) {
				t.Errorf("warnings %q missing %q", warnings, want)
			}
		}
		if !slices.Contains(p.Effective.Config.Packages, "neovim") {
			t.Errorf("local extra_packages not resolved into packages: %v", p.Effective.Config.Packages)
		}
		if slices.Contains(p.Committed.Packages, "neovim") || p.Committed.Security.Level != "strict" {
			t.Errorf("committed resolution carries local overrides: %+v", p.Committed)
		}
	})
}

func TestProjectPolicy_Apply(t *testing.T) {
	t.Parallel()
	strictClient := &types.ClientConfig{
		Name:          "acme",
		SecurityLevel: "strict",
		BlockedMCP:    []string{types.MCPWildcard},
		AllowedMCP:    []string{"context7"},
	}
	tests := []struct {
		name    string
		project types.QsdevConfig
		local   *LocalConfig
		answers types.WizardAnswers
		check   func(t *testing.T, a types.WizardAnswers)
	}{
		{
			name:    "client strict raises level, hooks, permission, tools and MCP policy",
			project: types.QsdevConfig{Version: 2, Client: strictClient},
			answers: types.WizardAnswers{
				ClaudeCode:      true,
				ComplianceLevel: "baseline",
				MCPServers:      []string{"context7", "github", "socket"},
				EnabledTools:    map[string]bool{"semgrep": false},
			},
			check: func(t *testing.T, a types.WizardAnswers) {
				t.Helper()
				if a.ComplianceLevel != "strict" {
					t.Errorf("ComplianceLevel = %q, want strict", a.ComplianceLevel)
				}
				if !a.Hooks.PreCommit || !a.Hooks.AuditLog || !a.Hooks.AutoFormat {
					t.Errorf("strict hooks not enabled: %+v", a.Hooks)
				}
				if a.PermissionLevel != "minimal" {
					t.Errorf("PermissionLevel = %q, want the strict client's minimal", a.PermissionLevel)
				}
				if !a.EnabledTools["gitleaks"] {
					t.Errorf("compliance-required gitleaks not enabled: %v", a.EnabledTools)
				}
				if a.EnabledTools["semgrep"] {
					t.Error("recorded semgrep decision overridden")
				}
				if !slices.Equal(a.MCPServers, []string{"context7"}) {
					t.Errorf("MCPServers = %v, want [context7]", a.MCPServers)
				}
				if !slices.Equal(a.MCPPolicy.Blocked, []string{types.MCPWildcard}) || !slices.Equal(a.MCPPolicy.Allowed, []string{"context7"}) {
					t.Errorf("MCPPolicy = %+v", a.MCPPolicy)
				}
			},
		},
		{
			name:    "never lowers a stricter answer or replaces a chosen permission",
			project: types.QsdevConfig{Version: 2, Security: types.SecurityConfig{Level: "baseline"}},
			answers: types.WizardAnswers{ClaudeCode: true, ComplianceLevel: "strict", PermissionLevel: "permissive"},
			check: func(t *testing.T, a types.WizardAnswers) {
				t.Helper()
				if a.ComplianceLevel != "strict" || a.PermissionLevel != "permissive" {
					t.Errorf("answers loosened or overridden: level %q, permission %q", a.ComplianceLevel, a.PermissionLevel)
				}
			},
		},
		{
			name:    "local raise applies, local lowering does not",
			project: types.QsdevConfig{Version: 2, Security: types.SecurityConfig{Level: "enhanced"}},
			local:   &LocalConfig{Security: types.SecurityConfig{Level: "baseline"}},
			answers: types.WizardAnswers{ClaudeCode: true},
			check: func(t *testing.T, a types.WizardAnswers) {
				t.Helper()
				if a.ComplianceLevel != "enhanced" || !a.Hooks.PreCommit {
					t.Errorf("floor not applied: level %q, hooks %+v", a.ComplianceLevel, a.Hooks)
				}
			},
		},
		{
			name:    "removed client block clears a stale MCP policy",
			project: types.QsdevConfig{Version: 2},
			answers: types.WizardAnswers{MCPPolicy: types.MCPPolicy{Blocked: []string{"github"}}, MCPServers: []string{"github"}},
			check: func(t *testing.T, a types.WizardAnswers) {
				t.Helper()
				if !a.MCPPolicy.IsZero() || !slices.Equal(a.MCPServers, []string{"github"}) {
					t.Errorf("stale policy kept: %+v, servers %v", a.MCPPolicy, a.MCPServers)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p, err := ResolveProjectPolicy(&tt.project, tt.local)
			if err != nil {
				t.Fatal(err)
			}
			a := tt.answers
			p.Apply(&a)
			tt.check(t, a)
		})
	}
}

func TestPreserveCommittedPolicy(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "strict", AgeGating: boolP(true)},
		Client:   &types.ClientConfig{Name: "acme", BlockedMCP: []string{"github"}},
		Git:      types.GitConfig{BranchPattern: "^feat/"},
		Tools:    types.ToolsConfig{Config: map[string]map[string]any{"semgrep": {"rules": "p/ci"}}},
	}
	tests := []struct {
		name      string
		freshLvl  string
		wantLevel string
	}{
		{name: "keeps the stricter committed level", freshLvl: "baseline", wantLevel: "strict"},
		{name: "keeps an unset fresh level at the committed floor", freshLvl: "", wantLevel: "strict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fresh := types.QsdevConfig{Security: types.SecurityConfig{Level: tt.freshLvl}}
			PreserveCommittedPolicy(&fresh, committed)
			if fresh.Security.Level != tt.wantLevel {
				t.Errorf("level = %q, want %q", fresh.Security.Level, tt.wantLevel)
			}
			if fresh.Security.AgeGating == nil || !*fresh.Security.AgeGating {
				t.Error("age_gating floor dropped")
			}
			if fresh.Client == nil || fresh.Client == committed.Client || fresh.Client.Name != "acme" {
				t.Errorf("client not carried as a copy: %+v", fresh.Client)
			}
			if fresh.Git.BranchPattern != "^feat/" || fresh.Tools.Config["semgrep"]["rules"] != "p/ci" {
				t.Errorf("git/tools.config not carried: %+v %+v", fresh.Git, fresh.Tools.Config)
			}
		})
	}
	fresh := types.QsdevConfig{Security: types.SecurityConfig{Level: "enhanced"}}
	PreserveCommittedPolicy(&fresh, nil)
	if fresh.Security.Level != "enhanced" || fresh.Client != nil {
		t.Errorf("nil committed changed fresh: %+v", fresh)
	}
}
