package sectools_test

import (
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// clusterImagePolicy is the subset of a policy-controller ClusterImagePolicy
// the tests inspect.
type clusterImagePolicy struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Images []struct {
			Glob string `yaml:"glob"`
		} `yaml:"images"`
		Authorities []struct {
			Keyless struct {
				URL        string `yaml:"url"`
				Identities []struct {
					Issuer        string `yaml:"issuer"`
					IssuerRegExp  string `yaml:"issuerRegExp"`
					SubjectRegExp string `yaml:"subjectRegExp"`
				} `yaml:"identities"`
			} `yaml:"keyless"`
			Ctlog struct {
				URL string `yaml:"url"`
			} `yaml:"ctlog"`
		} `yaml:"authorities"`
	} `yaml:"spec"`
}

func cosignPolicyFor(t *testing.T, remote string) *types.GeneratedFile {
	t.Helper()
	a := types.WizardAnswers{}
	a.Detected.RemoteURL = remote
	f, err := sectools.GenerateCosignPolicy(a)
	if err != nil {
		t.Fatalf("GenerateCosignPolicy(%q) error: %v", remote, err)
	}
	return f
}

func TestGenerateCosignPolicy_Structure(t *testing.T) {
	t.Parallel()
	for _, remote := range []string{"", "https://github.com/acme/widget.git", "git@gitlab.com:acme/widget.git"} {
		t.Run(remote, func(t *testing.T) {
			t.Parallel()
			f := cosignPolicyFor(t, remote)
			if f.Path != ".cosign/policy.yaml" {
				t.Errorf("Path = %q, want %q", f.Path, ".cosign/policy.yaml")
			}
			if f.Mode != 0o644 {
				t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
			}
			// Users extend or fill in the policy; regeneration must not clobber it.
			if f.Strategy != types.ManualMerge {
				t.Errorf("Strategy = %v, want ManualMerge", f.Strategy)
			}
			if f.Owner != "container-security" {
				t.Errorf("Owner = %q, want %q", f.Owner, "container-security")
			}
		})
	}
}

// TestGenerateCosignPolicy_GitHubRemote checks that a github.com origin yields
// an enforcing policy scoped to the project's own GHCR path and pinned to the
// project's own workflows — never qsdev's repository or a "**" image glob.
func TestGenerateCosignPolicy_GitHubRemote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		remote   string
		wantName string
		wantGlob string
	}{
		{"https", "https://github.com/acme/widget.git", "acme-widget-keyless", "ghcr.io/acme/widget"},
		{"scp", "git@github.com:acme/widget.git", "acme-widget-keyless", "ghcr.io/acme/widget"},
		{"ssh", "ssh://git@github.com/acme/widget", "acme-widget-keyless", "ghcr.io/acme/widget"},
		{"mixed case", "https://github.com/AcmeCorp/Widget_API.git", "acmecorp-widget-api-keyless", "ghcr.io/acmecorp/widget_api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := cosignPolicyFor(t, tt.remote)

			var p clusterImagePolicy
			if err := yaml.Unmarshal(f.Content, &p); err != nil {
				t.Fatalf("generated policy is not valid YAML: %v\n%s", err, f.Content)
			}
			if p.APIVersion != "policy.sigstore.dev/v1beta1" || p.Kind != "ClusterImagePolicy" {
				t.Errorf("apiVersion/kind = %q/%q", p.APIVersion, p.Kind)
			}
			if p.Metadata.Name != tt.wantName {
				t.Errorf("metadata.name = %q, want %q", p.Metadata.Name, tt.wantName)
			}

			var globs []string
			for _, img := range p.Spec.Images {
				globs = append(globs, img.Glob)
			}
			if want := []string{tt.wantGlob, tt.wantGlob + "/**"}; strings.Join(globs, ",") != strings.Join(want, ",") {
				t.Errorf("images = %v, want %v", globs, want)
			}

			if len(p.Spec.Authorities) != 1 || len(p.Spec.Authorities[0].Keyless.Identities) != 1 {
				t.Fatalf("want exactly one keyless identity, got %+v", p.Spec.Authorities)
			}
			auth := p.Spec.Authorities[0]
			if auth.Keyless.URL != "https://fulcio.sigstore.dev" || auth.Ctlog.URL != "https://rekor.sigstore.dev" {
				t.Errorf("fulcio/rekor = %q/%q", auth.Keyless.URL, auth.Ctlog.URL)
			}
			id := auth.Keyless.Identities[0]
			if id.Issuer != branding.GitHubActionsOIDCIssuer || id.IssuerRegExp != "" {
				t.Errorf("issuer = %q (regexp %q), want exact %q", id.Issuer, id.IssuerRegExp, branding.GitHubActionsOIDCIssuer)
			}
			assertSubjectPinned(t, id.SubjectRegExp, tt.remote)
		})
	}
}

