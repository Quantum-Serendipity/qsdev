package ecosystem

import "testing"

func TestCleanProjectDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"empty is the root", "", ""},
		{"dot is the root", ".", ""},
		{"plain directory", "frontend", "frontend"},
		{"nested directory", "apps/web", "apps/web"},
		{"trailing slash cleaned", "web/", "web"},
		{"dot segments cleaned", "./apps/../web", "web"},
		{"surrounding space trimmed", " web ", "web"},
		{"absolute path rejected", "/etc", ""},
		{"parent rejected", "..", ""},
		{"escaping path rejected", "../other", ""},
		{"escaping after clean rejected", "web/../../other", ""},
		{"shell metacharacter rejected", "web;rm -rf ~", ""},
		{"nix antiquote rejected", "${builtins.abort}", ""},
		{"quote rejected", `web"x`, ""},
		{"leading dash rejected", "-rf", ""},
		{"backslash rejected", `web\ui`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CleanProjectDir(tt.dir); got != tt.want {
				t.Errorf("CleanProjectDir(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestModuleConfigDirectoryHelpers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		extras  map[string]string
		wantDir string
		wantIn  string
		wantCmd string
	}{
		{"no directory", nil, "", ".npmrc", "npm ci"},
		{"subdirectory", map[string]string{ExtraDirectory: "frontend"}, "frontend", "frontend/.npmrc", "(cd frontend && npm ci)"},
		{"unsafe directory falls back to root", map[string]string{ExtraDirectory: "../x"}, "", ".npmrc", "npm ci"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ModuleConfig{Extras: tt.extras}
			if got := cfg.Directory(); got != tt.wantDir {
				t.Errorf("Directory() = %q, want %q", got, tt.wantDir)
			}
			if got := cfg.InDirectory(".npmrc"); got != tt.wantIn {
				t.Errorf("InDirectory() = %q, want %q", got, tt.wantIn)
			}
			if got := cfg.InDirectoryCommand("npm ci"); got != tt.wantCmd {
				t.Errorf("InDirectoryCommand() = %q, want %q", got, tt.wantCmd)
			}
		})
	}
}
