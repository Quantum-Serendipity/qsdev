package secrets

import (
	"path"
	"slices"
	"strings"
	"testing"
)

// formerHomeDenyRelPaths is a frozen copy of the list that
// internal/sandbox/denylist.HomeDenyRelPaths held before it moved here. The
// move must not change the content; extending the list is a separate change
// that updates this copy deliberately.
var formerHomeDenyRelPaths = []string{
	".ssh",
	".gnupg",
	".aws",
	".azure",
	".config/gcloud",
	".kube",
	".docker/config.json",
	".netrc",
	".cargo/credentials.toml",
	".cargo/credentials",
	".nuget/NuGet/NuGet.Config",
	".config/NuGet/NuGet.Config",
	".config/containers/auth.json",
	".terraform.d/credentials.tfrc.json",
	".config/helm/registry",
	".config/helm/repositories.yaml",
}

func TestCredentialPaths_SameAsFormerHomeDenyRelPaths(t *testing.T) {
	t.Parallel()

	got := slices.Clone(CredentialPaths())
	want := slices.Clone(formerHomeDenyRelPaths)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("CredentialPaths() = %q, want the former HomeDenyRelPaths set %q", got, want)
	}
}

func TestCredentialPaths_Shape(t *testing.T) {
	t.Parallel()

	for _, p := range CredentialPaths() {
		switch {
		case p == "", strings.HasPrefix(p, "/"), strings.HasPrefix(p, "~"):
			t.Errorf("entry %q is not home-relative", p)
		case strings.Contains(p, `\`):
			t.Errorf("entry %q is not slash-separated", p)
		case strings.Contains("/"+p+"/", "/../"), strings.HasSuffix(p, "/"):
			t.Errorf("entry %q is not a clean relative path", p)
		}
	}
}

// TestSecretFilePatterns pins the secret-material file names a secrets
// directory is guarded for: each is a single-segment glob, the required
// credential and config shapes are present, and source code is not matched.
func TestSecretFilePatterns(t *testing.T) {
	t.Parallel()

	got := SecretFilePatterns()
	for _, want := range []string{".env", ".env.*", "*.env", "*.key", "*.pem", "*.p12", "*.pfx",
		"*.json", "*.yaml", "*.yml", "*.toml", "*.txt"} {
		if !slices.Contains(got, want) {
			t.Errorf("SecretFilePatterns() lacks %q", want)
		}
	}
	for _, p := range got {
		if p == "" || strings.Contains(p, "/") || strings.Contains(p, `\`) {
			t.Errorf("entry %q is not a single path segment", p)
		}
		if _, err := path.Match(p, ""); err != nil {
			t.Errorf("entry %q is not a valid glob: %v", p, err)
		}
		for _, src := range []string{"patterns.go", "known_vars_test.go", "README.md", "doc.go"} {
			if ok, _ := path.Match(p, src); ok {
				t.Errorf("entry %q matches source file %s", p, src)
			}
		}
	}
	got[0] = "mutated"
	if SecretFilePatterns()[0] == "mutated" {
		t.Error("SecretFilePatterns returned a shared slice; mutating it changed the canon")
	}
}

// TestCredentialPaths_ReturnsCopy pins that a caller cannot mutate the canon.
func TestCredentialPaths_ReturnsCopy(t *testing.T) {
	t.Parallel()

	first := CredentialPaths()
	first[0] = "mutated"
	if CredentialPaths()[0] == "mutated" {
		t.Error("CredentialPaths returned a shared slice; mutating it changed the canon")
	}
}
