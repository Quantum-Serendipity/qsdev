// Package cigeneration provides the canonical catalog of SHA-pinned GitHub
// Action references (ActionRef) shared by the workflow emitters in
// internal/gitworkflow, internal/teamreport and internal/profile.
//
// It no longer generates CI workflows itself: the former CIFragmentProducer /
// GenerateWorkflow machinery was unreachable dead code (wired to no producer in
// the fragment accumulator) and was removed. Project CI/security-scan workflows
// are generated solely via the infrastructure-profile path — see
// InfraProfile.ConfigFiles (internal/profile), invoked from
// DevenvGenerator.Generate (addons/devenv/generator.go).
//
// Every entry here must be referenced by an emitter. An unreferenced entry
// cannot be exercised by any test or workflow run, so nothing detects when its
// SHA rots — this catalog previously accumulated four entries whose SHAs did
// not resolve upstream at all. Add a pin when something emits it, not before.
package cigeneration

import "fmt"

// ActionRef is a SHA-pinned reference to a GitHub Action, combining the
// full owner/repo@SHA form with a human-readable tag for comments.
type ActionRef struct {
	Owner string
	Repo  string
	SHA   string
	Tag   string
}

// String returns the pinned reference: owner/repo@SHA.
func (a ActionRef) String() string {
	return fmt.Sprintf("%s/%s@%s", a.Owner, a.Repo, a.SHA)
}

// Comment returns a YAML-suitable comment with the human-readable tag.
func (a ActionRef) Comment() string {
	return "# " + a.Tag
}

// SHA-pinned action references.
//
// Each SHA is the commit that the named tag resolves to upstream, verified
// against the GitHub API rather than transcribed. A SHA that merely looks
// plausible is worse than no pin: it fails at workflow run time, long after
// review, and the tag comment beside it reads as authoritative.
//
// TestActionPinsMatchWorkflows keeps these in step with .github/workflows,
// which Dependabot updates and this file it cannot see.
var (
	ActionCheckout = ActionRef{
		Owner: "actions",
		Repo:  "checkout",
		SHA:   "de0fac2e4500dabe0009e67214ff5f5447ce83dd",
		Tag:   "v6.0.2",
	}
	ActionHardenRunner = ActionRef{
		Owner: "step-security",
		Repo:  "harden-runner",
		SHA:   "9af89fc71515a100421586dfdb3dc9c984fbf411",
		Tag:   "v2.19.4",
	}
	ActionUploadArtifact = ActionRef{
		Owner: "actions",
		Repo:  "upload-artifact",
		SHA:   "043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
		Tag:   "v7.0.1",
	}
	// Paired with ActionUploadArtifact by the team workflow emitter. Both are
	// on the v4+ artifact backend, which is not interoperable with v3 and
	// earlier; keep them on that generation together.
	ActionDownloadArtifact = ActionRef{
		Owner: "actions",
		Repo:  "download-artifact",
		SHA:   "3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c",
		Tag:   "v8.0.1",
	}
	ActionOSVScanner = ActionRef{
		Owner: "google",
		Repo:  "osv-scanner-action/osv-scanner-action",
		SHA:   "6e4298ebc4db23e847df9b2e2de2939d6f066c67",
		Tag:   "v2.5.1",
	}
	ActionGrype = ActionRef{
		Owner: "anchore",
		Repo:  "scan-action",
		SHA:   "1638637db639e0ade3258b51db49a9a137574c3e",
		Tag:   "v6",
	}
	// snyk/actions publishes no release tags; master is the documented
	// reference, so the SHA is the only thing actually pinning it.
	ActionSnyk = ActionRef{
		Owner: "snyk",
		Repo:  "actions",
		SHA:   "9cf6ca713d71123d2d229cc3d7f145b96ea3c518",
		Tag:   "master",
	}
	ActionLabeler = ActionRef{
		Owner: "actions",
		Repo:  "labeler",
		SHA:   "8558fd74291d67161a8a78ce36a881fa63b766a9",
		Tag:   "v5.0.0",
	}
)
