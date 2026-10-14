package cigeneration

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// nixSystems are the systems the catalog pins a Nix release tarball for, one
// per GitHub-hosted runner platform install-nix.sh supports.
var nixSystems = []string{"x86_64-linux", "aarch64-linux", "aarch64-darwin", "x86_64-darwin"}

// nixDigestKey is the pin key holding the tarball digest for system.
func nixDigestKey(system string) string {
	return "NIX_SHA256_" + strings.ToUpper(strings.ReplaceAll(system, "-", "_"))
}

// TestNixReleasePinsShape checks the catalog's Nix pins through the shared
// pin-file reader: exactly one exact X.Y.Z NIX_VERSION and one hex SHA-256
// per supported system, each under its matching "# source:" line, and every
// one of them read by the embedded installer.
func TestNixReleasePinsShape(t *testing.T) {
	t.Parallel()

	pins := parsePinFile(t, "embedded nix-release.env", NixReleasePins())
	want := []string{"NIX_VERSION"}
	for _, sys := range nixSystems {
		want = append(want, nixDigestKey(sys))
	}
	var got []string
	for key := range pins {
		got = append(got, key)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("embedded nix-release.env defines %v, want exactly %v", got, want)
	}
	script := string(InstallNixScript())
	for _, key := range want {
		if !strings.Contains(script, key) {
			t.Errorf("install-nix.sh never reads %s; remove the pin or use it", key)
		}
	}
}

// TestNixInstallerAccessorsReturnCopies keeps the embedded bytes immutable:
// a caller that edits what it was given must not change what the next
// caller gets.
func TestNixInstallerAccessorsReturnCopies(t *testing.T) {
	t.Parallel()

	for name, get := range map[string]func() []byte{
		"InstallNixScript": InstallNixScript,
		"NixReleasePins":   NixReleasePins,
	} {
		b := get()
		if len(b) == 0 {
			t.Fatalf("%s() is empty", name)
		}
		b[0] ^= 0xff
		if again := get(); again[0] == b[0] {
			t.Errorf("%s() returned the embedded bytes, not a copy", name)
		}
	}
}

// TestRepoNixInstallerIsCatalogOutput keeps this repository's own CI on the
// generator's output: .github/scripts/install-nix.sh and .github/nix-release.env
// are byte-identical to the catalog's embeds, and the script is executable.
func TestRepoNixInstallerIsCatalogOutput(t *testing.T) {
	t.Parallel()

	for rel, want := range map[string][]byte{
		InstallNixScriptPath: InstallNixScript(),
		NixReleasePinsPath:   NixReleasePins(),
	} {
		got, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from the catalog's embedded copy; regenerate it (qsdev update) or edit internal/cigeneration instead", rel)
		}
	}

	if mode, ok := gitIndexMode(t, InstallNixScriptPath); ok {
		if mode != "100755" {
			t.Errorf("%s has mode %s in the git index, want 100755", InstallNixScriptPath, mode)
		}
		return
	}
	if runtime.GOOS == "windows" {
		return // no git and no executable bit to check
	}
	fi, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(InstallNixScriptPath)))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("%s is not executable (mode %v)", InstallNixScriptPath, fi.Mode().Perm())
	}
}

// gitIndexMode returns rel's mode in the git index, or false when there is
// no git or no work tree with rel in it (an exported source tree).
func gitIndexMode(t *testing.T, rel string) (string, bool) {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", false
	}
	cmd := exec.Command(gitPath, "ls-files", "-s", "--", rel)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	fields := strings.Fields(string(out))
	if err != nil || len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}

// fakeNixCurl writes $FAKE_TARBALL to curl's -o path and logs the call.
const fakeNixCurl = `#!/bin/sh
out=""
url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift ;;
    https://*) url="$1" ;;
  esac
  shift
done
echo "curl $url" >>"$STUB_LOG"
cat "$FAKE_TARBALL" >"$out"
`

