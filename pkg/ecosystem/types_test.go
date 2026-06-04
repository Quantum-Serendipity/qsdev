package ecosystem

import "testing"

func TestModuleConfig_PM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    ModuleConfig
		defaultPM string
		want      string
	}{
		{
			name:      "returns configured package manager",
			config:    ModuleConfig{PackageManager: "pnpm"},
			defaultPM: "npm",
			want:      "pnpm",
		},
		{
			name:      "falls back to default when empty",
			config:    ModuleConfig{},
			defaultPM: "pip",
			want:      "pip",
		},
		{
			name:      "falls back to default when explicitly empty string",
			config:    ModuleConfig{PackageManager: ""},
			defaultPM: "npm",
			want:      "npm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.PM(tt.defaultPM)
			if got != tt.want {
				t.Errorf("PM(%q) = %q, want %q", tt.defaultPM, got, tt.want)
			}
		})
	}
}

func TestModuleConfig_Extra(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     ModuleConfig
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "returns value when present",
			config:     ModuleConfig{Extras: map[string]string{"build_tool": "mill"}},
			key:        "build_tool",
			defaultVal: "sbt",
			want:       "mill",
		},
		{
			name:       "falls back when key absent",
			config:     ModuleConfig{Extras: map[string]string{"other": "val"}},
			key:        "build_tool",
			defaultVal: "cabal",
			want:       "cabal",
		},
		{
			name:       "falls back when value empty",
			config:     ModuleConfig{Extras: map[string]string{"build_tool": ""}},
			key:        "build_tool",
			defaultVal: "tools-deps",
			want:       "tools-deps",
		},
		{
			name:       "falls back when Extras nil",
			config:     ModuleConfig{},
			key:        "variant",
			defaultVal: "terraform",
			want:       "terraform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.Extra(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("Extra(%q, %q) = %q, want %q", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}
