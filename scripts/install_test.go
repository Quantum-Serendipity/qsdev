package scripts_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// These tests run scripts/install.sh against a fake GitHub release served by a
// stub curl, on a PATH restricted to stub and allow-listed system tools, so no
// real download or install can happen.

const (
	releaseURL = "https://github.com/Quantum-Serendipity/qsdev/releases/download/"
	latestURL  = "https://api.github.com/repos/Quantum-Serendipity/qsdev/releases/latest"
	// sleeperEnv makes this test binary, installed as the "qsdev" under
	// test, keep running so the installer must replace a busy executable.
	sleeperEnv = "QSDEV_INSTALL_TEST_SLEEPER"
)

// fakeCurl serves release URLs from $FAKE_RELEASE and fails like `curl -f`
// on a 404 (exit 22) when the file is absent.
const fakeCurl = `#!/bin/sh
out=""; url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift ;;
    http*) url="$1" ;;
  esac
  shift
done
echo "$url" >> "$FAKE_RELEASE/../curl.log"
case "$url" in
  ` + releaseURL + `*) f="$FAKE_RELEASE/${url#` + releaseURL + `}" ;;
  ` + latestURL + `) f="$FAKE_RELEASE/latest.json" ;;
  *) exit 6 ;;
esac
[ -f "$f" ] || exit 22
if [ -n "$out" ]; then cat "$f" > "$out"; else cat "$f"; fi
`

// fakeCosign accepts a bundle whose content is "valid".
const fakeCosign = `#!/bin/sh
if [ "$2" = "--help" ]; then echo "  --bundle string"; exit 0; fi
echo "$*" >> "$FAKE_RELEASE/../cosign.log"
while [ $# -gt 0 ]; do
  if [ "$1" = "--bundle" ]; then b="$2"; fi
  shift
done
[ "$(cat "$b")" = "valid" ] && echo "Verified OK" && exit 0
echo "invalid signature" >&2; exit 1
`

// systemTools are the real binaries the installer may use.
var systemTools = []string{
	"sh", "tar", "gzip", "sha256sum", "shasum", "awk", "sed", "grep", "mktemp",
	"cp", "mv", "chmod", "mkdir", "rm", "uname", "head", "dirname", "basename",
	"readlink", "cat",
}

type release struct {
	version string
	binary  []byte // content of the qsdev binary in the archive
	bundle  string // "" = no bundle published
	latest  bool   // publish releases/latest pointing at version
}

type env struct {
	t          *testing.T
	root, home string
	releaseDir string
	pathDir    string
	installDir string
}

func archiveName(version string) string {
	osTitle := map[string]string{"linux": "Linux", "darwin": "Darwin"}[runtime.GOOS]
	arch := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[runtime.GOARCH]
	return fmt.Sprintf("qsdev_%s_%s_%s.tar.gz", version, osTitle, arch)
}

func newEnv(t *testing.T, withCosign bool, releases ...release) *env {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("install.sh supports Linux and macOS only")
	}
	root := t.TempDir()
	e := &env{
		t:          t,
		root:       root,
		home:       filepath.Join(root, "home"),
		releaseDir: filepath.Join(root, "release"),
		pathDir:    filepath.Join(root, "bin"),
	}
	e.installDir = filepath.Join(e.home, ".qsdev", "bin")
	for _, d := range []string{e.home, e.releaseDir, e.pathDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tool := range systemTools {
		if p, err := exec.LookPath(tool); err == nil {
			if err := os.Symlink(p, filepath.Join(e.pathDir, tool)); err != nil {
				t.Fatal(err)
			}
		}
	}
	e.writeExec(filepath.Join(e.pathDir, "curl"), fakeCurl)
	if withCosign {
		e.writeExec(filepath.Join(e.pathDir, "cosign"), fakeCosign)
	}
	for _, r := range releases {
		e.publish(r)
	}
	return e
}

