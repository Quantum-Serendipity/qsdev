package types

import (
	"slices"
	"testing"
)

func TestLanguageChoiceSetting(t *testing.T) {
	t.Parallel()
	lc := LanguageChoice{
		Name:           "java",
		Version:        "21",
		PackageManager: "gradle",
		Extras:         []string{"build_tool=maven", "kotlin", "empty=", "dup=a", "dup=b"},
	}
	tests := []struct {
		key    string
		want   string
		wantOK bool
	}{
		{SettingVersion, "21", true},
		{SettingPackageManager, "gradle", true},
		{"build_tool", "maven", true},
		{"kotlin", "true", true},
		{"empty", "", false},
		{"dup", "b", true},
		{"missing", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			t.Parallel()
			got, ok := lc.Setting(tt.key)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Setting(%q) = (%q, %v), want (%q, %v)", tt.key, got, ok, tt.want, tt.wantOK)
			}
		})
	}

	if _, ok := (LanguageChoice{}).Setting(SettingVersion); ok {
		t.Error("Setting(version) on an empty choice reports set")
	}
}

func TestLanguageChoiceWithSetting(t *testing.T) {
	t.Parallel()
	base := LanguageChoice{Name: "x", Extras: []string{"a=1", "flag", "b=2", "flag=false"}}
	tests := []struct {
		name       string
		key, value string
		want       LanguageChoice
	}{
		{"version field", SettingVersion, "1.2", LanguageChoice{Name: "x", Version: "1.2", Extras: base.Extras}},
		{"package manager field", SettingPackageManager, "uv", LanguageChoice{Name: "x", PackageManager: "uv", Extras: base.Extras}},
		{"replaces extra in place", "a", "9", LanguageChoice{Name: "x", Extras: []string{"a=9", "flag", "b=2", "flag=false"}}},
		{"replaces bare flag and duplicates", "flag", "true", LanguageChoice{Name: "x", Extras: []string{"a=1", "flag=true", "b=2"}}},
		{"appends new extra", "c", "3", LanguageChoice{Name: "x", Extras: []string{"a=1", "flag", "b=2", "flag=false", "c=3"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			orig := slices.Clone(base.Extras)
			got := base.WithSetting(tt.key, tt.value)
			if got.Name != tt.want.Name || got.Version != tt.want.Version ||
				got.PackageManager != tt.want.PackageManager || !slices.Equal(got.Extras, tt.want.Extras) {
				t.Errorf("WithSetting(%q, %q) = %+v, want %+v", tt.key, tt.value, got, tt.want)
			}
			if !slices.Equal(base.Extras, orig) {
				t.Errorf("WithSetting modified the receiver's extras: %v", base.Extras)
			}
			if v, _ := got.Setting(tt.key); v != tt.value {
				t.Errorf("Setting(%q) after WithSetting = %q, want %q", tt.key, v, tt.value)
			}
		})
	}
}
