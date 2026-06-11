package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

const (
	maxBundleSize = 1 << 20 // 1 MB for sigstore bundle.

	sigstoreBundleName = "checksums.txt.sigstore.json"

	expectedOIDCIssuer = "https://token.actions.githubusercontent.com"
)

// VerificationResult describes the outcome of Sigstore verification.
type VerificationResult struct {
	Verified bool
	Skipped  bool
	Message  string
}

// verifySigstoreBundle downloads (if the asset exists) and verifies the
// .sigstore.json bundle for checksums.txt using cosign.
//
// Behavior:
//   - If the bundle asset is not present in the release, returns Skipped (old release).
//   - If cosign is not on PATH, returns Skipped with advisory message.
//   - If cosign verifies successfully, returns Verified.
//   - If cosign is present AND bundle exists BUT verification fails, returns error (FAIL CLOSED).
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
			Message: "sigstore bundle not found in release; skipping signature verification",
		}, nil
	}

	// Check if cosign is available on PATH.
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return &VerificationResult{
			Skipped: true,
			Message: "cosign not found on PATH; skipping signature verification (install cosign for enhanced security)",
		}, nil
	}

	// Download the bundle.
	bundlePath := tmpDir + "/" + sigstoreBundleName
	if err := downloadFile(ctx, bundleURL, bundlePath, maxBundleSize); err != nil {
		return nil, fmt.Errorf("downloading sigstore bundle: %w", err)
	}

	// Construct the expected certificate identity from the release tag.
	b := branding.Get()
	identityPrefix := "https://github.com/" + b.GitHubOwner + "/" + b.GitHubRepo + "/.github/workflows/release.yml@refs/tags/"
	expectedIdentity := identityPrefix + release.TagName

	// Run cosign verify-blob.
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, cosignPath, "verify-blob",
		"--bundle", bundlePath,
		"--certificate-identity", expectedIdentity,
		"--certificate-oidc-issuer", expectedOIDCIssuer,
		checksumsPath,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("sigstore verification failed for %s: %s", release.TagName, detail)
	}

	return &VerificationResult{
		Verified: true,
		Message:  "sigstore signature verified: checksums.txt is authentically signed by the release workflow",
	}, nil
}

// logVerificationResult writes the verification outcome to stderr for user visibility.
func logVerificationResult(result *VerificationResult) {
	if result.Skipped {
		fmt.Fprintf(os.Stderr, "  [info] %s\n", result.Message)
	} else if result.Verified {
		fmt.Fprintf(os.Stderr, "  [verified] %s\n", result.Message)
	}
}
