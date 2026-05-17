package selfupdate

import (
	"context"
	"testing"
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
