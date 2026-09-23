package cloudcommon

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	providerBlockRe = regexp.MustCompile(`^\s*provider\s+"(aws|google|azurerm)"\s*\{?`)
	// requiredProviderRe accepts both the short source address
	// ("hashicorp/aws") and the fully-qualified registry forms.
	requiredProviderRe = regexp.MustCompile(`source\s*=\s*"(?:registry\.terraform\.io/|registry\.opentofu\.org/)?hashicorp/(aws|google|azurerm)"`)
)

// maxTFScanDepth bounds how many directory levels below projectRoot are
// scanned for .tf files (root is depth 0), covering layouts such as
// modules/aws/*.tf and infra/modules/aws/*.tf without walking a whole
// monorepo.
const maxTFScanDepth = 3

// skippedTFDirs are directories that never hold the project's own Terraform
// configuration: provider/module caches and vendored third-party trees.
var skippedTFDirs = map[string]bool{
	".terraform":   true,
	"node_modules": true,
	"vendor":       true,
}

// DetectTerraformProviders scans .tf files in projectRoot and its
// subdirectories (up to maxTFScanDepth levels) for cloud provider
// declarations. Returns a map of detected provider names.
//
// Subdirectories are always scanned: a root main.tf holding only the backend
// commonly delegates all provider use to ./modules/*, and provider-specific
// deny rules depend on seeing those.
func DetectTerraformProviders(projectRoot string) map[string]bool {
	result := make(map[string]bool)

	_ = filepath.WalkDir(projectRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Unreadable entries are skipped; detection is best-effort.
			return nil //nolint:nilerr // best-effort scan
		}
		if d.IsDir() {
			if path == projectRoot {
				return nil
			}
			name := d.Name()
			if skippedTFDirs[name] || strings.HasPrefix(name, ".") || tfScanDepth(projectRoot, path) > maxTFScanDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".tf") {
			scanTFFile(path, result)
		}
		return nil
	})
	return result
}

// tfScanDepth returns how many directory levels dir is below root.
func tfScanDepth(root, dir string) int {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return maxTFScanDepth + 1
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

func scanTFFile(path string, result map[string]bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if m := providerBlockRe.FindStringSubmatch(line); len(m) > 1 {
			result[m[1]] = true
		}

		if m := requiredProviderRe.FindStringSubmatch(line); len(m) > 1 {
			result[m[1]] = true
		}
	}
}
