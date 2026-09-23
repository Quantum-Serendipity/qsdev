package devinit

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
			config:  "version: 1\n",
			wantOut: "already at schema version 1 (current)",
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
