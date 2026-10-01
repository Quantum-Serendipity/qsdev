package canon

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// withOrgConfig points the org-overlay variable at a symlink to a file that
// does not exist yet and re-runs ensureInit, returning the link and its
// canonical target. Callers must not be parallel: the environment and the
// protected-prefix table are process-wide, and the cleanup rebuilds the table
// without the variable.
func withOrgConfig(t *testing.T) (link, target string) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target = filepath.Join(dir, "org", "defaults.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(t.TempDir(), "org-defaults.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv(orgConfigEnv(branding.Get()), link)
	resetInit := func() {
		initOnce = sync.Once{}
		initErr = nil
		protectedPrefixes = nil
		protectedSuffixes = nil
	}
	resetInit()
	t.Cleanup(resetInit)
	return link, target
}

// TestIsProtected_GeneratorInputs pins the protection of the files the
// Claude settings generator reads: the state directory (answers and state
// manifest), the devenv answers mirror, .envrc and the org overlay. Editing
// any of them could steer the next regeneration, so the agent must not.
func TestIsProtected_GeneratorInputs(t *testing.T) {
	link, target := withOrgConfig(t)

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}
	b := branding.Get()
	proj := filepath.FromSlash("/work/repo")

	tests := []struct {
		name     string
		path     string
		wantProt bool
		wantCat  string
	}{
		{"answers file", filepath.Join(proj, b.StateDir, "."+b.AppName+"-init-answers.yaml"), true, "answers"},
		{"state manifest", filepath.Join(proj, b.StateDir, "."+b.AppName+"-init-state.yaml"), true, "answers"},
		{"state dir itself", filepath.Join(proj, b.StateDir), true, "answers"},
		{"devenv answers copy", filepath.Join(proj, ".devenv", "."+b.AppName+"-answers.yaml"), true, "answers"},
		{"envrc", filepath.Join(proj, ".envrc"), true, "config"},
		{"home org overlay", filepath.Join(home, ".config", b.AppName, "defaults.yaml"), true, "config"},
		{"home org overlay dir", filepath.Join(home, ".config", b.AppName, "other.yaml"), true, "config"},
		{"org config env link", link, true, "config"},
		{"org config env target", target, true, "config"},

		// Lookalikes stay writable.
		{"embedded state dir name", filepath.Join(proj, "my"+b.StateDir, "x"), false, ""},
		{"envrc backup", filepath.Join(proj, "x.envrc.bak"), false, ""},
		{"envrc suffix", filepath.Join(proj, ".envrc.bak"), false, ""},
		{"devenv runtime", filepath.Join(proj, ".devenv", "profile", "bin", "x"), false, ""},
		{"devenv state", filepath.Join(proj, ".devenv", "state", "x"), false, ""},
		{"other app config", filepath.Join(home, ".config", b.AppName+"-other", "defaults.yaml"), false, ""},
		{"org target sibling", filepath.Join(filepath.Dir(target), "notes.yaml"), false, ""},
	}
	for _, tt := range tests {
		if gotProt, gotCat := IsProtected(tt.path); gotProt != tt.wantProt || gotCat != tt.wantCat {
			t.Errorf("%s: IsProtected(%q) = (%v, %q), want (%v, %q)",
				tt.name, tt.path, gotProt, gotCat, tt.wantProt, tt.wantCat)
		}
	}
}