// assertSubjectPinned compiles subjectRegExp and checks it accepts the
// project's own workflow SANs and rejects every other repository's.
func assertSubjectPinned(t *testing.T, subjectRegExp, remote string) {
	t.Helper()
	re, err := regexp.Compile(subjectRegExp)
	if err != nil {
		t.Fatalf("subjectRegExp %q does not compile: %v", subjectRegExp, err)
	}
	repo := "acme/widget"
	if strings.Contains(remote, "AcmeCorp") {
		repo = "AcmeCorp/Widget_API"
	}
	accept := []string{
		"https://github.com/" + repo + "/.github/workflows/release.yml@refs/tags/v1.0.0",
		"https://github.com/" + strings.ToLower(repo) + "/.github/workflows/build.yml@refs/heads/main",
	}
	reject := []string{
		branding.RepoURL() + "/.github/workflows/release.yml@refs/tags/v1.0.0",
		"https://github.com/attacker/widget/.github/workflows/release.yml@refs/tags/v1.0.0",
		"https://github.com/" + repo + "-fork/.github/workflows/release.yml@refs/tags/v1.0.0",
		"https://github.com/" + repo + "/.github/workflows/",
		"https://evil.example/https://github.com/" + repo + "/.github/workflows/x.yml@refs/heads/main",
	}
	for _, san := range accept {
		if !re.MatchString(san) {
			t.Errorf("subjectRegExp %q should accept %q", subjectRegExp, san)
		}
	}
	for _, san := range reject {
		if re.MatchString(san) {
			t.Errorf("subjectRegExp %q must reject %q", subjectRegExp, san)
		}
	}
}

// TestGenerateCosignPolicy_Template checks that without a github.com origin the
// policy is a commented template that enforces nothing, rather than an
// enforcing policy with a guessed (wrong) identity.
func TestGenerateCosignPolicy_Template(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		remote     string
		wantReason string
	}{
		{"no remote", "", "no git origin remote was detected"},
		{"gitlab", "git@gitlab.com:acme/widget.git", "not a github.com repository"},
		{"github enterprise", "https://github.example.com/acme/widget.git", "not a github.com repository"},
		{"github org page", "https://github.com/acme", "not a github.com repository"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			content := string(cosignPolicyFor(t, tt.remote).Content)

			var doc any
			if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
				t.Fatalf("template is not valid YAML: %v", err)
			}
			if doc != nil {
				t.Errorf("template must contain no YAML object (applies nothing), got %v", doc)
			}
			for _, want := range []string{"NOT ENFORCING", tt.wantReason, "TODO-REGISTRY", "TODO-OIDC-ISSUER", "TODO-SIGNING-SUBJECT-REGEXP"} {
				if !strings.Contains(content, want) {
					t.Errorf("template should contain %q", want)
				}
			}
			for _, line := range strings.Split(strings.TrimRight(content, "\n"), "\n") {
				if !strings.HasPrefix(line, "#") {
					t.Errorf("template line %q is not commented out", line)
				}
			}
			if owner := branding.Get().GitHubOwner; strings.Contains(content, owner) {
				t.Errorf("template must not reference qsdev's own repository owner %q", owner)
			}
		})
	}
}
