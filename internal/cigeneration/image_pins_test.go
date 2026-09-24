package cigeneration

import (
	"regexp"
	"strings"
	"testing"
)

// allImagePins is the image catalog checked by the tests in this package. Keep
// it in step with the exported ImageRef values; TestImagePinsCoverCatalog
// fails if it drifts.
func allImagePins() map[string]ImageRef {
	return map[string]ImageRef{
		"ImageSnyk": ImageSnyk,
	}
}

// TestImagePinsCoverCatalog fails when an image pin is declared in the catalog
// but not added to allImagePins, which would leave it unverified against the
// registry, or when allImagePins names a pin the catalog no longer declares.
func TestImagePinsCoverCatalog(t *testing.T) {
	t.Parallel()
	assertCoversCatalog(t, "ImageRef", allImagePins())
}

// TestImageRefsAreWellFormed is the offline half of the image pin checks. A
// tag, or a digest that is short or of another algorithm, is not an immutable
// pin; TestImagePinsResolveUpstream checks that a well-formed digest exists.
func TestImageRefsAreWellFormed(t *testing.T) {
	t.Parallel()

	digestRe := regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

	for name, ref := range allImagePins() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if ref.Repository == "" || strings.ContainsAny(ref.Repository, "@:") {
				t.Errorf("%s repository %q must be a bare repository with no tag or digest", name, ref.Repository)
			}
			if !digestRe.MatchString(ref.Digest) {
				t.Errorf("%s digest %q is not a full sha256 digest; tags are not immutable pins", name, ref.Digest)
			}
			if ref.Tag == "" {
				t.Errorf("%s has no Tag, so the emitted comment would not say which image runs", name)
			}
		})
	}
}

func TestImageRefFormatting(t *testing.T) {
	t.Parallel()

	ref := ImageRef{
		Repository: "example/tool",
		Digest:     "sha256:" + strings.Repeat("ab", 32),
		Tag:        "stable",
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"String", ref.String(), "docker://example/tool@sha256:" + strings.Repeat("ab", 32)},
		{"Comment", ref.Comment(), "# stable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
