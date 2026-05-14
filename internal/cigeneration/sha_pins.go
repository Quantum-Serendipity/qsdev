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
// Each SHA corresponds to the tagged release listed in the Tag field.
// SHAs should be verified against the upstream repository before production use.
var (
	ActionCheckout = ActionRef{
		Owner: "actions",
		Repo:  "checkout",
		SHA:   "11bd71901bbe5b1630ceea73d27597364c9af683",
		Tag:   "v4.2.2",
	}
	ActionHardenRunner = ActionRef{
		Owner: "step-security",
		Repo:  "harden-runner",
		SHA:   "0634a2670c59f64b4a01f0f96f84700a4088b9f0",
		Tag:   "v2.12.0",
	}
	ActionUploadArtifact = ActionRef{
		Owner: "actions",
		Repo:  "upload-artifact",
		SHA:   "ea165f8d65b6e75b540449e92b4886f43607fa02",
		Tag:   "v4.6.2",
	}
	ActionUploadSarif = ActionRef{
		Owner: "github",
		Repo:  "codeql-action",
		SHA:   "ff0a06e83cb2de871e5a09832bc6a81e7276941f",
		Tag:   "v3.28.18",
	}
	ActionSemgrep = ActionRef{
		Owner: "semgrep",
		Repo:  "semgrep-action",
		SHA:   "713efdd6cf1eadd5a227fc536c7f5b1731d32ddd",
		Tag:   "v1.2.0",
	}
	ActionGrype = ActionRef{
		Owner: "anchore",
		Repo:  "scan-action",
		SHA:   "2c901ab7a2a0168b0ece4efe2ad1b30fc1135484",
		Tag:   "v6.2.0",
	}
	ActionSyft = ActionRef{
		Owner: "anchore",
		Repo:  "sbom-action",
		SHA:   "61119d458adab75f756bc0b9e4bde25725f86a7a",
		Tag:   "v0.17.2",
	}
	ActionCosignInstaller = ActionRef{
		Owner: "sigstore",
		Repo:  "cosign-installer",
		SHA:   "3454372be43b5347950ddf1e4e2dc289b3a532da",
		Tag:   "v3.8.2",
	}
	ActionOSVScanner = ActionRef{
		Owner: "google",
		Repo:  "osv-scanner-action",
		SHA:   "6e2fede655b48e4ef7e24ab4cd20395d5a41f515",
		Tag:   "v2.0.2",
	}
	ActionClaudeCodeReview = ActionRef{
		Owner: "anthropics",
		Repo:  "claude-code-action",
		SHA:   "a0d3e11e71effa3e3a6b47e60f4ff66e7f2e60e9",
		Tag:   "v1.0.0",
	}
	ActionLabeler = ActionRef{
		Owner: "actions",
		Repo:  "labeler",
		SHA:   "8558fd74291d67161a8a78ce36a881fa63b766a9",
		Tag:   "v5.0.0",
	}
)
