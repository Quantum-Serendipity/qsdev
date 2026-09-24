package cloudcommon

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

var (
	providerBlockRe = regexp.MustCompile(`^\s*provider\s+"(aws|google(?:-beta)?|azurerm)"\s*\{?`)
	// requiredProviderRe accepts both the short source address
	// ("hashicorp/aws") and the fully-qualified registry forms.
	requiredProviderRe = regexp.MustCompile(`source\s*=\s*"(?:registry\.terraform\.io/|registry\.opentofu\.org/)?hashicorp/(aws|google(?:-beta)?|azurerm)"`)
)

// DetectTerraformProviders scans the Terraform/OpenTofu configuration files
// (.tf and .tofu) in projectRoot and its subdirectories, up to
// ecosystem.ProjectScanDepth levels, for cloud provider declarations. Returns
// a map of detected provider names.
//
// Subdirectories are always scanned: a root main.tf holding only the backend
// commonly delegates all provider use to ./modules/*, and provider-specific
// deny rules depend on seeing those. The walk is the one the terraform module
// uses for its own detection, so a provider found here implies Terraform is
// detected too.
func DetectTerraformProviders(projectRoot string) map[string]bool {
	result := make(map[string]bool)
	ecosystem.WalkProjectFiles(projectRoot, ecosystem.ProjectScanDepth, func(path string) bool {
		if IsTerraformHCLFile(filepath.Base(path)) {
			scanTFFile(path, result)
		}
		return true
	})
	return result
}

// IsTerraformHCLFile reports whether name is a Terraform or OpenTofu
// configuration file in HCL syntax (.tf, .tofu).
func IsTerraformHCLFile(name string) bool {
	return strings.HasSuffix(name, ".tf") || strings.HasSuffix(name, ".tofu")
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
