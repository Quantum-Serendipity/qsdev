package devinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestRunMigrate_Versions(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantOut string
		wantErr string
	}{
		{
			name:    "current version needs no migration",
			config:  fmt.Sprintf("version: %d\n", types.ConfigVersionCurrent),
			wantOut: fmt.Sprintf("already at schema version %d (current)", types.ConfigVersionCurrent),
		},
		{
			name:    "newer version is rejected",
			config:  "version: 99\n",
			wantErr: "schema version 99",
		},
		{
			name:    "version zero is rejected",
			config:  "version: 0\n",
			wantErr: "schema version 0",
		},
		{
			name:    "negative version is rejected",
			config:  "version: -3\n",
			wantErr: "schema version -3",
		},
		{
			name:    "missing version is rejected",
			config:  "project: x\n",
			wantErr: `missing "version"`,
		},
		{
			name:    "non-integer version is rejected",
			config:  "version: 1.5\n",
			wantErr: "must be an integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
			if err := os.WriteFile(cfgPath, []byte(tt.config), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			cmd, buf := newTestCmd()
			err := runMigrate(cmd, true)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want error containing %q (output %q)", err, tt.wantErr, buf.String())
				}
				if strings.Contains(buf.String(), "current") {
					t.Errorf("unsupported version reported as current: %q", buf.String())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(buf.String(), tt.wantOut) {
					t.Errorf("output %q does not contain %q", buf.String(), tt.wantOut)
				}
			}

			// No outcome may rewrite the file.
			data, readErr := os.ReadFile(cfgPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !bytes.Equal(data, []byte(tt.config)) {
				t.Errorf("config was rewritten:\n%s", data)
			}
		})
	}
}

// `config migrate --write` moves a version 1 infra profile out of `profile`
// into `infra_profile` and records the current schema version.
func TestRunMigrate_SplitsV1Profile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
	legacy := "version: 1\nprofile: enterprise\nlanguages:\n  - name: python\n    version: 3.10\n"
	if err := os.WriteFile(cfgPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	cmd, buf := newTestCmd()
	if err := runMigrate(cmd, true); err != nil {
		t.Fatalf("runMigrate: %v (output %q)", err, buf.String())
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := qsdevconfig.ParseQsdevConfigBytes(data)
	if err != nil {
		t.Fatalf("migrated config does not parse: %v\n%s", err, data)
	}
	for _, want := range []string{fmt.Sprintf("version: %d\n", types.ConfigVersionCurrent), "infra_profile: enterprise\n"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("migrated config lacks %q:\n%s", want, data)
		}
	}
	if cfg.Profile != "" || cfg.InfraProfile != "enterprise" {
		t.Errorf("Profile=%q InfraProfile=%q, want \"\" enterprise", cfg.Profile, cfg.InfraProfile)
	}
	if got := cfg.Languages[0].Version; got != "3.10" {
		t.Errorf("language version = %q, want 3.10 as written", got)
	}
}