func (e *env) writeExec(path, content string) {
	e.t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil { //nolint:gosec // test stub must be executable
		e.t.Fatal(err)
	}
}

func (e *env) publish(r release) {
	e.t.Helper()
	dir := filepath.Join(e.releaseDir, "v"+r.version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		e.t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "qsdev", Mode: 0o755, Size: int64(len(r.binary))}); err != nil {
		e.t.Fatal(err)
	}
	if _, err := tw.Write(r.binary); err != nil {
		e.t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		e.t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		e.t.Fatal(err)
	}
	name := archiveName(r.version)
	sum := sha256.Sum256(buf.Bytes())
	files := map[string]string{
		name:            buf.String(),
		"checksums.txt": hex.EncodeToString(sum[:]) + "  " + name + "\n",
	}
	if r.bundle != "" {
		files["checksums.txt.sigstore.json"] = r.bundle
	}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			e.t.Fatal(err)
		}
	}
	if r.latest {
		latest := fmt.Sprintf(`{"tag_name": "v%s"}`, r.version)
		if err := os.WriteFile(filepath.Join(e.releaseDir, "latest.json"), []byte(latest), 0o644); err != nil {
			e.t.Fatal(err)
		}
	}
}

// run executes install.sh with args and extra environment and returns its
// exit code and combined output.
func (e *env) run(extraEnv []string, args ...string) (int, string) {
	e.t.Helper()
	cmd := exec.Command(filepath.Join(e.pathDir, "sh"), append([]string{"install.sh", "--no-modify-path"}, args...)...)
	cmd.Env = append([]string{
		"PATH=" + e.pathDir,
		"HOME=" + e.home,
		"SHELL=/bin/sh",
		"NO_COLOR=1",
		"FAKE_RELEASE=" + e.releaseDir,
		"TMPDIR=" + e.root,
	}, extraEnv...)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, string(out)
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), string(out)
	default:
		e.t.Fatalf("running install.sh: %v", err)
		return -1, ""
	}
}

func (e *env) installed() []byte {
	e.t.Helper()
	b, err := os.ReadFile(filepath.Join(e.installDir, "qsdev"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		e.t.Fatal(err)
	}
	return b
}

var genuine = []byte("#!/bin/sh\necho genuine\n")

// TestInstall_Sigstore covers W177: once cosign is available, a missing
// Sigstore bundle must stop the install (an attacker who can replace the
// archive and checksums.txt can also delete the bundle).
func TestInstall_Sigstore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cosign      bool
		bundle      string
		env         []string
		wantCode    int
		wantInstall bool
		wantOut     string
	}{
		{"bundle missing fails closed", true, "", nil, 1, false, "Sigstore bundle unavailable"},
		{"bundle missing allowed explicitly", true, "", []string{"QSDEV_ALLOW_UNSIGNED=1"}, 0, true, "WITHOUT SIGNATURE VERIFICATION"},
		{"valid signature", true, "valid", nil, 0, true, "Sigstore signature verified"},
		{"invalid signature", true, "forged", nil, 1, false, "Sigstore verification FAILED"},
		{"no cosign", false, "", nil, 0, true, "cosign not found"},
		{"no cosign but signature required", false, "valid", []string{"QSDEV_REQUIRE_SIGNATURE=1"}, 1, false, "signature is required"},
		{"allow-unsigned and require-signature contradict", true, "", []string{"QSDEV_ALLOW_UNSIGNED=1", "QSDEV_REQUIRE_SIGNATURE=1"}, 1, false, "contradict"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t, tt.cosign, release{version: "1.2.3", binary: genuine, bundle: tt.bundle})
			code, out := e.run(append([]string{"QSDEV_INSTALL_VERSION=1.2.3"}, tt.env...))
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d\n%s", code, tt.wantCode, out)
			}
			if got := e.installed() != nil; got != tt.wantInstall {
				t.Errorf("installed = %v, want %v\n%s", got, tt.wantInstall, out)
			}
			if !strings.Contains(out, tt.wantOut) {
				t.Errorf("output lacks %q:\n%s", tt.wantOut, out)
			}
		})
	}
}

