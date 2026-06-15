package contentsign

import "time"

// KeyID is the 16-hex-digit Minisign key identifier (a non-secret hint used for
// display and reporting; trust is established by successful verification, not by
// matching the ID).
type KeyID string

// Verification status values reported in VerificationResult and provenance _meta.
const (
	// StatusSignedVerified means a detached signature was verified against a trusted key.
	StatusSignedVerified = "signed-verified"
	// StatusHashVerified means no signature exists but the content matched its recorded SHA-256.
	StatusHashVerified = "hash-verified"
	// StatusUnverified means the content is neither signed nor hash-checked.
	StatusUnverified = "unverified"
	// StatusFailed means verification was attempted and failed (tamper or untrusted key).
	StatusFailed = "failed"
)

// VerificationResult records the outcome of verifying one content file. A failed
// verification is an expected result (Verified=false), not a Go error.
type VerificationResult struct {
	Path      string    `json:"path"`
	Signed    bool      `json:"signed"`           // a detached .minisig file exists
	Verified  bool      `json:"verified"`         // signature valid for a trusted key
	KeyID     KeyID     `json:"key_id,omitempty"` // ID of the verifying key, when verified
	Trusted   bool      `json:"trusted"`          // verifying key is in the trusted set
	Status    string    `json:"status"`           // one of the Status* constants
	Reason    string    `json:"reason,omitempty"` // human-readable failure detail
	CheckedAt time.Time `json:"checked_at"`
}

// ContentManifestEntry is contentsign's own minimal view of a signable artifact.
// It is intentionally decoupled from mcpregistry.DocSetEntry to keep contentsign
// a leaf package with no domain coupling; callers map their type onto this one.
type ContentManifestEntry struct {
	Path   string // absolute path to the content file
	SHA256 string // optional recorded hash for hash-verification fallback
}

// SigPath returns the detached signature path: the content path plus ".minisig"
// (the Minisign sidecar convention).
func (e ContentManifestEntry) SigPath() string {
	return e.Path + ".minisig"
}

// SignOptions configures signing of a single file.
type SignOptions struct {
	KeyPath        string // path to the Minisign secret key file
	Password       string // password for an encrypted secret key ("" if unencrypted)
	TrustedComment string // authenticated comment embedded in the signature
	Force          bool   // overwrite an existing signature file
}

// VerifyOptions configures verification.
type VerifyOptions struct {
	TrustedKeys    []PublicKey // when empty, keys are loaded from DefaultTrustedKeysDir
	RequireTrusted bool        // treat signed-but-untrusted (or unsigned) as failure
}

// SanitizeOptions controls the Unicode sanitization pipeline. NormalizeNFKC is
// off by default because NFKC is lossy on technical content (e.g. the micro sign
// U+00B5 folds to Greek mu); it is enabled only for prose-only, code-aware use.
type SanitizeOptions struct {
	StripInvisible bool // zero-width, BiDi, tag chars, variation selectors, specials
	StripControl   bool // C0/C1 controls except \t \n \r
	StripHTML      bool // HTML comments and hidden elements
	NormalizeNFKC  bool // NFKC normalization (default false; lossy on technical text)
}

// DefaultSanitizeOptions returns the lossless download-time profile: strip
// invisible, control, and HTML constructs but do not apply NFKC.
func DefaultSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{
		StripInvisible: true,
		StripControl:   true,
		StripHTML:      true,
		NormalizeNFKC:  false,
	}
}

// SanitizeReport summarizes what a sanitization pass changed.
type SanitizeReport struct {
	NFKCChanged   bool           `json:"nfkc_changed"`
	RunesStripped int            `json:"runes_stripped"`
	Categories    map[string]int `json:"categories,omitempty"` // e.g. {"zero-width": 12, "tag": 3}
}

// DatamarkOptions controls the datamarking transform (prompt-injection defense).
type DatamarkOptions struct {
	RandomizeMarker    bool // pick a random PUA marker rune per invocation (default true)
	MarkerRune         rune // explicit marker when RandomizeMarker is false
	PreserveCodeBlocks bool // leave fenced code blocks unmarked (default true)
	PreserveInlineCode bool // leave inline `code` unmarked (default true)
	IncludeFraming     bool // wrap output in self-describing framing (default true)
	Source             string
	VerificationStatus string
	ContentHashPrefix  string
}

// DatamarkMetadata describes how content was datamarked, for round-tripping and framing.
type DatamarkMetadata struct {
	MarkerRune rune      `json:"-"`
	MarkerHex  string    `json:"marker_hex"` // e.g. "U+E042"
	Framed     bool      `json:"framed"`
	Timestamp  time.Time `json:"timestamp"`
}

// ProvenanceMetadata is the qsdev/-namespaced provenance attached to MCP
// documentation responses (wired in P32).
type ProvenanceMetadata struct {
	VerificationStatus string `json:"verification_status"` // one of the Status* constants
	ContentHash        string `json:"content_hash"`        // SHA-256 hex
	Source             string `json:"source"`              // e.g. "devdocs/go", "kiwix:..."
	LastVerified       string `json:"last_verified"`       // RFC 3339 timestamp
}

// ContentDiff summarizes the structural difference between two content versions.
type ContentDiff struct {
	AddedEntries    int   `json:"added_entries"`
	RemovedEntries  int   `json:"removed_entries"`
	ModifiedEntries int   `json:"modified_entries"`
	SizeChangeBytes int64 `json:"size_change_bytes"`
}
