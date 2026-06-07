package cloudcommon

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
)

var (
	providerBlockRe    = regexp.MustCompile(`^\s*provider\s+"(aws|google|azurerm)"\s*\{?`)
	requiredProviderRe = regexp.MustCompile(`source\s*=\s*"hashicorp/(aws|google|azurerm)"`)
)

// DetectTerraformProviders scans .tf files in projectRoot for cloud provider
// declarations. Returns a map of detected provider names.
func DetectTerraformProviders(projectRoot string) map[string]bool {
	result := make(map[string]bool)

	matches, err := filepath.Glob(filepath.Join(projectRoot, "*.tf"))
	if err != nil || len(matches) == 0 {
		// Also check one level deep for monorepos.
		deepMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*", "*.tf"))
		matches = append(matches, deepMatches...)
	}

	for _, path := range matches {
		scanTFFile(path, result)
	}
	return result
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
