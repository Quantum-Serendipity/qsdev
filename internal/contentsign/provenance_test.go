package contentsign

import (
	"testing"
	"time"
)

func TestProvenanceMetadataToMeta(t *testing.T) {
	t.Parallel()

	p := ProvenanceMetadata{
		VerificationStatus: StatusSignedVerified,
		ContentHash:        "deadbeef",
		Source:             "devdocs/go",
		LastVerified:       "2026-06-15T10:00:00Z",
	}
	meta := p.ToMeta()

	want := map[string]string{
		"qsdev/verificationStatus": StatusSignedVerified,
		"qsdev/contentHash":        "deadbeef",
		"qsdev/source":             "devdocs/go",
		"qsdev/lastVerified":       "2026-06-15T10:00:00Z",
	}
	if len(meta) != len(want) {
		t.Fatalf("ToMeta returned %d keys, want %d: %v", len(meta), len(want), meta)
	}
	for k, wantVal := range want {
		got, ok := meta[k]
		if !ok {
			t.Errorf("missing key %q", k)
			continue
		}
		if got != wantVal {
			t.Errorf("key %q = %v, want %q", k, got, wantVal)
		}
	}
}

func TestBuildProvenanceMetadata(t *testing.T) {
	t.Parallel()

	checkedAt := time.Date(2026, 6, 15, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		status string
	}{
		{name: "signed verified", status: StatusSignedVerified},
		{name: "hash verified", status: StatusHashVerified},
		{name: "unverified", status: StatusUnverified},
		{name: "failed", status: StatusFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := VerificationResult{
				Status:    tc.status,
				CheckedAt: checkedAt,
			}
			p := BuildProvenanceMetadata(result, "devdocs/go", "cafef00d")

			if p.VerificationStatus != tc.status {
				t.Errorf("VerificationStatus = %q, want %q", p.VerificationStatus, tc.status)
			}
			if p.Source != "devdocs/go" {
				t.Errorf("Source = %q, want %q", p.Source, "devdocs/go")
			}
			if p.ContentHash != "cafef00d" {
				t.Errorf("ContentHash = %q, want %q", p.ContentHash, "cafef00d")
			}

			parsed, err := time.Parse(time.RFC3339, p.LastVerified)
			if err != nil {
				t.Fatalf("LastVerified %q is not RFC 3339: %v", p.LastVerified, err)
			}
			if !parsed.Equal(checkedAt) {
				t.Errorf("LastVerified parsed to %v, want %v", parsed, checkedAt)
			}
		})
	}
}
