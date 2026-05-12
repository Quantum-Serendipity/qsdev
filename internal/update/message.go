package update

import "fmt"

// NixMergeInstructions returns user-facing instructions for manually merging devenv.nix.
func NixMergeInstructions(sidecarPath string) string {
	return fmt.Sprintf(`devenv.nix has been modified since it was generated.
A new version has been written to: %s

To review the differences:
  diff -u devenv.nix %s

To accept the new version:
  mv %s devenv.nix

To keep your current version:
  rm %s

To merge manually, compare the two files and incorporate the changes you need.`, sidecarPath, sidecarPath, sidecarPath, sidecarPath)
}

// NixForceOverwriteWarning returns a warning when --force overwrites modified devenv.nix.
func NixForceOverwriteWarning() string {
	return "WARNING: Overwriting modified devenv.nix (--force). Your customizations have been replaced."
}
