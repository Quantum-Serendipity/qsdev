package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestSecurityConfigs_NoOverwriteOfDetectionMarkers guards against a module
// replacing a file its own Detect treats as a project marker. Such a file
// necessarily pre-exists in every project that triggers the marker (e.g. a
// Bazel repo's .bazelrc), so an Overwrite strategy would destroy user content
// on first init. Marker-like files must use Skip or a qsdev-owned sibling.
func TestSecurityConfigs_NoOverwriteOfDetectionMarkers(t *testing.T) {
	t.Parallel()

	for _, mod := range ecosystem.DefaultRegistry().All() {
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			for _, cfg := range configVariants(mod) {
				for _, f := range mod.SecurityConfigs(cfg) {
					if f.Strategy != types.Overwrite {
						continue
					}
					dir := t.TempDir()
					full := filepath.Join(dir, filepath.FromSlash(f.Path))
					if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(full, f.Content, 0o644); err != nil {
						t.Fatal(err)
					}
					if mod.Detect(dir).Detected {
						t.Errorf("%s (package manager %q) overwrites %s, which is one of its own detection markers",
							mod.Name(), cfg.PackageManager, f.Path)
					}
				}
			}
		})
	}
}

// configVariants returns the default config plus one config per declared
// package manager, so PM-specific security files are exercised too. Modules
// that select the tool through Extras (java build_tool, cpp package_manager)
// or a flavour flag (Yarn Classic) get the same name there as well.
func configVariants(mod ecosystem.EcosystemModule) []ecosystem.ModuleConfig {
	variants := []ecosystem.ModuleConfig{{}}
	for _, pm := range mod.PackageManagers() {
		variants = append(variants,
			ecosystem.ModuleConfig{PackageManager: pm.Name},
			ecosystem.ModuleConfig{PackageManager: pm.Name, Extras: map[string]string{
				"build_tool": pm.Name, "package_manager": pm.Name, "yarn_classic": "true",
			}},
		)
	}
	return variants
}

// TestSecurityConfigs_UserOwnedConfigsNotOverwritten guards the conventional
// package-manager and tool configs that projects maintain themselves (a pnpm
// monorepo's workspace list, scoped registries in .npmrc, Gradle JVM
// settings, Cargo registries). Replacing them on first init silently breaks
// the project, so they must never use the Overwrite strategy.
func TestSecurityConfigs_UserOwnedConfigsNotOverwritten(t *testing.T) {
	t.Parallel()
	userOwned := map[string]bool{
		".npmrc": true, "pnpm-workspace.yaml": true, ".yarnrc.yml": true, ".yarnrc": true,
		"bunfig.toml": true, "pip.conf": true, ".cargo/config.toml": true, "nuget.config": true,
		"gradle.properties": true, "init.gradle": true, ".mvn/settings.xml": true,
		".terraformrc": true, ".bazelrc": true, ".bundle/config": true, ".gemrc": true,
		".hadolint.yaml": true, "vcpkg-configuration.json": true, "Directory.Build.props": true,
	}
	for _, mod := range ecosystem.DefaultRegistry().All() {
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			for _, cfg := range configVariants(mod) {
				for _, f := range mod.SecurityConfigs(cfg) {
					if userOwned[f.Path] && f.Strategy == types.Overwrite {
						t.Errorf("%s (package manager %q) overwrites user-owned %s", mod.Name(), cfg.PackageManager, f.Path)
					}
				}
			}
		})
	}
}
