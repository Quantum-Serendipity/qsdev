package cigeneration

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// usesRe matches a SHA-pinned action reference in a workflow file, e.g.
//
//	uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
//
// The action path is captured whole, so subpath actions such as
// google/osv-scanner-action/osv-scanner-action compare correctly.
var usesRe = regexp.MustCompile(`uses:\s+(\S+)@([0-9a-f]{40})\s+#\s+(\S+)`)

type workflowPin struct {
	sha  string
	tag  string
	file string
}

// workflowPins collects every SHA-pinned action used by this repository's own
// workflows, keyed by action path.
func workflowPins(t *testing.T) map[string]workflowPin {
	t.Helper()

	dir := filepath.Join("..", "..", ".github", "workflows")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}

	pins := make(map[string]workflowPin)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yml" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		for _, m := range usesRe.FindAllStringSubmatch(string(b), -1) {
			pins[m[1]] = workflowPin{sha: m[2], tag: m[3], file: e.Name()}
		}
	}

	if len(pins) == 0 {
		t.Fatal("found no pinned actions in .github/workflows; the parser or the layout changed")
	}
	return pins
}

// TestActionPinsMatchWorkflows keeps this catalog in step with the repository's
// own workflows.
//
// Dependabot updates .github/workflows and cannot see this file, so every
// actions PR silently widens any gap between them. That is not hypothetical:
// ActionUploadArtifact sat at v4.6.2 while the workflows had moved to v7.0.1,
// and the emitted team workflow would have paired mismatched artifact majors.
//
// Only actions this repository actually uses are compared. Entries emitted
// solely into generated projects (Grype, Snyk, download-artifact) have no local
// counterpart to check against.
func TestActionPinsMatchWorkflows(t *testing.T) {
	t.Parallel()

	pins := workflowPins(t)

	catalog := map[string]ActionRef{
		"ActionCheckout":       ActionCheckout,
		"ActionHardenRunner":   ActionHardenRunner,
		"ActionUploadArtifact": ActionUploadArtifact,
		"ActionOSVScanner":     ActionOSVScanner,
		"ActionGrype":          ActionGrype,
		"ActionSnyk":           ActionSnyk,
		"ActionLabeler":        ActionLabeler,
		// ActionDownloadArtifact is emitted only into generated team
		// workflows, so it has no counterpart here.
	}

	compared := 0
	for name, ref := range catalog {
		path := ref.Owner + "/" + ref.Repo
		pin, used := pins[path]
		if !used {
			continue
		}
		compared++

		t.Run(name, func(t *testing.T) {
			if ref.SHA != pin.sha {
				t.Errorf("%s pins %s@%s but .github/workflows/%s uses @%s\n"+
					"Dependabot updates the workflow and not this catalog; bump the catalog to match.",
					name, path, ref.SHA, pin.file, pin.sha)
			}
			if ref.Tag != pin.tag {
				t.Errorf("%s is tagged %q but .github/workflows/%s comments %q",
					name, ref.Tag, pin.file, pin.tag)
			}
		})
	}

	if compared == 0 {
		t.Error("compared no pins; the catalog and the workflows no longer overlap, so this test guards nothing")
	}
}

// TestActionRefsAreWellFormed catches pins that cannot resolve without needing
// network access.
//
// This catalog previously carried four entries whose SHAs did not exist
// upstream at all — two of them shared a prefix with the real commit and then
// diverged, which reads as correct under review and only fails when a workflow
// runs. Shape checks cannot catch a well-formed but wrong SHA;
// TestActionPinsResolveUpstream does that against the GitHub API.
func TestActionRefsAreWellFormed(t *testing.T) {
	t.Parallel()

	sha40 := regexp.MustCompile(`^[0-9a-f]{40}$`)

	for name, ref := range map[string]ActionRef{
		"ActionCheckout":         ActionCheckout,
		"ActionHardenRunner":     ActionHardenRunner,
		"ActionUploadArtifact":   ActionUploadArtifact,
		"ActionDownloadArtifact": ActionDownloadArtifact,
		"ActionOSVScanner":       ActionOSVScanner,
		"ActionGrype":            ActionGrype,
		"ActionSnyk":             ActionSnyk,
		"ActionLabeler":          ActionLabeler,
	} {
		t.Run(name, func(t *testing.T) {
			if ref.Owner == "" || ref.Repo == "" {
				t.Errorf("%s has an empty Owner or Repo", name)
			}
			if !sha40.MatchString(ref.SHA) {
				t.Errorf("%s SHA %q is not a full 40-character commit SHA; "+
					"short SHAs and tags are not immutable pins", name, ref.SHA)
			}
			if ref.Tag == "" {
				t.Errorf("%s has no Tag, so the emitted comment would not say which version runs", name)
			}
		})
	}
}
