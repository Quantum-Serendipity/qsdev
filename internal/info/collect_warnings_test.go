package info

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectInfo_SurfacesUnreadableProjectFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       string
		state        string
		wantProfile  string
		wantWarnings []string
	}{
		{
			name:         "corrupt config reports unknown profile",
			config:       "version: [not-an-int\n",
			wantProfile:  SecurityProfileUnknown,
			wantWarnings: []string{"config could not be parsed"},
		},
		{
			name:         "corrupt state is reported",
			config:       "version: 1\nsecurity:\n  level: enhanced\n",
			state:        "files: [this is: not a map\n",
			wantProfile:  "enhanced",
			wantWarnings: []string{"state could not be loaded"},
		},
		{
			name:        "healthy project has no warnings",
			config:      "version: 1\nsecurity:\n  level: enhanced\n",
			wantProfile: "enhanced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(tt.config), 0o644); err != nil {
				t.Fatalf("writing config: %v", err)
			}
			if tt.state != "" {
				stateDir := filepath.Join(dir, ".devinit")
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatalf("creating state dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(stateDir, ".qsdev-init-state.yaml"), []byte(tt.state), 0o644); err != nil {
					t.Fatalf("writing state: %v", err)
				}
			}

			got, err := CollectInfo(dir)
			if err != nil {
				t.Fatalf("CollectInfo: %v", err)
			}
			if got.SecurityProfile != tt.wantProfile {
				t.Errorf("SecurityProfile = %q, want %q", got.SecurityProfile, tt.wantProfile)
			}
			if len(got.Warnings) != len(tt.wantWarnings) {
				t.Fatalf("Warnings = %q, want %d warning(s)", got.Warnings, len(tt.wantWarnings))
			}
			for i, want := range tt.wantWarnings {
				if !strings.Contains(got.Warnings[i], want) {
					t.Errorf("Warnings[%d] = %q, want it to contain %q", i, got.Warnings[i], want)
				}
			}
		})
	}
}

func TestFormatDefault_StableCategoryOrderAndWarnings(t *testing.T) {
	t.Parallel()

	pi := &ProjectInfo{
		ProjectName:     "demo",
		SecurityProfile: SecurityProfileUnknown,
		ToolsByCategory: map[string]int{"Security": 1, "AI": 2, "Linting": 3, "Docs": 4},
		Warnings:        []string{"config could not be parsed: boom"},
	}

	var first string
	for i := range 20 {
		var buf bytes.Buffer
		if err := FormatDefault(pi, &buf); err != nil {
			t.Fatalf("FormatDefault: %v", err)
		}
		out := buf.String()
		if i == 0 {
			first = out
			continue
		}
		if out != first {
			t.Fatalf("FormatDefault output is not deterministic:\n%s\nvs\n%s", first, out)
		}
	}

	aiIdx := strings.Index(first, "  AI: 2")
	secIdx := strings.Index(first, "  Security: 1")
	if aiIdx < 0 || secIdx < 0 || aiIdx > secIdx {
		t.Errorf("categories not sorted in output:\n%s", first)
	}
	if !strings.Contains(first, "Warning:       config could not be parsed: boom") {
		t.Errorf("warning not rendered:\n%s", first)
	}
}
