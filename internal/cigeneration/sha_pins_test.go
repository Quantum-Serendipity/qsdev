package cigeneration

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoWorkflowsDir is this repository's own workflow directory, relative to
// the package directory tests run in.
var repoWorkflowsDir = filepath.Join("..", "..", ".github", "workflows")

// repoWorkflowPins parses the repository's own workflows.
func repoWorkflowPins(t *testing.T) map[string]WorkflowPin {
	t.Helper()

	pins, err := ParseWorkflowPins(repoWorkflowsDir)
	if err != nil {
		t.Fatal(err)
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
// Every catalog entry the repository's workflows also use is compared; the
// rest are emitted solely into generated projects and have no local
// counterpart. The dependabot-fixup workflow runs go generate on actions
// branches, which applies SyncActionPins and keeps this test green.
func TestActionPinsMatchWorkflows(t *testing.T) {
	t.Parallel()

	pins := repoWorkflowPins(t)

	compared := 0
	for name, ref := range AllActionRefs() {
		path := ref.Owner + "/" + ref.Repo
		pin, used := pins[path]
		if !used {
			continue
		}
		compared++

		t.Run(name, func(t *testing.T) {
			if ref.SHA != pin.SHA {
				t.Errorf("%s pins %s@%s but .github/workflows/%s uses @%s\n"+
					"Dependabot updates the workflow and not this catalog; run go generate ./internal/cigeneration/",
					name, path, ref.SHA, pin.File, pin.SHA)
			}
			if ref.Tag != pin.Tag {
				t.Errorf("%s is tagged %q but .github/workflows/%s comments %q; run go generate ./internal/cigeneration/",
					name, ref.Tag, pin.File, pin.Tag)
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

	for name, ref := range AllActionRefs() {
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

// exactVersionRe matches an exact release tag such as v2.17.1.
var exactVersionRe = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// TestGoreleaserVersionPinned is the W191 regression test. The action is SHA
// pinned, but its `version` input chooses the goreleaser binary it downloads,
// and the release job runs that binary with the signing identity and tap
// tokens. A range such as "~> v2" runs whatever was published last, and CI
// would not even validate the version the release uses. The input resolves
// through tool-versions.env (U26-02), which must hold an exact release.
func TestGoreleaserVersionPinned(t *testing.T) {
	t.Parallel()

	toolVersions := readToolVersions(t)
	versions := map[string]string{} // version -> first workflow using it
	for file, wf := range readRepoWorkflows(t) {
		for jobName, job := range wf.Jobs {
			for _, s := range job.Steps {
				if !strings.HasPrefix(s.Uses, "goreleaser/goreleaser-action@") {
					continue
				}
				where := file + " job " + jobName
				ref := toolEnvRefRe.FindStringSubmatch(s.With["version"])
				if ref == nil {
					t.Errorf("%s: goreleaser-action version %q, want ${{ env.KEY }} from %s", where, s.With["version"], toolVersionsFile)
					continue
				}
				v, ok := toolVersions[ref[1]]
				if !ok {
					t.Errorf("%s: goreleaser-action version env.%s is not in %s", where, ref[1], toolVersionsFile)
					continue
				}
				if !exactVersionRe.MatchString(v) {
					t.Errorf("%s: goreleaser version %s=%q is not an exact release (want vX.Y.Z)", where, ref[1], v)
				}
				if _, seen := versions[v]; !seen {
					versions[v] = file
				}
			}
		}
	}
	if len(versions) == 0 {
		t.Fatal("found no goreleaser-action steps; the parser or the layout changed")
	}
	if len(versions) > 1 {
		t.Errorf("workflows use different goreleaser versions %v; CI must validate the version the release runs", versions)
	}
}

// TestAllActionRefsCoversExportedVars fails when an ActionRef is declared in
// the catalog but missing from AllActionRefs, or the reverse. Every check that
// iterates the registry (drift, shape, upstream resolution) would otherwise
// skip that pin silently, the state that let four unresolvable SHAs and an
// unchecked ActionInstallNix accumulate.
func TestAllActionRefsCoversExportedVars(t *testing.T) {
	t.Parallel()
	assertCoversCatalog(t, "ActionRef", AllActionRefs())
}
