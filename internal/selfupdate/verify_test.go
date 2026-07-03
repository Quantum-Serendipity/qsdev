package selfupdate

import (
	"context"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestVerifySigstoreBundle_NoBundleAsset(t *testing.T) {
	release := &Release{
		Version: "1.0.0",
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
			{Name: "qsdev_1.0.0_Linux_x86_64.tar.gz", URL: "https://example.com/archive.tar.gz"},
		},
	}

	result, err := verifySigstoreBundleImpl(context.Background(), release, "/tmp/fake-checksums.txt", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Skipped {
		t.Error("expected Skipped=true when bundle asset is not in release")
	}
	if result.Verified {
		t.Error("expected Verified=false")
	}
}

func TestCosignVerifyArgs_ExactIdentityPin(t *testing.T) {
	args := cosignVerifyArgs("v1.2.3", "/tmp/bundle.json", "/tmp/checksums.txt")

	// Must pin the EXACT identity, never the permissive regexp variant.
	if slices.Contains(args, "--certificate-identity-regexp") {
		t.Fatalf("args must not use --certificate-identity-regexp (permissive); got %v", args)
	}

	idIdx := slices.Index(args, "--certificate-identity")
	if idIdx < 0 || idIdx+1 >= len(args) {
		t.Fatalf("args missing --certificate-identity <value>; got %v", args)
	}

	_, wantIdentity := branding.ReleaseWorkflowIdentity("v1.2.3")
	if got := args[idIdx+1]; got != wantIdentity {
		t.Errorf("certificate-identity = %q, want %q", got, wantIdentity)
	}

	issuerIdx := slices.Index(args, "--certificate-oidc-issuer")
	if issuerIdx < 0 || issuerIdx+1 >= len(args) {
		t.Fatalf("args missing --certificate-oidc-issuer <value>; got %v", args)
	}
	if got, want := args[issuerIdx+1], "https://token.actions.githubusercontent.com"; got != want {
		t.Errorf("certificate-oidc-issuer = %q, want %q", got, want)
	}

	// The checksums path must be the final positional argument to verify-blob.
	if got := args[len(args)-1]; got != "/tmp/checksums.txt" {
		t.Errorf("last arg = %q, want the checksums path %q", got, "/tmp/checksums.txt")
	}
}

func TestVerifySigstoreBundle_MockedVerifier(t *testing.T) {
	oldFn := verifySigstoreBundle
	t.Cleanup(func() { verifySigstoreBundle = oldFn })

	verifySigstoreBundle = func(ctx context.Context, release *Release, checksumsPath, tmpDir string) (*VerificationResult, error) {
		return &VerificationResult{Verified: true, Message: "mock verified"}, nil
	}

	release := &Release{
		Version: "1.0.0",
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: sigstoreBundleName, URL: "https://example.com/bundle.json"},
		},
	}

	result, err := verifySigstoreBundle(context.Background(), release, "/tmp/checksums.txt", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verified {
		t.Error("expected Verified=true from mock")
	}
}
