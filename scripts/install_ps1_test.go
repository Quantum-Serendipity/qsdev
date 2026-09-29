package scripts_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run scripts/install.ps1 under pwsh with Invoke-WebRequest and
// Invoke-RestMethod replaced by functions that serve a fake release from disk
// (functions take precedence over cmdlets), so nothing is downloaded. They
// skip when pwsh is not installed.

// psHarness defines the stubs; FAKE_RELEASE is the release directory.
const psHarness = `
function Invoke-WebRequest {
    param([string]$Uri, [string]$OutFile, [switch]$UseBasicParsing)
    $rel = $Uri -replace '^` + releaseURL + `', ''
    $src = Join-Path $env:FAKE_RELEASE $rel
    if ($rel -eq $Uri -or -not (Test-Path -LiteralPath $src)) { throw "Response status code does not indicate success: 404 (Not Found)." }
    Copy-Item -LiteralPath $src -Destination $OutFile
}
function Invoke-RestMethod {
    param([string]$Uri)
    $src = Join-Path $env:FAKE_RELEASE "latest.json"
    if ($Uri -ne "` + latestURL + `" -or -not (Test-Path -LiteralPath $src)) { throw "404 (Not Found)." }
    Get-Content -Raw -LiteralPath $src | ConvertFrom-Json
}
`

// fakeCosignPS accepts a bundle whose content is "valid".
const fakeCosignPS = `
if ($args -contains "--help") { "  --bundle string"; exit 0 }
$i = [array]::IndexOf($args, "--bundle")
if ((Get-Content -Raw -LiteralPath $args[$i + 1]).Trim() -eq "valid") { "Verified OK"; exit 0 }
"invalid signature"; exit 1
`

const winArchive = "qsdev_1.2.3_Windows_x86_64.zip"

type psEnv struct {
	t          *testing.T
	root       string
	releaseDir string
	cosignDir  string
	installDir string
}

