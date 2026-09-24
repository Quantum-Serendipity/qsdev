package selfupdate

//go:generate go run ./internal/gentrustedroot -o trusted_root.json

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
	maxBundleSize = 1 << 20 // 1 MB for sigstore bundle.

	sigstoreBundleName = "checksums.txt.sigstore.json"
)

// sigstoreTrustedRootJSON is the Sigstore public-good trusted root (Fulcio CA
// chain, Rekor and CT log keys, timestamp authority chain) pinned into the
// binary. It is the ONLY trust anchor for self-update signature verification:
// nothing is resolved from PATH, the network or the user's ~/.sigstore cache,
// so a shim, a tampered TUF cache or a compromised mirror cannot vouch for a
// release. Refresh it with `go generate ./internal/selfupdate`, which fetches
// it through an authenticated TUF update (see internal/gentrustedroot).
//
//go:embed trusted_root.json
var sigstoreTrustedRootJSON []byte

// loadTrustedRoot parses the embedded trusted root once per process.
var loadTrustedRoot = sync.OnceValues(func() (*root.TrustedRoot, error) {
	return root.NewTrustedRootFromJSON(sigstoreTrustedRootJSON)
})

// VerificationResult describes the outcome of Sigstore verification.
type VerificationResult struct {
	Verified bool
	Skipped  bool
	Message  string
}

// verifySigstoreBundle downloads (if the asset exists) and verifies the
// .sigstore.json bundle for checksums.txt in-process with sigstore-go.
//
// Behavior:
//   - If the bundle asset is not present in the release, returns Skipped (an
//     unsigned release); strict mode turns that into a refusal.
//   - If the bundle verifies against the embedded trusted root AND the exact
//     release-workflow identity for this tag, returns Verified.
//   - Any other outcome (download failure, malformed bundle, bad signature,
//     wrong identity, missing transparency-log proof) is an error: FAIL CLOSED.
var verifySigstoreBundle = verifySigstoreBundleImpl

func verifySigstoreBundleImpl(ctx context.Context, release *Release, checksumsPath, tmpDir string) (*VerificationResult, error) {
	// Find the bundle asset in the release.
	bundleURL := ""
	for _, a := range release.Assets {
		if a.Name == sigstoreBundleName {
			bundleURL = a.URL
			break
		}
	}
	if bundleURL == "" {
		return &VerificationResult{
			Skipped: true,
			Message: fmt.Sprintf("release %s publishes no %s signature bundle; its checksums cannot be authenticated", release.TagName, sigstoreBundleName),
		}, nil
	}

	bundlePath := filepath.Join(tmpDir, sigstoreBundleName)
	if err := downloadFile(ctx, bundleURL, bundlePath, maxBundleSize); err != nil {
		return nil, fmt.Errorf("downloading sigstore bundle: %w", err)
	}

	if err := verifyChecksumsBundle(release.TagName, bundlePath, checksumsPath); err != nil {
		return nil, fmt.Errorf("sigstore verification failed for %s: %w", release.TagName, err)
	}

	return &VerificationResult{
		Verified: true,
		Message:  "sigstore signature verified: checksums.txt is authentically signed by the release workflow",
	}, nil
}

// verifyChecksumsBundle verifies that bundlePath is a valid Sigstore bundle
// over the exact bytes of checksumsPath, issued by the pinned public-good
// Fulcio CA to this release's EXACT signing identity, with a Rekor inclusion
// proof, an embedded SCT and a trusted observer timestamp — the same checks
// `cosign verify-blob --certificate-identity` performs, done in-process.
func verifyChecksumsBundle(tag, bundlePath, checksumsPath string) error {
	if tag == "" {
		return errors.New("release has no tag to derive the signing identity from")
	}

	trustedRoot, err := loadTrustedRoot()
	if err != nil {
		return fmt.Errorf("loading embedded sigstore trusted root: %w", err)
	}

	b, err := bundle.LoadJSONFromPath(bundlePath)
	if err != nil {
		return fmt.Errorf("parsing bundle: %w", err)
	}

	verifier, err := verify.NewVerifier(trustedRoot,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return fmt.Errorf("creating verifier: %w", err)
	}

	identity, err := releaseCertificateIdentity(tag)
	if err != nil {
		return err
	}

	artifact, err := os.Open(checksumsPath)
	if err != nil {
		return fmt.Errorf("opening checksums: %w", err)
	}
	defer artifact.Close()

	if _, err := verifier.Verify(b, verify.NewPolicy(
		verify.WithArtifact(artifact),
		verify.WithCertificateIdentity(identity),
	)); err != nil {
		return err
	}
	return nil
}

// releaseCertificateIdentity returns the certificate identity a release's
// signature must carry. The SAN comes from branding.ReleaseWorkflowIdentity(tag)
// and pins both the release workflow file and the git ref (refs/tags/<tag>).
// It is matched EXACTLY (no regexp), so a signature produced by a different
// workflow or on a different ref is rejected (fail closed).
func releaseCertificateIdentity(tag string) (verify.CertificateIdentity, error) {
	issuer, san := branding.ReleaseWorkflowIdentity(tag)
	identity, err := verify.NewShortCertificateIdentity(issuer, "", san, "")
	if err != nil {
		return verify.CertificateIdentity{}, fmt.Errorf("building expected signing identity: %w", err)
	}
	return identity, nil
}

// logVerificationResult writes the verification outcome to stderr for user visibility.
func logVerificationResult(result *VerificationResult) {
	if result.Skipped {
		fmt.Fprintf(os.Stderr, "  [warning] %s; installing WITHOUT signature verification (--no-strict)\n", result.Message)
	} else if result.Verified {
		fmt.Fprintf(os.Stderr, "  [verified] %s\n", result.Message)
	}
}