// TestContainsProtectedPath_GeneratorInputs checks that the raw-command scan
// agrees with IsProtected on the generator inputs, so a Bash mutation is
// caught as well as a Write.
func TestContainsProtectedPath_GeneratorInputs(t *testing.T) {
	t.Parallel()
	b := branding.Get()
	tests := []struct {
		input string
		want  bool
	}{
		{"sed -i s/true/false/ " + b.StateDir + "/." + b.AppName + "-init-answers.yaml", true},
		{"rm -rf " + b.StateDir, true},
		{"echo x > .devenv/." + b.AppName + "-answers.yaml", true},
		{"echo x >> .envrc", true},
		{"cat x.envrc.bak", false},
		{"ls .devenv/profile/bin/x", false},
		{"cat my" + b.StateDir + "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := ContainsProtectedPath(tt.input); got != tt.want {
				t.Errorf("ContainsProtectedPath(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestIsProtected_GeneratorInputs_WhiteLabel checks that every generator-input
// location follows the branding a white-label build sets, rather than the
// qsdev names.
func TestIsProtected_GeneratorInputs_WhiteLabel(t *testing.T) {
	t.Parallel()
	cfg := branding.Default()
	cfg.StateDir = ".acme-state"
	cfg.AppName = "acme"
	cfg.EnvPrefix = "ACME_"
	org := filepath.FromSlash("/srv/org/acme.yaml")
	getenv := func(k string) string {
		if k == "ACME_ORG_CONFIG" {
			return org
		}
		return ""
	}
	tables := newPathTables(cfg, getenv)

	segTests := []struct {
		key     string
		wantCat string
	}{
		{"/work/repo/.acme-state/.acme-init-answers.yaml", "answers"},
		{"/work/repo/.acme-state", "answers"},
		{"/work/repo/.devenv/.acme-answers.yaml", "answers"},
		{"/work/repo/.envrc", "config"},
		{"/work/repo/.devinit/.qsdev-init-answers.yaml", ""},
		{"/work/repo/.devenv/.qsdev-answers.yaml", ""},
	}
	for _, tt := range segTests {
		if got := tables.segmentCategory(tt.key, false); got != tt.wantCat {
			t.Errorf("segmentCategory(%q) = %q, want %q", tt.key, got, tt.wantCat)
		}
	}

	scanTests := []struct {
		input string
		want  bool
	}{
		{"sed -i x .acme-state/.acme-init-answers.yaml", true},
		{"rm -rf .acme-state", true},
		{"echo x > .devenv/.acme-answers.yaml", true},
		{"echo x > .devenv/.qsdev-answers.yaml", false},
		// The org overlay: its home-relative directory, the variable naming
		// it, and the file the variable names.
		{"U=~; echo x > $U/.config/acme/defaults.yaml", true},
		{`echo x >> "$ACME_ORG_CONFIG"`, true},
		{"sh -c 'echo x > /srv/org/acme.yaml'", true},
		{"echo x > ~/.config/acmex/defaults.yaml", false},
		{"echo $MY_ACME_ORG_CONFIG", false},
		{"echo x > ~/.config/qsdev/defaults.yaml", false},
		{"echo $QSDEV_ORG_CONFIG", false},
	}
	for _, tt := range scanTests {
		if got := tables.containsProtectedPath(tt.input, false); got != tt.want {
			t.Errorf("containsProtectedPath(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}

	for _, name := range tables.tokens {
		if name == "ACME_ORG_CONFIG" {
			t.Errorf("tokens (and so ProtectedNames) hold the variable name %q", name)
		}
	}

	home := filepath.FromSlash("/home/alice")
	var got []string
	for _, e := range orgOverlayEntries(cfg, home, getenv) {
		if e.category != "config" {
			t.Errorf("org overlay entry %q category = %q, want config", e.path, e.category)
		}
		got = append(got, e.path)
	}
	wantDir := filepath.Join(home, ".config", "acme") + string(filepath.Separator)
	if !slices.Contains(got, wantDir) {
		t.Errorf("org overlay entries %q lack %q", got, wantDir)
	}
	wantOrg, err := filepath.Abs(org)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(got, wantOrg) {
		t.Errorf("org overlay entries %q lack %q", got, wantOrg)
	}
}

// TestGeneratorInputsMatchWriters cross-checks canon's derivation against the
// packages that write the generator inputs, so a renamed file cannot silently
// fall out of protection. catalog.OrgConfigPath skips its home fallback in a
// test binary, so the overlay is checked through the environment variable.
func TestGeneratorInputsMatchWriters(t *testing.T) {
	withOrgConfig(t)
	proj := filepath.FromSlash("/work/repo")

	tests := []struct {
		writer  string
		path    string
		wantCat string
	}{
		{"answers.PrimaryPath", answers.PrimaryPath(proj), "answers"},
		{"answers.DevenvCopyFile", filepath.Join(proj, filepath.FromSlash(answers.DevenvCopyFile())), "answers"},
		{"state.InitStateFile", filepath.Join(proj, filepath.FromSlash(state.InitStateFile())), "answers"},
		{"catalog.OrgConfigPath", catalog.OrgConfigPath(), "config"},
	}
	for _, tt := range tests {
		if strings.TrimSpace(tt.path) == "" {
			t.Errorf("%s returned an empty path", tt.writer)
			continue
		}
		p, err := Canonicalize(tt.path)
		if err != nil {
			t.Fatalf("Canonicalize(%q): %v", tt.path, err)
		}
		for _, candidate := range []string{tt.path, p} {
			if got, cat := IsProtected(candidate); !got || cat != tt.wantCat {
				t.Errorf("%s: IsProtected(%q) = (%v, %q), want (true, %q)", tt.writer, candidate, got, cat, tt.wantCat)
			}
		}
	}
}

// TestFindProbes_CoverGeneratorInputs verifies the find probes are derived
// from the protected tables: every generator input, project-relative or
// home-anchored, is a probe, and every probe below a protected directory is
// itself protected, so a find name pattern is tried against real targets.
func TestFindProbes_CoverGeneratorInputs(t *testing.T) {
	link, _ := withOrgConfig(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("getting home dir: %v", err)
	}
	b := branding.Get()
	relative, absolute := FindProbes()

	for _, want := range []string{
		answers.PrimaryDir() + "/" + answers.PrimaryFilename(),
		answers.PrimaryDir(),
		answers.DevenvCopyFile(),
		".envrc",
		".claude",
		".claude/settings.json",
	} {
		if !slices.Contains(relative, want) {
			t.Errorf("relative probes lack %q:\n%s", want, strings.Join(relative, "\n"))
		}
	}
	for _, want := range []string{
		filepath.ToSlash(filepath.Join(home, ".config", b.AppName, "defaults.yaml")),
		filepath.ToSlash(link),
	} {
		if !slices.Contains(absolute, want) {
			t.Errorf("absolute probes lack %q:\n%s", want, strings.Join(absolute, "\n"))
		}
	}
	proj := filepath.FromSlash("/work/repo")
	for _, rel := range relative {
		p := filepath.Join(proj, filepath.FromSlash(rel))
		if strings.Contains(rel, "/") && !isProtectedOrAncestor(p) {
			t.Errorf("relative probe %q is neither protected nor above a protected entry", rel)
		}
	}
}

// isProtectedOrAncestor reports whether p is protected or is a directory
// holding a protected segment.
func isProtectedOrAncestor(p string) bool {
	if ok, _ := IsProtected(p); ok {
		return true
	}
	ok, _ := IsProtected(filepath.Join(p, "x"))
	return ok
}