func newPSEnv(t *testing.T, cosign bool, bundle string, sbomFirst bool) *psEnv {
	t.Helper()
	if _, err := exec.LookPath("pwsh"); err != nil {
		t.Skip("pwsh not installed")
	}
	if _, err := exec.LookPath("cosign"); err == nil && !cosign {
		t.Skip("a real cosign is on PATH")
	}
	root := t.TempDir()
	e := &psEnv{
		t: t, root: root,
		releaseDir: filepath.Join(root, "release"),
		cosignDir:  filepath.Join(root, "cosignbin"),
		installDir: filepath.Join(root, "inst"),
	}
	dir := filepath.Join(e.releaseDir, "v1.2.3")
	for _, d := range []string{dir, e.cosignDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if cosign {
		e.write(filepath.Join(e.cosignDir, "cosign.ps1"), fakeCosignPS)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("qsdev.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("genuine")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	zipLine := hex.EncodeToString(sum[:]) + "  " + winArchive + "\n"
	// goreleaser lists the archive's SBOMs, whose names extend the archive
	// name, in the same file.
	sbomLine := strings.Repeat("0", 64) + "  " + winArchive + ".cdx.json\n"
	checksums := zipLine + sbomLine
	if sbomFirst {
		checksums = sbomLine + zipLine
	}
	e.write(filepath.Join(dir, winArchive), buf.String())
	e.write(filepath.Join(dir, "checksums.txt"), checksums)
	if bundle != "" {
		e.write(filepath.Join(dir, "checksums.txt.sigstore.json"), bundle)
	}
	e.write(filepath.Join(e.releaseDir, "latest.json"), `{"tag_name": "v1.2.3"}`)
	e.write(filepath.Join(root, "harness.ps1"), psHarness)
	return e
}

func (e *psEnv) write(path, content string) {
	e.t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		e.t.Fatal(err)
	}
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func (e *psEnv) run(extraEnv []string, args ...string) (int, string) {
	e.t.Helper()
	script, err := filepath.Abs("install.ps1")
	if err != nil {
		e.t.Fatal(err)
	}
	command := fmt.Sprintf(". %s; & %s -InstallDir %s -NoModifyPath %s",
		psQuote(filepath.Join(e.root, "harness.ps1")), psQuote(script), psQuote(e.installDir), strings.Join(args, " "))
	cmd := exec.Command("pwsh", "-NoProfile", "-NonInteractive", "-Command", command)
	cmd.Env = append(os.Environ(),
		"PATH="+e.cosignDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_RELEASE="+e.releaseDir,
		"TMPDIR="+e.root,
		"QSDEV_INSTALL_VERSION=", "QSDEV_VERSION=", "QSDEV_ALLOW_UNSIGNED=", "QSDEV_REQUIRE_SIGNATURE=",
	)
	cmd.Env = append(cmd.Env, extraEnv...)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, string(out)
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), string(out)
	default:
		e.t.Fatalf("running pwsh: %v", err)
		return -1, ""
	}
}

// installed reports whether qsdev.exe was written. The script builds paths
// with "\", which on a non-Windows test host names a file beside InstallDir.
func (e *psEnv) installed() bool {
	for _, p := range []string{filepath.Join(e.installDir, "qsdev.exe"), e.installDir + `\qsdev.exe`} {
		if b, err := os.ReadFile(p); err == nil {
			return string(b) == "genuine"
		}
	}
	return false
}

// TestInstallPS1 covers W188 (exact checksum match, Sigstore fail-closed,
// arm64 fallback) and W179 (QSDEV_VERSION is not a pin) for install.ps1.
func TestInstallPS1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cosign      bool
		bundle      string
		sbomFirst   bool
		env         []string
		args        []string
		wantCode    int
		wantInstall bool
		wantOut     string
	}{
		{"checksum matched exactly despite SBOM lines", false, "", true, nil, []string{"-Version", "1.2.3"}, 0, true, "Checksum verified"},
		{"no cosign", false, "", false, nil, []string{"-Version", "v1.2.3"}, 0, true, "cosign not found"},
		{"bundle missing fails closed", true, "", false, nil, []string{"-Version", "1.2.3"}, 1, false, "Sigstore bundle unavailable"},
		{"bundle missing allowed explicitly", true, "", false, []string{"QSDEV_ALLOW_UNSIGNED=1"}, []string{"-Version", "1.2.3"}, 0, true, "WITHOUT SIGNATURE VERIFICATION"},
		{"valid signature", true, "valid", false, nil, []string{"-Version", "1.2.3"}, 0, true, "Sigstore signature verified"},
		{"invalid signature", true, "forged", false, nil, []string{"-Version", "1.2.3"}, 1, false, "Sigstore verification FAILED"},
		{"signature required without cosign", false, "valid", false, nil, []string{"-Version", "1.2.3", "-RequireSignature"}, 1, false, "signature is required"},
		{"allow-unsigned and require-signature contradict", true, "", false, nil, []string{"-Version", "1.2.3", "-AllowUnsigned", "-RequireSignature"}, 1, false, "contradict"},
		{"arm64 falls back to x86_64", false, "", false, nil, []string{"-Version", "1.2.3", "-ForceArch", "arm64", "-DryRun"}, 0, false, winArchive},
		{"devenv QSDEV_VERSION ignored", false, "", false, []string{"QSDEV_VERSION=v0.7.4-7-g952bd4f"}, nil, 0, true, "Ignoring QSDEV_VERSION"},
		{"invalid pin rejected", false, "", false, []string{"QSDEV_INSTALL_VERSION=0.7.4-7+g952bd4f"}, nil, 1, false, "Invalid version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newPSEnv(t, tt.cosign, tt.bundle, tt.sbomFirst)
			code, out := e.run(tt.env, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d\n%s", code, tt.wantCode, out)
			}
			if got := e.installed(); got != tt.wantInstall {
				t.Errorf("installed = %v, want %v\n%s", got, tt.wantInstall, out)
			}
			if !strings.Contains(out, tt.wantOut) {
				t.Errorf("output lacks %q:\n%s", tt.wantOut, out)
			}
		})
	}
}