// TestInstall_VersionPin covers W179: QSDEV_VERSION, which every generated
// devenv exports, must not pin the installer; QSDEV_INSTALL_VERSION does and
// is validated.
func TestInstall_VersionPin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		env      []string
		args     []string
		wantCode int
		wantOut  string
	}{
		{"devenv QSDEV_VERSION ignored", []string{"QSDEV_VERSION=v0.7.4-7-g952bd4f"}, nil, 0, "Ignoring QSDEV_VERSION"},
		{"devenv release QSDEV_VERSION ignored", []string{"QSDEV_VERSION=1.0.0"}, nil, 0, "v2.0.0 (latest release)"},
		{"install pin with v prefix", []string{"QSDEV_INSTALL_VERSION=v1.0.0", "QSDEV_VERSION=2.0.0"}, nil, 0, "v1.0.0 (QSDEV_INSTALL_VERSION)"},
		{"--version flag", nil, []string{"--version", "1.0.0"}, 0, "v1.0.0 (--version)"},
		{"invalid pin rejected", []string{"QSDEV_INSTALL_VERSION=0.7.4-7+g952bd4f"}, nil, 1, "Invalid version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t, false,
				release{version: "1.0.0", binary: []byte("#!/bin/sh\necho one\n")},
				release{version: "2.0.0", binary: []byte("#!/bin/sh\necho two\n"), latest: true})
			code, out := e.run(tt.env, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d\n%s", code, tt.wantCode, out)
			}
			if !strings.Contains(out, tt.wantOut) {
				t.Errorf("output lacks %q:\n%s", tt.wantOut, out)
			}
		})
	}
}

// TestInstallSleeper is the stand-in for a running qsdev (MCP server, hook).
// It is a no-op unless sleeperEnv is set.
func TestInstallSleeper(t *testing.T) {
	if os.Getenv(sleeperEnv) == "" {
		return
	}
	time.Sleep(time.Minute)
}

