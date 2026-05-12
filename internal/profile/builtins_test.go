package profile

import (
	"testing"
)

func TestBuiltinProfiles_NonEmpty(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			if p.Name == "" {
				t.Error("Name is empty")
			}
			if p.Description == "" {
				t.Error("Description is empty")
			}
		})
	}
}

func TestBuiltinProfiles_EnvironmentVarsNonEmpty(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			env := p.EnvironmentVars()
			if len(env) == 0 {
				t.Errorf("profile %q produces empty EnvironmentVars", p.Name)
			}
		})
	}
}

func TestConsultingDefault_Structure(t *testing.T) {
	p := ConsultingDefault

	if p.Registry.Type != RegistryNexus {
		t.Errorf("Registry.Type = %q, want nexus", p.Registry.Type)
	}
	if p.NixCache.Type != NixCacheCachix {
		t.Errorf("NixCache.Type = %q, want cachix", p.NixCache.Type)
	}
	if p.BuildCache.Type != BuildCacheSccache {
		t.Errorf("BuildCache.Type = %q, want sccache", p.BuildCache.Type)
	}
	if p.Scanning.Vulnerability != VulnScannerOSV {
		t.Errorf("Scanning.Vulnerability = %q, want osv", p.Scanning.Vulnerability)
	}
	if p.Scanning.Behavioral != BehavioralSocket {
		t.Errorf("Scanning.Behavioral = %q, want socket", p.Scanning.Behavioral)
	}
	if p.Updates.Type != UpdateToolRenovate {
		t.Errorf("Updates.Type = %q, want renovate", p.Updates.Type)
	}
	if p.Updates.AgeGatingDays != 3 {
		t.Errorf("Updates.AgeGatingDays = %d, want 3", p.Updates.AgeGatingDays)
	}
	if !p.Updates.AutomergePatches {
		t.Error("Updates.AutomergePatches should be true")
	}
}

func TestStartupGitHub_Structure(t *testing.T) {
	p := StartupGitHub

	if p.Registry.Type != RegistryGitHub {
		t.Errorf("Registry.Type = %q, want github", p.Registry.Type)
	}
	if p.BuildCache.Type != BuildCacheTurborepo {
		t.Errorf("BuildCache.Type = %q, want turborepo", p.BuildCache.Type)
	}
	if p.Updates.Type != UpdateToolDependabot {
		t.Errorf("Updates.Type = %q, want dependabot", p.Updates.Type)
	}
	if p.Updates.AgeGatingDays != 0 {
		t.Errorf("Updates.AgeGatingDays = %d, want 0", p.Updates.AgeGatingDays)
	}
}

func TestEnterprise_Structure(t *testing.T) {
	p := Enterprise

	if p.Registry.Type != RegistryArtifactory {
		t.Errorf("Registry.Type = %q, want artifactory", p.Registry.Type)
	}
	if p.Scanning.Vulnerability != VulnScannerSnyk {
		t.Errorf("Scanning.Vulnerability = %q, want snyk", p.Scanning.Vulnerability)
	}
	if p.Updates.AgeGatingDays != 7 {
		t.Errorf("Updates.AgeGatingDays = %d, want 7", p.Updates.AgeGatingDays)
	}
	if p.SBOM.Signing != SBOMSigningCosign {
		t.Errorf("SBOM.Signing = %q, want cosign", p.SBOM.Signing)
	}

	// Enterprise should include nuget
	hasNuget := false
	for _, eco := range p.Registry.Ecosystems {
		if eco == "nuget" {
			hasNuget = true
		}
	}
	if !hasNuget {
		t.Error("Enterprise registry should include nuget ecosystem")
	}
}
