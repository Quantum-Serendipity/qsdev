package cigeneration

// ImageRef is a digest-pinned reference to a container image that an emitted
// workflow step runs directly (`uses: docker://...`).
//
// An ActionRef pins only an action's metadata. A Docker action whose
// action.yml names an image by tag runs whatever that tag points at when the
// job starts, so its SHA pin does not pin the code that executes. Steps that
// run such an image are emitted as container steps pinned by digest instead.
type ImageRef struct {
	// Repository is the Docker Hub repository, e.g. "snyk/snyk". It carries
	// neither a tag nor a digest.
	Repository string
	// Digest is the image index (or manifest) digest, "sha256:<64 hex>".
	Digest string
	// Tag is the tag Digest was taken from. It is recorded for the emitted
	// comment and for drift checks; it is not part of the pin.
	Tag string
}

// String returns the pinned reference in workflow `uses:` form:
// docker://repository@digest.
func (r ImageRef) String() string {
	return "docker://" + r.Repository + "@" + r.Digest
}

// Comment returns a YAML-suitable comment naming the tag the digest came from.
func (r ImageRef) Comment() string {
	return "# " + r.Tag
}

// Digest-pinned image references.
//
// Each digest is what the named tag resolved to in the registry when it was
// recorded, verified against the registry rather than transcribed. The tags
// themselves are mutable (upstream rebuilds them), so the digest is the whole
// of the pin. TestImagePinsResolveUpstream checks that every digest still
// resolves and fails once a pin has fallen too far behind its tag.
var (
	// ImageSnyk is the Snyk CLI image the snyk/actions root action runs
	// (docker://snyk/snyk:node). It is emitted as a container step because
	// that action.yml names the image by its mutable tag, so a SHA-pinned
	// snyk/actions reference still ran unpinned code with SNYK_TOKEN.
	//
	// The node variant carries the Snyk CLI (copied into the image, so the
	// digest pins it) but no pip, Maven or Go toolchain. The image entrypoint
	// installs a project's dependencies before scanning when those tools are
	// present (pip install -r requirements.txt, mvn install, a curl | sh dep
	// install), so the language-specific variants would run the scanned
	// repository's install hooks with SNYK_TOKEN in the environment.
	ImageSnyk = ImageRef{
		Repository: "snyk/snyk",
		Digest:     "sha256:0827c5acdea5f2b9ddc94c9c2d423493ae9f40f1315850e8626141fb04e6f42b",
		Tag:        "node",
	}
)
