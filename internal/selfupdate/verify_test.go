package selfupdate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// releaseFixtureDir holds checksums.txt and its Sigstore bundle exactly as
// published by the real v0.7.9 release workflow, so verification is exercised
// end to end against the embedded trusted root with no network access.
const (
	releaseFixtureDir = "testdata/release-v0.7.9"
	releaseFixtureTag = "v0.7.9"
)

func fixturePath(name string) string { return filepath.Join(releaseFixtureDir, name) }

// writeTampered copies src to a temp file with extra bytes appended.
func writeTampered(t *testing.T, src string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), filepath.Base(src))
	if err := os.WriteFile(out, append(data, "deadbeef  evil.tar.gz\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEmbeddedTrustedRoot(t *testing.T) {
	t.Parallel()
	tr, err := loadTrustedRoot()
	if err != nil {
		t.Fatalf("embedded trusted root does not parse: %v", err)
	}
	if len(tr.FulcioCertificateAuthorities()) == 0 {
		t.Error("embedded trusted root has no Fulcio certificate authority")
	}
	if len(tr.RekorLogs()) == 0 {
		t.Error("embedded trusted root has no Rekor transparency log")
	}
	if len(tr.CTLogs()) == 0 {
		t.Error("embedded trusted root has no CT log")
	}
}

func TestReleaseCertificateIdentity_ExactPin(t *testing.T) {
	t.Parallel()
	identity, err := releaseCertificateIdentity("v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	wantIssuer, wantSAN := branding.ReleaseWorkflowIdentity("v1.2.3")
	if got := identity.SubjectAlternativeName.SubjectAlternativeName; got != wantSAN {
		t.Errorf("SAN = %q, want %q", got, wantSAN)
	}
	if got := identity.Issuer.Issuer; got != wantIssuer {
		t.Errorf("issuer = %q, want %q", got, wantIssuer)
	}
	// Must pin the EXACT identity, never a permissive regexp.
	if re := identity.SubjectAlternativeName.Regexp.String(); re != "" {
		t.Errorf("SAN regexp = %q, want none (exact match only)", re)
	}
	if re := identity.Issuer.Regexp.String(); re != "" {
		t.Errorf("issuer regexp = %q, want none (exact match only)", re)
	}
}

func TestVerifyChecksumsBundle(t *testing.T) {
	t.Parallel()
	malformed := filepath.Join(t.TempDir(), sigstoreBundleName)
	if err := os.WriteFile(malformed, []byte(`{"bundle":"fake"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		tag       string
		bundle    string
		checksums string
		wantErr   string
	}{
		{name: "genuine release verifies", tag: releaseFixtureTag, bundle: fixturePath(sigstoreBundleName), checksums: fixturePath("checksums.txt")},
		{name: "tampered checksums rejected", tag: releaseFixtureTag, bundle: fixturePath(sigstoreBundleName), checksums: writeTampered(t, fixturePath("checksums.txt")), wantErr: "verif"},
		{name: "signature from another tag rejected", tag: "v0.7.8", bundle: fixturePath(sigstoreBundleName), checksums: fixturePath("checksums.txt"), wantErr: "SAN"},
		{name: "empty tag rejected", tag: "", bundle: fixturePath(sigstoreBundleName), checksums: fixturePath("checksums.txt"), wantErr: "no tag"},
		{name: "malformed bundle rejected", tag: releaseFixtureTag, bundle: malformed, checksums: fixturePath("checksums.txt"), wantErr: "parsing bundle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := verifyChecksumsBundle(tt.tag, tt.bundle, tt.checksums)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected verification to fail closed, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q should mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestVerifySigstoreBundle_NoBundleAsset(t *testing.T) {
	t.Parallel()
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
	if !result.Skipped || result.Verified {
		t.Errorf("result = %+v, want Skipped when the release has no bundle", result)
	}
	if !strings.Contains(result.Message, sigstoreBundleName) {
		t.Errorf("message %q should name the missing %s asset", result.Message, sigstoreBundleName)
	}
}

// TestVerifySigstoreBundleImpl_IgnoresPATHCosign proves verification no longer
// trusts whatever `cosign` is first on PATH: a shim that always exits 0 is
// present, yet a tampered checksums file is still rejected and a genuine one
// verifies without the shim ever running.
func TestVerifySigstoreBundleImpl_IgnoresPATHCosign(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake cosign is a shell script")
	}
	bundleData, err := os.ReadFile(fixturePath(sigstoreBundleName))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bundleData)
	}))
	t.Cleanup(srv.Close)

	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "cosign-ran")
	shim := fmt.Sprintf("#!/bin/sh\ntouch %q\nexit 0\n", marker)
	if err := os.WriteFile(filepath.Join(binDir, "cosign"), []byte(shim), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Setenv("GITHUB_TOKEN", "")

	release := &Release{
		Version: "0.7.9",
		TagName: releaseFixtureTag,
		Assets:  []Asset{{Name: sigstoreBundleName, URL: srv.URL + "/bundle"}},
	}

	tests := []struct {
		name      string
		checksums string
		wantErr   bool
	}{
		{name: "tampered checksums fail closed despite a passing PATH cosign", checksums: writeTampered(t, fixturePath("checksums.txt")), wantErr: true},
		{name: "genuine checksums verify in-process", checksums: fixturePath("checksums.txt")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := verifySigstoreBundleImpl(context.Background(), release, tt.checksums, t.TempDir())
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", result)
				}
				if !strings.Contains(err.Error(), "sigstore verification failed for "+releaseFixtureTag) {
					t.Errorf("error %q should name the failed release", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !result.Verified || result.Skipped {
					t.Errorf("result = %+v, want Verified", result)
				}
			}
			if _, err := os.Stat(marker); err == nil {
				t.Error("the cosign found on PATH was executed; verification must be in-process")
			}
		})
	}
}
