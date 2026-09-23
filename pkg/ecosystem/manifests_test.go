package ecosystem

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRecordManifests(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, f := range []string{"build.gradle.kts", "gradle/libs.versions.toml", "project/plugins.sbt", "project/deps.sbt", "project/build.properties"} {
		p := filepath.Join(dir, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		patterns []string
		want     string
	}{
		{"none exist", []string{"pom.xml", "build.gradle"}, ""},
		{"only existing listed", []string{"build.gradle", "build.gradle.kts", "gradle/libs.versions.toml"}, "build.gradle.kts,gradle/libs.versions.toml"},
		{"glob and duplicates", []string{"project/*.sbt", "project/plugins.sbt"}, "project/deps.sbt,project/plugins.sbt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RecordManifests(dir, tt.patterns...); got != tt.want {
				t.Errorf("RecordManifests(%v) = %q, want %q", tt.patterns, got, tt.want)
			}
		})
	}
}

func TestDetectedManifests(t *testing.T) {
	t.Parallel()

	candidates := []ManifestFileInfo{
		{Path: "build.gradle", Ecosystem: "gradle", LockFile: "gradle.lockfile"},
		{Path: "build.gradle.kts", Ecosystem: "gradle", LockFile: "gradle.lockfile"},
		{Path: "project/*.sbt", Ecosystem: "sbt"},
	}
	fallback := []ManifestFileInfo{{Path: "build.gradle", Ecosystem: "gradle"}}

	tests := []struct {
		name   string
		extras map[string]string
		want   []ManifestFileInfo
	}{
		{"no record falls back", nil, fallback},
		{"record without candidates falls back", map[string]string{ExtraManifests: "pom.xml"}, fallback},
		{
			"recorded files replace the default name",
			map[string]string{ExtraManifests: "build.gradle.kts,pom.xml,project/plugins.sbt"},
			[]ManifestFileInfo{
				{Path: "build.gradle.kts", Ecosystem: "gradle", LockFile: "gradle.lockfile"},
				{Path: "project/plugins.sbt", Ecosystem: "sbt"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetectedManifests(ModuleConfig{Extras: tt.extras}, candidates, fallback)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DetectedManifests = %+v, want %+v", got, tt.want)
			}
		})
	}
}
