package selfupdate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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

// fakeCosignRelease serves a sigstore bundle asset from an httptest server and
// puts a fake `cosign` (exiting with exitCode and recording its argv) first on
// PATH. It returns the release and the argv record file.
func fakeCosignRelease(t *testing.T, exitCode int) (*Release, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bundle":"fake"}`))
	}))
	t.Cleanup(srv.Close)

	binDir := t.TempDir()
	argvFile := filepath.Join(t.TempDir(), "argv")
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\necho 'fake cosign failure' >&2\nexit %d\n", argvFile, exitCode)
	if err := os.WriteFile(filepath.Join(binDir, "cosign"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	return &Release{
		Version: "1.0.0",
		TagName: "v1.0.0",
		Assets:  []Asset{{Name: sigstoreBundleName, URL: srv.URL + "/bundle"}},
	}, argvFile
}

func TestVerifySigstoreBundleImpl_Cosign(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake cosign is a shell script")
	}

	tests := []struct {
		name     string
		exitCode int
		wantErr  bool
	}{
		{name: "cosign rejects the bundle: fail closed", exitCode: 1, wantErr: true},
		{name: "cosign accepts the bundle: verified", exitCode: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, argvFile := fakeCosignRelease(t, tt.exitCode)
			checksums := filepath.Join(t.TempDir(), "checksums.txt")
			tmpDir := t.TempDir()

			result, err := verifySigstoreBundleImpl(context.Background(), release, checksums, tmpDir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error when cosign verification fails, got %+v", result)
				}
				if !strings.Contains(err.Error(), "fake cosign failure") {
					t.Errorf("error %q should carry cosign's stderr", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !result.Verified || result.Skipped {
					t.Errorf("result = %+v, want Verified", result)
				}
			}

			argv, err := os.ReadFile(argvFile)
			if err != nil {
				t.Fatalf("fake cosign was not invoked: %v", err)
			}
			got := strings.Split(strings.TrimSpace(string(argv)), "\n")
			// The bundle is downloaded into tmpDir under its asset name.
			want := cosignVerifyArgs(release.TagName, tmpDir+"/"+sigstoreBundleName, checksums)
			if !slices.Equal(got, want) {
				t.Errorf("cosign argv = %q, want %q", got, want)
			}
		})
	}
}

func TestVerifySigstoreBundleImpl_NoCosign(t *testing.T) {
	release := &Release{
		Version: "1.0.0",
		TagName: "v1.0.0",
		Assets:  []Asset{{Name: sigstoreBundleName, URL: "https://example.invalid/bundle"}},
	}
	t.Setenv("PATH", t.TempDir()) // no cosign anywhere

	result, err := verifySigstoreBundleImpl(context.Background(), release, "/tmp/checksums.txt", t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Skipped || result.Verified {
		t.Errorf("result = %+v, want Skipped (cosign absent)", result)
	}
}
