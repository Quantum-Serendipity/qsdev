package detect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestDetect_UpdateBotConfigs pins which files detection records as a
// dependency-update bot's configuration, so the generators can keep a
// project to one bot.
func TestDetect_UpdateBotConfigs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{"none", nil, nil},
		{"dependabot.yml", []string{".github/dependabot.yml"}, []string{".github/dependabot.yml"}},
		{"dependabot.yaml", []string{".github/dependabot.yaml"}, []string{".github/dependabot.yaml"}},
		{"dependabot outside .github", []string{"dependabot.yml"}, nil},
		{"renovate.json", []string{"renovate.json"}, []string{"renovate.json"}},
		{"renovate.json5", []string{"renovate.json5"}, []string{"renovate.json5"}},
		{".renovaterc", []string{".renovaterc"}, []string{".renovaterc"}},
		{".renovaterc.json", []string{".renovaterc.json"}, []string{".renovaterc.json"}},
		{".renovaterc.json5", []string{".renovaterc.json5"}, []string{".renovaterc.json5"}},
		{".github/renovate.json", []string{".github/renovate.json"}, []string{".github/renovate.json"}},
		{".github/renovate.json5", []string{".github/renovate.json5"}, []string{".github/renovate.json5"}},
		{".gitlab/renovate.json", []string{".gitlab/renovate.json"}, []string{".gitlab/renovate.json"}},
		{".gitlab/renovate.json5", []string{".gitlab/renovate.json5"}, []string{".gitlab/renovate.json5"}},
		{"directory is not a config", []string{"renovate.json/x"}, nil},
		{"both", []string{"renovate.json", ".github/dependabot.yml"}, []string{".github/dependabot.yml", "renovate.json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for _, f := range tt.files {
				path := filepath.Join(dir, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			got := Detect(t.Context(), dir, hostFakes()...).UpdateBotConfigs
			if !slices.Equal(got, tt.want) {
				t.Errorf("UpdateBotConfigs = %v, want %v", got, tt.want)
			}
		})
	}
}