// fakeNixUname reports a Linux x86_64 runner.
const fakeNixUname = `#!/bin/sh
case "$1" in
  -s) echo Linux ;;
  -m) echo x86_64 ;;
  *) echo "unexpected uname $*" >&2; exit 2 ;;
esac
`

// fakeNixTar "extracts" a release: it creates the bundled installer under
// the -C directory. The installer creates the nix the script checks.
const fakeNixTar = `#!/bin/sh
dir=""
while [ $# -gt 0 ]; do
  case "$1" in
    -C) dir="$2"; shift ;;
  esac
  shift
done
echo "tar" >>"$STUB_LOG"
d="$dir/nix-$FAKE_NIX_VERSION-x86_64-linux"
mkdir -p "$d"
cat >"$d/install" <<EOF
#!/bin/sh
echo "install \$*" >>"$STUB_LOG"
mkdir -p "\$HOME/.nix-profile/bin"
printf '#!/bin/sh\necho "nix (Nix) $FAKE_NIX_VERSION"\n' >"\$HOME/.nix-profile/bin/nix"
chmod +x "\$HOME/.nix-profile/bin/nix"
EOF
chmod +x "$d/install"
`

// fakeNixSudo logs and succeeds, so a host AppArmor setting never matters.
const fakeNixSudo = `#!/bin/sh
echo "sudo $*" >>"$STUB_LOG"
`

// installNixHostTools are the real binaries install-nix.sh runs; everything
// that touches the network, the system or the release is stubbed.
var installNixHostTools = []string{"bash", "sh", "sha256sum", "mktemp", "cat", "chmod", "mkdir", "rm"}

// installNixRun is one hermetic install-nix.sh run's outcome.
type installNixRun struct {
	code int
	out  string
	log  []string
}