// startSleeper runs the executable at path as a TestInstallSleeper. A write
// descriptor for path can briefly leak into a child that another parallel
// test forks, which makes exec fail with ETXTBSY, so that is retried.
func startSleeper(t *testing.T, path string) *exec.Cmd {
	t.Helper()
	for attempt := 0; ; attempt++ {
		cmd := exec.Command(path, "-test.run=^TestInstallSleeper$")
		cmd.Env = append(os.Environ(), sleeperEnv+"=1")
		err := cmd.Start()
		if err == nil {
			return cmd
		}
		if !errors.Is(err, syscall.ETXTBSY) || attempt == 50 {
			t.Fatal(err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestInstall_ReplacesRunningBinary covers W180: upgrading while qsdev runs
// must not fail with "Text file busy", and must swap the file atomically.
func TestInstall_ReplacesRunningBinary(t *testing.T) {
	t.Parallel()
	e := newEnv(t, false, release{version: "1.2.3", binary: genuine})
	if err := os.MkdirAll(e.installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	self, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(e.installDir, "qsdev")
	if err := os.WriteFile(target, self, 0o755); err != nil { //nolint:gosec // must be executable
		t.Fatal(err)
	}
	running := startSleeper(t, target)
	t.Cleanup(func() { _ = running.Process.Kill(); _ = running.Wait() })

	code, out := e.run([]string{"QSDEV_INSTALL_VERSION=1.2.3"})
	if code != 0 {
		t.Fatalf("exit %d, want 0\n%s", code, out)
	}
	if got := e.installed(); !bytes.Equal(got, genuine) {
		t.Errorf("installed binary was not replaced\n%s", out)
	}
	leftovers, _ := filepath.Glob(filepath.Join(e.installDir, ".qsdev.new.*"))
	if len(leftovers) > 0 {
		t.Errorf("staging files left behind: %v", leftovers)
	}
}

// TestInstall_VerifyOnly covers W178: --verify-only compares the installed
// binary with the binary inside the verified release archive, exits non-zero
// on a mismatch, never executes the installed binary, and does not need the
// GitHub API.
func TestInstall_VerifyOnly(t *testing.T) {
	t.Parallel()
	// Executing this binary leaves a marker, which verification must not do.
	trojan := func(marker string) []byte {
		return []byte("#!/bin/sh\ntouch " + marker + "\necho 'qsdev version 1.2.3'\n")
	}
	tests := []struct {
		name       string
		tamper     bool
		receipt    bool
		unsigned   bool // the release's Sigstore bundle is removed
		args       []string
		wantCode   int
		wantOutput string
	}{
		{"genuine binary", false, true, false, nil, 0, "matches the signed qsdev v1.2.3"},
		{"tampered binary", true, true, false, nil, 1, "does NOT match"},
		{"no receipt and no version", false, false, false, nil, 1, "--version"},
		{"no receipt with --version", false, false, false, []string{"--version", "1.2.3"}, 0, "matches"},
		{"tampered, version from flag", true, false, false, []string{"--version", "1.2.3"}, 1, "does NOT match"},
		{"unsigned release fails closed", false, true, true, nil, 1, "Sigstore bundle unavailable"},
		{"--no-verify rejected", false, true, false, []string{"--no-verify"}, 1, "cannot be combined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newEnv(t, true, release{version: "1.2.3", binary: genuine, bundle: "valid"})
			if code, out := e.run([]string{"QSDEV_INSTALL_VERSION=1.2.3"}); code != 0 {
				t.Fatalf("install: exit %d\n%s", code, out)
			}
			marker := filepath.Join(e.root, "executed")
			if tt.tamper {
				if err := os.WriteFile(filepath.Join(e.installDir, "qsdev"), trojan(marker), 0o755); err != nil { //nolint:gosec // must be executable
					t.Fatal(err)
				}
			}
			if !tt.receipt {
				if err := os.Remove(filepath.Join(e.installDir, ".qsdev-version")); err != nil {
					t.Fatal(err)
				}
			}
			if tt.unsigned {
				if err := os.Remove(filepath.Join(e.releaseDir, "v1.2.3", "checksums.txt.sigstore.json")); err != nil {
					t.Fatal(err)
				}
			}
			// The GitHub API is unreachable: verification must not need it.
			_ = os.Remove(filepath.Join(e.releaseDir, "latest.json"))

			code, out := e.run(nil, append([]string{"--verify-only"}, tt.args...)...)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d\n%s", code, tt.wantCode, out)
			}
			if !strings.Contains(out, tt.wantOutput) {
				t.Errorf("output lacks %q:\n%s", tt.wantOutput, out)
			}
			if _, err := os.Stat(marker); err == nil {
				t.Error("--verify-only executed the installed binary")
			}
		})
	}
}

// TestInstall_RequireSignatureWithoutChecksumTool: with no sha256 tool the
// archive cannot be tied to the signed checksums.txt, so --require-signature
// must fail rather than install unverified.
func TestInstall_RequireSignatureWithoutChecksumTool(t *testing.T) {
	t.Parallel()
	e := newEnv(t, true, release{version: "1.2.3", binary: genuine, bundle: "valid"})
	for _, tool := range []string{"sha256sum", "shasum"} {
		if err := os.Remove(filepath.Join(e.pathDir, tool)); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
	}
	code, out := e.run([]string{"QSDEV_INSTALL_VERSION=1.2.3"}, "--require-signature")
	if code != 1 || e.installed() != nil {
		t.Errorf("exit %d, installed %v; want exit 1 and nothing installed\n%s", code, e.installed() != nil, out)
	}
	if !strings.Contains(out, "signature cannot be checked") {
		t.Errorf("output lacks the reason:\n%s", out)
	}
}
