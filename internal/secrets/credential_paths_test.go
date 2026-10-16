package secrets

import (
	"path"
	"slices"
	"strings"
	"testing"
)

// formerHomeDenyRelPaths is a frozen copy of the credential canon. It began as
// the list internal/sandbox/denylist.HomeDenyRelPaths held before it moved
// here, and was deliberately extended with the package-registry token stores
// (U09-WS2) and the remaining per-user credential stores (U19-WS5). Any change
// to the canon must update this copy on purpose.
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
	// U09-WS2: package-registry token stores.
	".npmrc",
	".yarnrc.yml",
	".yarnrc",
	".bunfig.toml",
	".config/.bunfig.toml",
	".pypirc",
	".config/pip/pip.conf",
	".pip/pip.conf",
	"Library/Application Support/pip/pip.conf",
	"AppData/Roaming/pip/pip.ini",
	".gem/credentials",
	".local/share/gem/credentials",
	".bundle/config",
	".config/composer/auth.json",
	".composer/auth.json",
	"AppData/Roaming/Composer/auth.json",
	".local/share/pypoetry/auth.toml",
	// U19-WS5: further per-user credential stores.
	".m2/settings.xml",
	".gradle/gradle.properties",
	".config/gh",
	".git-credentials",
	".pgpass",
	".vault-token",
	".claude/.credentials.json",
}

func TestCredentialPaths_SameAsFormerHomeDenyRelPaths(t *testing.T) {
	t.Parallel()

	got := slices.Clone(CredentialPaths())
	want := slices.Clone(formerHomeDenyRelPaths)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("CredentialPaths() = %q, want the frozen canon %q", got, want)
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

// TestCredentialStores_DirFlag pins which stores are directories (their
// contents are credentials too) and that CredentialPaths is the Rel
// projection of CredentialStores in the same order, without duplicates.
func TestCredentialStores_DirFlag(t *testing.T) {
	t.Parallel()

	wantDirs := []string{
		".ssh", ".gnupg", ".aws", ".azure", ".config/gcloud", ".kube",
		".config/helm/registry", ".config/gh",
	}
	stores := CredentialStores()
	var dirs, rels []string
	for _, s := range stores {
		rels = append(rels, s.Rel)
		if s.Dir {
			dirs = append(dirs, s.Rel)
		}
	}
	slices.Sort(dirs)
	slices.Sort(wantDirs)
	if !slices.Equal(dirs, wantDirs) {
		t.Errorf("Dir stores = %q, want %q", dirs, wantDirs)
	}
	if got := CredentialPaths(); !slices.Equal(got, rels) {
		t.Errorf("CredentialPaths() = %q, want the Rel projection of CredentialStores() %q", got, rels)
	}
	if uniq := slices.Compact(slices.Sorted(slices.Values(rels))); len(uniq) != len(rels) {
		t.Errorf("CredentialStores has duplicate Rel entries: %q", rels)
	}
	stores[0].Rel = "mutated"
	if CredentialStores()[0].Rel == "mutated" {
		t.Error("CredentialStores returned a shared slice; mutating it changed the canon")
	}
}

// TestCredentialPaths_RegistryTokens pins the package-registry token stores
// (npm, Yarn, Bun, PyPI, pip, RubyGems, Bundler, Composer, Poetry, Cargo).
func TestCredentialPaths_RegistryTokens(t *testing.T) {
	t.Parallel()

	assertContainsAll(t, CredentialPaths(),
		".npmrc", ".yarnrc.yml", ".yarnrc", ".bunfig.toml", ".config/.bunfig.toml",
		".pypirc", ".config/pip/pip.conf", ".pip/pip.conf",
		"Library/Application Support/pip/pip.conf", "AppData/Roaming/pip/pip.ini",
		".gem/credentials", ".local/share/gem/credentials", ".bundle/config",
		".config/composer/auth.json", ".composer/auth.json", "AppData/Roaming/Composer/auth.json",
		".local/share/pypoetry/auth.toml", ".cargo/credentials.toml", ".cargo/credentials",
	)
}

// TestCredentialPaths_UserStores pins the per-user tool credential stores,
// and that the whole ~/.claude directory is not one of them (only its
// credentials file is).
func TestCredentialPaths_UserStores(t *testing.T) {
	t.Parallel()

	got := CredentialPaths()
	assertContainsAll(t, got,
		".m2/settings.xml", ".gradle/gradle.properties", ".config/gh", ".git-credentials",
		".pgpass", ".vault-token", ".claude/.credentials.json",
	)
	if slices.Contains(got, ".claude") {
		t.Error("CredentialPaths() contains the whole .claude directory; only .claude/.credentials.json belongs")
	}
}

func TestCredentialHomePaths(t *testing.T) {
	t.Parallel()

	stores := CredentialStores()
	got := CredentialHomePaths()
	if len(got) != len(stores) {
		t.Fatalf("CredentialHomePaths() has %d entries, want %d", len(got), len(stores))
	}
	for i, s := range stores {
		if want := "~/" + s.Rel; got[i] != want {
			t.Errorf("CredentialHomePaths()[%d] = %q, want %q", i, got[i], want)
		}
	}
}

func TestCredentialReadGlobs(t *testing.T) {
	t.Parallel()

	stores := CredentialStores()
	got := CredentialReadGlobs()
	if len(got) != len(stores) {
		t.Fatalf("CredentialReadGlobs() has %d entries, want %d", len(got), len(stores))
	}
	for i, s := range stores {
		want := "~/" + s.Rel
		if s.Dir {
			want += "/**"
		}
		if got[i] != want {
			t.Errorf("CredentialReadGlobs()[%d] = %q, want %q", i, got[i], want)
		}
	}
	assertContainsAll(t, got, "~/.ssh/**", "~/.config/gh/**", "~/.npmrc", "~/.docker/config.json")
}

func assertContainsAll(t *testing.T, got []string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("%q lacks %q", got, w)
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

// TestSecretPatternClasses pins the split of the secret-file canon: the
// material and data classes are disjoint, together they are exactly
// SecretFilePatterns, and each accessor returns a fresh slice.
func TestSecretPatternClasses(t *testing.T) {
	t.Parallel()

	material, data := SecretMaterialPatterns(), SecretDataPatterns()
	if len(material) == 0 || len(data) == 0 {
		t.Fatalf("empty class: material %q, data %q", material, data)
	}
	for _, p := range material {
		if slices.Contains(data, p) {
			t.Errorf("%q is in both the material and the data class", p)
		}
	}
	union := slices.Concat(material, data)
	all := SecretFilePatterns()
	slices.Sort(union)
	slices.Sort(all)
	if !slices.Equal(union, all) {
		t.Errorf("material and data = %q, SecretFilePatterns() = %q", union, all)
	}

	tests := []struct {
		name string
		get  func() []string
	}{
		{"SecretMaterialPatterns", SecretMaterialPatterns},
		{"SecretDataPatterns", SecretDataPatterns},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.get()
			got[0] = "mutated"
			if tt.get()[0] == "mutated" {
				t.Errorf("%s returned a shared slice; mutating it changed the canon", tt.name)
			}
		})
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