// runInstallNix runs the embedded install-nix.sh with pins as its pin file,
// against stub curl, uname, tar, sudo and installer that serve tarball as the
// release, with HOME, XDG_CONFIG_HOME, RUNNER_TEMP, GITHUB_PATH and GITHUB_ENV
// in a temporary directory.
func runInstallNix(t *testing.T, pins, version string, tarball []byte) installNixRun {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("install-nix.sh runs on Linux and macOS runners; it needs bash")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	home := filepath.Join(root, "home")
	for _, d := range []string{binDir, home} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tool := range installNixHostTools {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s is not installed; the CI runner provides it", tool)
		}
		if err := os.Symlink(p, filepath.Join(binDir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{
		"curl": fakeNixCurl, "uname": fakeNixUname, "tar": fakeNixTar, "sudo": fakeNixSudo,
	} {
		writeStub(t, filepath.Join(binDir, name), body)
	}
	script := filepath.Join(root, "install-nix.sh")
	writeStub(t, script, string(InstallNixScript()))
	pinsFile := filepath.Join(root, "nix-release.env")
	tarballFile := filepath.Join(root, "release.tar.xz")
	logFile := filepath.Join(root, "stub.log")
	for path, b := range map[string][]byte{pinsFile: []byte(pins), tarballFile: tarball} {
		if err := os.WriteFile(path, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := exec.Command(filepath.Join(binDir, "bash"), script, pinsFile)
	cmd.Env = []string{
		"PATH=" + binDir,
		"HOME=" + home,
		"XDG_CONFIG_HOME=" + filepath.Join(root, "xdg"),
		"RUNNER_TEMP=" + root,
		"TMPDIR=" + root,
		"GITHUB_PATH=" + filepath.Join(root, "github_path"),
		"GITHUB_ENV=" + filepath.Join(root, "github_env"),
		"STUB_LOG=" + logFile,
		"FAKE_TARBALL=" + tarballFile,
		"FAKE_NIX_VERSION=" + version,
	}
	out, err := cmd.CombinedOutput()
	res := installNixRun{out: string(out)}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.code = exitErr.ExitCode()
	default:
		t.Fatalf("running install-nix.sh: %v", err)
	}
	if b, err := os.ReadFile(logFile); err == nil {
		res.log = strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return res
}

func writeStub(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil { //nolint:gosec // test stub must be executable
		t.Fatal(err)
	}
}

// nixPinsFile renders a pin file in nix-release.env's shape with digest for
// x86_64-linux and a fixed valid digest for the other systems; extra is
// appended verbatim.
func nixPinsFile(version, digest, extra string) string {
	var b strings.Builder
	b.WriteString("# source: nix-release NixOS/nix\nNIX_VERSION=" + version + "\n")
	for _, sys := range nixSystems {
		d := strings.Repeat("0", 64)
		if sys == "x86_64-linux" {
			d = digest
		}
		b.WriteString("\n# source: nix-sha256 " + sys + "\n" + nixDigestKey(sys) + "=" + d + "\n")
	}
	b.WriteString(extra)
	return b.String()
}

// stubRan reports whether a stub log line starts with name.
func stubRan(log []string, name string) bool {
	return slices.ContainsFunc(log, func(l string) bool { return l == name || strings.HasPrefix(l, name+" ") })
}

// TestInstallNixScript_VerifiesDigestBeforeRunning is the installer's
// supply-chain contract: nothing from the downloaded tarball is extracted or
// run unless its SHA-256 matches the pin, and malformed pins fail before any
// download.
func TestInstallNixScript_VerifiesDigestBeforeRunning(t *testing.T) {
	t.Parallel()

	const version = "2.34.8"
	tarball := []byte("pretend nix release tarball\n")
	sum := sha256.Sum256(tarball)
	good := hex.EncodeToString(sum[:])
	other := sha256.Sum256([]byte("a different tarball\n"))

	tests := []struct {
		name        string
		pins        string
		wantOK      bool
		wantCurl    bool
		wantInOut   string
		wantInstall bool
	}{
		{
			name:      "mismatched digest",
			pins:      nixPinsFile(version, hex.EncodeToString(other[:]), ""),
			wantCurl:  true,
			wantInOut: "FAILED",
		},
		{
			name:        "matching digest",
			pins:        nixPinsFile(version, good, ""),
			wantOK:      true,
			wantCurl:    true,
			wantInOut:   "Installed nix (Nix) " + version + " for x86_64-linux",
			wantInstall: true,
		},
		{
			name:      "non-hex digest",
			pins:      nixPinsFile(version, strings.Repeat("g", 64), ""),
			wantInOut: "is not a SHA-256 hex digest",
		},
		{
			name:      "v-prefixed version",
			pins:      nixPinsFile("v"+version, good, ""),
			wantInOut: "is not an exact release",
		},
		{
			name:      "key defined twice",
			pins:      nixPinsFile(version, good, "\n# source: nix-release NixOS/nix\nNIX_VERSION="+version+"\n"),
			wantInOut: "NIX_VERSION is defined 2 times",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := runInstallNix(t, tt.pins, version, tarball)
			if ok := res.code == 0; ok != tt.wantOK {
				t.Errorf("exit code %d, want success=%v\noutput:\n%s", res.code, tt.wantOK, res.out)
			}
			if !strings.Contains(res.out, tt.wantInOut) {
				t.Errorf("output does not contain %q:\n%s", tt.wantInOut, res.out)
			}
			if got := stubRan(res.log, "curl"); got != tt.wantCurl {
				t.Errorf("curl ran = %v, want %v (log %q)", got, tt.wantCurl, res.log)
			}
			for _, step := range []string{"tar", "install"} {
				if got := stubRan(res.log, step); got != tt.wantInstall {
					t.Errorf("%s ran = %v, want %v (log %q)", step, got, tt.wantInstall, res.log)
				}
			}
			if tt.wantInstall {
				tarAt := slices.IndexFunc(res.log, func(l string) bool { return l == "tar" })
				installAt := slices.IndexFunc(res.log, func(l string) bool { return strings.HasPrefix(l, "install ") })
				if tarAt > installAt {
					t.Errorf("the installer ran before the tarball was extracted (log %q)", res.log)
				}
			}
		})
	}
}
