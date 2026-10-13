package secrets

import (
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

// TestCredentialPaths_ReturnsCopy pins that a caller cannot mutate the canon.
func TestCredentialPaths_ReturnsCopy(t *testing.T) {
	t.Parallel()

	first := CredentialPaths()
	first[0] = "mutated"
	if CredentialPaths()[0] == "mutated" {
		t.Error("CredentialPaths returned a shared slice; mutating it changed the canon")
	}
}
