package bwrap

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

func TestFilterEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      map[string]string
		category sandbox.HookCategory
		wantKeys []string // keys that must be present
		denyKeys []string // keys that must be absent
	}{
		{
			name: "strips credential patterns",
			env: map[string]string{
				"AWS_SECRET_ACCESS_KEY": "secret",
				"GITHUB_TOKEN":          "ghp_xxx",
				"NPM_TOKEN":             "npm_xxx",
				"MY_API_KEY":            "key123",
				"DATABASE_PASSWORD":     "pw",
				"PATH":                  "/usr/bin",
				"HOME":                  "/home/user",
				"LANG":                  "en_US.UTF-8",
			},
			category: sandbox.CategoryLinter,
			wantKeys: []string{"PATH", "HOME", "LANG"},
			denyKeys: []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN", "NPM_TOKEN", "MY_API_KEY", "DATABASE_PASSWORD"},
		},
		{
			name: "preserves git variables",
			env: map[string]string{
				"GIT_DIR":         "/repo/.git",
				"GIT_WORK_TREE":   "/repo",
				"GIT_AUTHOR_NAME": "Test",
			},
			category: sandbox.CategoryFormatter,
			wantKeys: []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_AUTHOR_NAME"},
			denyKeys: nil,
		},
		{
			name: "preserves LC variables",
			env: map[string]string{
				"LC_ALL":   "C",
				"LC_CTYPE": "UTF-8",
			},
			category: sandbox.CategoryLinter,
			wantKeys: []string{"LC_ALL", "LC_CTYPE"},
			denyKeys: nil,
		},
		{
			name: "strips unknown variables",
			env: map[string]string{
				"RANDOM_VAR": "x",
				"FOO_BAR":    "y",
				"PATH":       "/usr/bin",
			},
			category: sandbox.CategoryLinter,
			wantKeys: []string{"PATH"},
			denyKeys: []string{"RANDOM_VAR", "FOO_BAR"},
		},
		{
			name:     "empty env returns empty map",
			env:      map[string]string{},
			category: sandbox.CategoryLinter,
			wantKeys: nil,
			denyKeys: nil,
		},
		{
			name: "deny overrides allowlist",
			env: map[string]string{
				"PATH_TOKEN": "should-be-stripped",
				"PATH":       "/usr/bin",
			},
			category: sandbox.CategoryLinter,
			wantKeys: []string{"PATH"},
			denyKeys: []string{"PATH_TOKEN"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FilterEnvironment(tt.env, tt.category)

			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("expected key %q to be present, but it was stripped", key)
				}
			}

			for _, key := range tt.denyKeys {
				if _, ok := got[key]; ok {
					t.Errorf("expected key %q to be stripped, but it was present", key)
				}
			}

			if tt.env != nil && len(tt.env) == 0 && len(got) != 0 {
				t.Errorf("expected empty map, got %d entries", len(got))
			}
		})
	}
}
