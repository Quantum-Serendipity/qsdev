package scripts_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
)

// These tests run scripts/bump-tool-pins.sh against stub gh and curl
// commands that serve canned release metadata, so no real GitHub or Go
// module proxy request is made.

// fakeGH answers `gh api repos/<owner>/<repo>/releases?per_page=100` from
// $FAKE_UPSTREAM/github/<owner>_<repo>.json and
// `gh api repos/<owner>/<repo>/tags?per_page=100` from
// $FAKE_UPSTREAM/github/<owner>_<repo>.tags.json, and fails like a 404
// otherwise. It logs each API request to $FAKE_UPSTREAM/calls.log. `gh pr
// list` reports no open PR and `gh pr create` succeeds; both are logged to
// $FAKE_UPSTREAM/pr.log.
const fakeGH = `#!/bin/sh
if [ "$1" = "pr" ]; then
  echo "gh pr $2" >>"$FAKE_UPSTREAM/pr.log"
  exit 0
fi
[ "$1" = "api" ] || { echo "unexpected gh call: $*" >&2; exit 2; }
echo "gh api $2" >>"$FAKE_UPSTREAM/calls.log"
path="${2#repos/}"
case "$path" in
  */releases\?per_page=100) repo="${path%/releases?per_page=100}" suffix=json ;;
  */tags\?per_page=100) repo="${path%/tags?per_page=100}" suffix=tags.json ;;
  *) echo "unexpected gh api path: $2" >&2; exit 2 ;;
esac
f="$FAKE_UPSTREAM/github/$(echo "$repo" | tr / _).$suffix"
[ -f "$f" ] || { echo "HTTP 404" >&2; exit 1; }
cat "$f"
`

// fakeUpstreamCurl answers https://proxy.golang.org/<module>/@v/list and
// .../@v/<version>.info from $FAKE_UPSTREAM/goproxy/<module with / as _>/,
// and https://releases.nixos.org/nix/<path> from $FAKE_UPSTREAM/nix/<path>
// (a HEAD request, -I, answers with a Last-Modified header read from
// <path>.lm). It fails like ` + "`curl -f`" + ` on a 404 (exit 22) otherwise.
// It logs each request to $FAKE_UPSTREAM/calls.log.
const fakeUpstreamCurl = `#!/bin/sh
url=""
head=false
for a in "$@"; do
  case "$a" in
    https://*) url="$a" ;;
    --*) ;;
    -*I*) head=true ;;
  esac
done
if [ "$head" = true ]; then
  echo "curl -I $url" >>"$FAKE_UPSTREAM/calls.log"
else
  echo "curl $url" >>"$FAKE_UPSTREAM/calls.log"
fi
case "$url" in
  https://proxy.golang.org/*/@v/*)
    rest="${url#https://proxy.golang.org/}"
    mod="${rest%/@v/*}"
    file="${rest##*/@v/}"
    f="$FAKE_UPSTREAM/goproxy/$(echo "$mod" | tr / _)/$file"
    ;;
  https://releases.nixos.org/nix/*)
    f="$FAKE_UPSTREAM/nix/${url#https://releases.nixos.org/nix/}"
    ;;
  *) echo "unexpected curl url: $url" >&2; exit 6 ;;
esac
[ -f "$f" ] || exit 22
if [ "$head" = true ]; then
  printf 'HTTP/2 200\r\nlast-modified: %s\r\n\r\n' "$(cat "$f.lm")"
else
  cat "$f"
fi
`

// fakeGit logs the git commands --open-pr runs to $FAKE_UPSTREAM/pr.log: a
// commit as "git commit -- <paths>" followed by one "message: " line per -m
// argument with its newlines turned into "|", anything else as "git <cmd>".
const fakeGit = `#!/bin/sh
log="$FAKE_UPSTREAM/pr.log"
if [ "$1" != commit ]; then
  echo "git $1" >>"$log"
  exit 0
fi
shift
paths=""
msgs=""
while [ $# -gt 0 ]; do
  case "$1" in
    -m) msgs="$msgs$(printf '%s' "$2" | tr '\n' '|')
"; shift ;;
    --) shift; paths="$*"; break ;;
  esac
  shift
done
echo "git commit -- $paths" >>"$log"
printf '%s' "$msgs" | while IFS= read -r m; do echo "message: $m" >>"$log"; done
`

// bumpTools are the real binaries bump-tool-pins.sh may use.
var bumpTools = []string{"bash", "sh", "jq", "mktemp", "mv", "rm", "cat", "tr", "sha256sum"}

// nixRelease is a Nix tag as upstream lists it. Its tarballs and their
// published digests exist on the fake releases.nixos.org only when published
// is set, and then only for systems (all of nixSystems when nil).
type nixRelease struct {
	version   string
	age       time.Duration
	published bool
	systems   []string
}

// nixSystems are the systems the CI pin catalog's nix-release.env pins a Nix
// tarball for.
var nixSystems = []string{"x86_64-linux", "aarch64-linux", "aarch64-darwin", "x86_64-darwin"}

// nixDigest is the fake published SHA-256 of a Nix release tarball.
func nixDigest(version, system string) string {
	sum := sha256.Sum256([]byte(version + " " + system))
	return hex.EncodeToString(sum[:])
}

const bumpEnvFile = `# Tool versions.

# source: github example/tool
TOOL_VERSION=v1.2.3

# source: goproxy example.com/mod
MOD_VERSION=v0.4.0
`

type upstreamRelease struct {
	version string
	age     time.Duration
	// prerelease marks a GitHub prerelease; time overrides the published
	// time with a raw value.
	prerelease bool
	time       string
}

func (r upstreamRelease) published(now time.Time) string {
	if r.time != "" {
		return r.time
	}
	return now.Add(-r.age).Format(time.RFC3339Nano)
}

// runBump runs bump-tool-pins.sh against envFile with the given upstream
// releases, newest first as the APIs list them, and returns its exit code,
// combined output and the env file content afterwards.
func runBump(t *testing.T, envFile string, gh, proxy map[string][]upstreamRelease, nix []nixRelease) (int, string, string) {
	t.Helper()
	res := runBumpFiles(t, bumpRun{envFiles: []string{envFile}, gh: gh, proxy: proxy, nix: nix})
	return res.code, res.out, res.after[0]
}

// bumpManifest is qsdev's manifest of machine-owned generated files, which
// bump-tool-pins.sh reads from the repository root. Taking the name from
// state.ManifestFile keeps the script's literal bound to it.
var bumpManifest = state.ManifestFile()

// bumpRun is one bump-tool-pins.sh run: the env files passed to it, in
// order, the manifest content in its working directory ("" for none), the
// upstream releases its stubs serve and whether it opens a PR.
type bumpRun struct {
	envFiles []string
	manifest string
	gh       map[string][]upstreamRelease
	proxy    map[string][]upstreamRelease
	nix      []nixRelease
	openPR   bool
}

// bumpResult is what a bumpRun left behind: the exit code, combined output,
// each env file's path (relative to the working directory) and content
// afterwards, the manifest afterwards, the upstream requests the stubs
// answered and, with openPR, the git and gh PR commands the script ran.
type bumpResult struct {
	code     int
	out      string
	paths    []string
	after    []string
	manifest string
	calls    []string
	prEvents []string
}

// runBumpFiles runs bump-tool-pins.sh with run's env files, each written to
// its own path, against stub upstreams, and with stub git and gh pr
// commands that only record their arguments.
func runBumpFiles(t *testing.T, run bumpRun) bumpResult {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("bump-tool-pins.sh runs on the Linux CI runner; it needs bash")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	upstream := filepath.Join(root, "upstream")
	for _, d := range []string{binDir, filepath.Join(upstream, "github"), filepath.Join(upstream, "goproxy"), filepath.Join(upstream, "nix")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tool := range bumpTools {
		p, err := exec.LookPath(tool)
		if err != nil {
			t.Skipf("%s is not installed; the CI runner provides it", tool)
		}
		if err := os.Symlink(p, filepath.Join(binDir, tool)); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(t, filepath.Join(binDir, "gh"), fakeGH)
	writeExec(t, filepath.Join(binDir, "curl"), fakeUpstreamCurl)
	writeExec(t, filepath.Join(binDir, "git"), fakeGit)

	now := time.Now().UTC()
	writeNixUpstream(t, filepath.Join(upstream, "github"), filepath.Join(upstream, "nix"), run.nix, now)
	for repo, rels := range run.gh {
		items := make([]string, 0, len(rels))
		for _, r := range rels {
			items = append(items, fmt.Sprintf(`{"tag_name": %q, "published_at": %q, "draft": false, "prerelease": %t}`,
				r.version, r.published(now), r.prerelease))
		}
		writeFile(t, filepath.Join(upstream, "github", strings.ReplaceAll(repo, "/", "_")+".json"),
			"["+strings.Join(items, ",")+"]")
	}
	for mod, rels := range run.proxy {
		dir := filepath.Join(upstream, "goproxy", strings.ReplaceAll(mod, "/", "_"))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		var list strings.Builder
		for _, r := range rels {
			list.WriteString(r.version + "\n")
			writeFile(t, filepath.Join(dir, r.version+".info"),
				fmt.Sprintf(`{"Version": %q, "Time": %q}`, r.version, r.published(now)))
		}
		writeFile(t, filepath.Join(dir, "list"), list.String())
	}
	script, err := filepath.Abs("bump-tool-pins.sh")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{script}
	if run.openPR {
		args = append(args, "--open-pr")
	}
	// The script runs from root, as from the repository root in CI, and is
	// passed each env file relative to it.
	envPaths := make([]string, len(run.envFiles))
	for i, content := range run.envFiles {
		dir := filepath.Join(root, "env", strconv.Itoa(i))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		envPaths[i] = "env/" + strconv.Itoa(i) + "/tool-versions.env"
		writeFile(t, filepath.Join(dir, "tool-versions.env"), content)
		args = append(args, envPaths[i])
	}
	manifestPath := filepath.Join(root, bumpManifest)
	if run.manifest != "" {
		writeFile(t, manifestPath, run.manifest)
	}

	cmd := exec.Command(filepath.Join(binDir, "bash"), args...)
	cmd.Dir = root
	cmd.Env = []string{"PATH=" + binDir, "HOME=" + root, "TMPDIR=" + root, "FAKE_UPSTREAM=" + upstream}
	out, err := cmd.CombinedOutput()
	res := bumpResult{out: string(out), paths: envPaths}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.code = exitErr.ExitCode()
	default:
		t.Fatalf("running bump-tool-pins.sh: %v", err)
	}
	for _, p := range envPaths {
		after, rerr := os.ReadFile(filepath.Join(root, filepath.FromSlash(p)))
		if rerr != nil {
			t.Fatal(rerr)
		}
		res.after = append(res.after, string(after))
	}
	if run.manifest != "" {
		after, rerr := os.ReadFile(manifestPath)
		if rerr != nil {
			t.Fatal(rerr)
		}
		res.manifest = string(after)
	}
	res.calls = readLog(t, filepath.Join(upstream, "calls.log"))
	res.prEvents = readLog(t, filepath.Join(upstream, "pr.log"))
	return res
}

// readLog returns the lines a stub appended to path, or nil when it wrote
// none.
func readLog(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
}

// writeNixUpstream serves rels as the tags of example/nix and, for the
// published ones, their tarball digests on the fake releases.nixos.org, each
// with a Last-Modified time of the release's age.
func writeNixUpstream(t *testing.T, githubDir, nixDir string, rels []nixRelease, now time.Time) {
	t.Helper()
	if rels == nil {
		return
	}
	tags := make([]string, 0, len(rels))
	for _, r := range rels {
		tags = append(tags, fmt.Sprintf(`{"name": %q}`, r.version))
		if !r.published {
			continue
		}
		dir := filepath.Join(nixDir, "nix-"+r.version)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		systems := r.systems
		if systems == nil {
			systems = nixSystems
		}
		for _, sys := range systems {
			f := filepath.Join(dir, "nix-"+r.version+"-"+sys+".tar.xz.sha256")
			writeFile(t, f, nixDigest(r.version, sys))
			writeFile(t, f+".lm", now.Add(-r.age).Format(http.TimeFormat))
		}
	}
	writeFile(t, filepath.Join(githubDir, "example_nix.tags.json"), "["+strings.Join(tags, ",")+"]")
}

func writeExec(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil { //nolint:gosec // test stub must be executable
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const day = 24 * time.Hour

// TestBumpToolPins covers the U26-02 tool pin updater: it moves a pin to the
// newest exact release that has aged past the cooldown, even when a newer
// release is still young, never downgrades, and fails without rewriting
// anything on malformed upstream data.
func TestBumpToolPins(t *testing.T) {
	t.Parallel()

	bumped := strings.NewReplacer("TOOL_VERSION=v1.2.3", "TOOL_VERSION=v1.3.0",
		"MOD_VERSION=v0.4.0", "MOD_VERSION=v0.5.1").Replace(bumpEnvFile)
	tests := []struct {
		name     string
		envFile  string
		gh       map[string][]upstreamRelease
		proxy    map[string][]upstreamRelease
		wantCode int
		wantEnv  string
	}{
		{
			name:    "aged releases are taken",
			envFile: bumpEnvFile,
			gh:      map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 4 * day}}},
			proxy:   map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", age: 10 * day}}},
			wantEnv: bumped,
		},
		{
			name:    "releases inside the cooldown are held back",
			envFile: bumpEnvFile,
			gh:      map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 2 * day}}},
			proxy:   map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", age: time.Hour}}},
			wantEnv: bumpEnvFile,
		},
		{
			name:    "an aged release is taken when a newer one is still young",
			envFile: bumpEnvFile,
			gh: map[string][]upstreamRelease{"example/tool": {
				{version: "v1.3.1", age: day}, {version: "v1.3.0", age: 5 * day}, {version: "v1.2.9", age: 9 * day},
			}},
			proxy: map[string][]upstreamRelease{"example.com/mod": {
				{version: "v0.3.0", age: 90 * day}, {version: "v0.5.1", age: 4 * day}, {version: "v0.5.2", age: time.Hour},
			}},
			wantEnv: bumped,
		},
		{
			name:    "never downgrades",
			envFile: bumpEnvFile,
			gh:      map[string][]upstreamRelease{"example/tool": {{version: "v1.1.4", age: 400 * day}}},
			proxy:   map[string][]upstreamRelease{"example.com/mod": {{version: "v0.4.0", age: 30 * day}}},
			wantEnv: bumpEnvFile,
		},
		{
			name:    "prereleases and non-exact versions are ignored",
			envFile: bumpEnvFile,
			gh: map[string][]upstreamRelease{"example/tool": {
				{version: "v1.4.0", age: 10 * day, prerelease: true}, {version: "v1.3.0-rc.1", age: 10 * day},
			}},
			proxy: map[string][]upstreamRelease{"example.com/mod": {
				{version: "v0.5.0-rc.1", age: 10 * day}, {version: "v0.5.1", age: 10 * day},
			}},
			wantEnv: strings.Replace(bumpEnvFile, "MOD_VERSION=v0.4.0", "MOD_VERSION=v0.5.1", 1),
		},
		{
			name:     "unreadable release time fails without rewriting",
			envFile:  bumpEnvFile,
			gh:       map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 10 * day}}},
			proxy:    map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", time: "yesterday"}}},
			wantCode: 1,
			wantEnv:  bumpEnvFile,
		},
		{
			name:     "missing upstream fails",
			envFile:  bumpEnvFile,
			gh:       map[string][]upstreamRelease{},
			proxy:    map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", age: 10 * day}}},
			wantCode: 1,
			wantEnv:  bumpEnvFile,
		},
		{
			name:     "key without a source line fails",
			envFile:  "OTHER_VERSION=v1.0.0\n",
			wantCode: 1,
			wantEnv:  "OTHER_VERSION=v1.0.0\n",
		},
		{
			name:     "unknown source kind fails",
			envFile:  "# source: gitlab example/tool\nTOOL_VERSION=v1.2.3\n",
			wantCode: 1,
			wantEnv:  "# source: gitlab example/tool\nTOOL_VERSION=v1.2.3\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, out, got := runBump(t, tc.envFile, tc.gh, tc.proxy, nil)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, out)
			}
			if got != tc.wantEnv {
				t.Errorf("env file after run:\n%s\nwant:\n%s\noutput:\n%s", got, tc.wantEnv, out)
			}
		})
	}
}

// nixEnvFile renders a Nix pin block: NIX_VERSION and the digest of each
// system's tarball for that version.
func nixEnvFile(version string, systems ...string) string {
	var b strings.Builder
	b.WriteString("# source: nix-release example/nix\nNIX_VERSION=" + version + "\n")
	for _, sys := range systems {
		key := "NIX_SHA256_" + strings.ToUpper(strings.ReplaceAll(sys, "-", "_"))
		fmt.Fprintf(&b, "\n# source: nix-sha256 %s\n%s=%s\n", sys, key, nixDigest(version, sys))
	}
	return b.String()
}

// TestBumpToolPinsNix covers the Nix installer pins: NIX_VERSION moves like
// any other pin, but only to a tag whose tarballs releases.nixos.org has
// published, aged from that publication, and every digest pinned below it
// moves with it to the new release's published SHA-256.
func TestBumpToolPinsNix(t *testing.T) {
	t.Parallel()

	current := nixEnvFile("2.34.8", nixSystems...)
	tests := []struct {
		name     string
		envFile  string
		nix      []nixRelease
		wantCode int
		wantEnv  string
	}{
		{
			name:    "an aged release moves the version and every digest",
			envFile: current,
			nix:     []nixRelease{{version: "2.35.0", age: 5 * day, published: true}, {version: "2.34.8", age: 90 * day, published: true}},
			wantEnv: nixEnvFile("2.35.0", nixSystems...),
		},
		{
			name:    "a release inside the cooldown is held back",
			envFile: current,
			nix:     []nixRelease{{version: "2.35.0", age: day, published: true}},
			wantEnv: current,
		},
		{
			name:    "a tag with no published tarball is not a release",
			envFile: current,
			nix: []nixRelease{
				{version: "2.36.0", age: 30 * day}, {version: "2.35.1", age: 4 * day, published: true}, {version: "2.35.1-rc1", age: 9 * day, published: true},
			},
			wantEnv: nixEnvFile("2.35.1", nixSystems...),
		},
		{
			name:    "never downgrades",
			envFile: current,
			nix:     []nixRelease{{version: "2.33.9", age: 300 * day, published: true}},
			wantEnv: current,
		},
		{
			name:     "a missing digest fails without rewriting",
			envFile:  current,
			nix:      []nixRelease{{version: "2.35.0", age: 5 * day, published: true, systems: []string{"x86_64-linux"}}},
			wantCode: 1,
			wantEnv:  current,
		},
		{
			name:     "a digest before the version fails",
			envFile:  "# source: nix-sha256 x86_64-linux\nNIX_SHA256_X86_64_LINUX=" + nixDigest("2.34.8", "x86_64-linux") + "\n",
			wantCode: 1,
			wantEnv:  "# source: nix-sha256 x86_64-linux\nNIX_SHA256_X86_64_LINUX=" + nixDigest("2.34.8", "x86_64-linux") + "\n",
		},
		{
			name:     "a v-prefixed Nix version fails",
			envFile:  nixEnvFile("v2.34.8"),
			nix:      []nixRelease{},
			wantCode: 1,
			wantEnv:  nixEnvFile("v2.34.8"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code, out, got := runBump(t, tc.envFile, nil, nil, tc.nix)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, out)
			}
			if got != tc.wantEnv {
				t.Errorf("env file after run:\n%s\nwant:\n%s\noutput:\n%s", got, tc.wantEnv, out)
			}
		})
	}
}

// checkBumpFiles reports a run whose exit code or env files differ from the
// wanted ones.
func checkBumpFiles(t *testing.T, res bumpResult, wantCode int, wantEnvs []string) {
	t.Helper()
	if res.code != wantCode {
		t.Errorf("exit code = %d, want %d\noutput:\n%s", res.code, wantCode, res.out)
	}
	for i, want := range wantEnvs {
		if res.after[i] != want {
			t.Errorf("env file %d after run:\n%s\nwant:\n%s\noutput:\n%s", i, res.after[i], want, res.out)
		}
	}
}

// TestBumpToolPins_MultipleFiles covers a run over several env files: each
// file's pins move as a single-file run would move them, and a pin that is
// lower in a later file still moves to the newest aged release.
func TestBumpToolPins_MultipleFiles(t *testing.T) {
	t.Parallel()

	bumped := strings.NewReplacer("TOOL_VERSION=v1.2.3", "TOOL_VERSION=v1.3.0",
		"MOD_VERSION=v0.4.0", "MOD_VERSION=v0.5.1").Replace(bumpEnvFile)
	modFile := func(v string) string { return "# source: goproxy example.com/mod\nMOD_VERSION=" + v + "\n" }
	tests := []struct {
		name     string
		run      bumpRun
		wantEnvs []string
	}{
		{
			name: "github and goproxy pins in one file, the nix-release block in another",
			run: bumpRun{
				envFiles: []string{bumpEnvFile, nixEnvFile("2.34.8", nixSystems...)},
				gh:       map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 4 * day}}},
				proxy:    map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", age: 10 * day}}},
				nix:      []nixRelease{{version: "2.35.0", age: 5 * day, published: true}},
			},
			wantEnvs: []string{bumped, nixEnvFile("2.35.0", nixSystems...)},
		},
		{
			name: "a lower pin in a later file sees releases below the earlier pin",
			run: bumpRun{
				envFiles: []string{modFile("v0.5.0"), modFile("v0.4.0")},
				proxy: map[string][]upstreamRelease{"example.com/mod": {
					{version: "v0.4.5", age: 20 * day}, {version: "v0.5.0", age: 10 * day},
				}},
			},
			wantEnvs: []string{modFile("v0.5.0"), modFile("v0.5.0")},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			checkBumpFiles(t, runBumpFiles(t, tc.run), 0, tc.wantEnvs)
		})
	}
}

// TestBumpToolPins_MultipleFilesAtomic covers malformed upstream data in a
// later file: the run fails before writing any file, including an earlier
// file whose pins would move.
func TestBumpToolPins_MultipleFilesAtomic(t *testing.T) {
	t.Parallel()

	nixFile := nixEnvFile("2.34.8", nixSystems...)
	res := runBumpFiles(t, bumpRun{
		envFiles: []string{bumpEnvFile, nixFile},
		gh:       map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 4 * day}}},
		proxy:    map[string][]upstreamRelease{"example.com/mod": {{version: "v0.5.1", age: 10 * day}}},
		nix:      []nixRelease{{version: "2.35.0", age: 5 * day, published: true, systems: []string{"x86_64-linux"}}},
	})
	checkBumpFiles(t, res, 1, []string{bumpEnvFile, nixFile})
}

// TestBumpToolPins_SameNixPinsMoveTogether covers two identical nix-release
// blocks, as in the CI pin catalog's source and its generated copy: they end
// byte-identical, and the shared release is looked up and aged once.
func TestBumpToolPins_SameNixPinsMoveTogether(t *testing.T) {
	t.Parallel()

	current := nixEnvFile("2.34.8", nixSystems...)
	res := runBumpFiles(t, bumpRun{
		envFiles: []string{current, current},
		nix: []nixRelease{
			{version: "2.36.0", age: day, published: true}, {version: "2.35.0", age: 5 * day, published: true},
		},
	})
	want := nixEnvFile("2.35.0", nixSystems...)
	checkBumpFiles(t, res, 0, []string{want, want})

	once := []string{
		"gh api repos/example/nix/tags?per_page=100",
		"curl -I https://releases.nixos.org/nix/nix-2.35.0/nix-2.35.0-x86_64-linux.tar.xz.sha256",
		"curl -I https://releases.nixos.org/nix/nix-2.36.0/nix-2.36.0-x86_64-linux.tar.xz.sha256",
	}
	for _, sys := range nixSystems {
		once = append(once, "curl https://releases.nixos.org/nix/nix-2.35.0/nix-2.35.0-"+sys+".tar.xz.sha256")
	}
	for _, call := range once {
		if n := countOf(res.calls, call); n != 1 {
			t.Errorf("%q requested %d times, want once; requests:\n%s", call, n, strings.Join(res.calls, "\n"))
		}
	}
}

// TestBumpToolPins_RecordsGeneratedDigests covers a bump of a file qsdev
// generated (the CI pin catalog's copy, .github/nix-release.env): its digest
// in qsdev's manifest moves to the new content in the same run, and with
// --open-pr in the same commit, so `qsdev check` still reads the copy as
// unmodified. Other manifest lines, files the manifest does not record and a
// run that moves nothing or fails leave the manifest untouched.
func TestBumpToolPins_RecordsGeneratedDigests(t *testing.T) {
	t.Parallel()

	current := nixEnvFile("2.34.8", nixSystems...)
	bumped := nixEnvFile("2.35.0", nixSystems...)
	digest := func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])
	}
	other := strings.Repeat("ab", 32) + "  .claude/rules/go-conventions.md\n"
	manifest := func(copyContent string) string {
		return other + digest(copyContent) + "  env/1/tool-versions.env\n"
	}
	tests := []struct {
		name         string
		nix          []nixRelease
		openPR       bool
		wantCode     int
		wantEnvs     []string
		wantManifest string
		wantCommit   string
	}{
		{
			name:         "a moved pin records the copy's new digest",
			nix:          []nixRelease{{version: "2.35.0", age: 5 * day, published: true}},
			wantEnvs:     []string{bumped, bumped},
			wantManifest: manifest(bumped),
		},
		{
			name:         "--open-pr commits the manifest with the pin files",
			nix:          []nixRelease{{version: "2.35.0", age: 5 * day, published: true}},
			openPR:       true,
			wantEnvs:     []string{bumped, bumped},
			wantManifest: manifest(bumped),
			wantCommit:   "git commit -- env/0/tool-versions.env env/1/tool-versions.env " + bumpManifest,
		},
		{
			name:         "no pin moves",
			nix:          []nixRelease{{version: "2.35.0", age: day, published: true}},
			wantEnvs:     []string{current, current},
			wantManifest: manifest(current),
		},
		{
			name:         "a failed run writes nothing",
			nix:          []nixRelease{{version: "2.35.0", age: 5 * day, published: true, systems: []string{"x86_64-linux"}}},
			wantCode:     1,
			wantEnvs:     []string{current, current},
			wantManifest: manifest(current),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runBumpFiles(t, bumpRun{
				envFiles: []string{current, current},
				manifest: manifest(current),
				nix:      tc.nix,
				openPR:   tc.openPR,
			})
			checkBumpFiles(t, res, tc.wantCode, tc.wantEnvs)
			if res.manifest != tc.wantManifest {
				t.Errorf("manifest after run:\n%s\nwant:\n%s\noutput:\n%s", res.manifest, tc.wantManifest, res.out)
			}
			if tc.wantCommit == "" {
				return
			}
			var commits []string
			for _, ev := range res.prEvents {
				if strings.HasPrefix(ev, "git commit ") {
					commits = append(commits, ev)
				}
			}
			if !slices.Equal(commits, []string{tc.wantCommit}) {
				t.Errorf("commits = %q, want [%q]", commits, tc.wantCommit)
			}
		})
	}
}

func countOf(lines []string, s string) int {
	n := 0
	for _, l := range lines {
		if l == s {
			n++
		}
	}
	return n
}

// TestBumpToolPins_OpenPRCommitsAllFiles covers --open-pr over several env
// files: every changed file, and only those, goes into one commit whose
// message summarises each file, and one PR is opened.
func TestBumpToolPins_OpenPRCommitsAllFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gh       map[string][]upstreamRelease
		wantFile []int
	}{
		{
			name:     "both files change",
			gh:       map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: 4 * day}}},
			wantFile: []int{0, 1},
		},
		{
			name:     "only the second file changes",
			gh:       map[string][]upstreamRelease{"example/tool": {{version: "v1.3.0", age: day}}},
			wantFile: []int{1},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runBumpFiles(t, bumpRun{
				envFiles: []string{"# source: github example/tool\nTOOL_VERSION=v1.2.3\n", nixEnvFile("2.34.8", nixSystems...)},
				gh:       tc.gh,
				nix:      []nixRelease{{version: "2.35.0", age: 5 * day, published: true}},
				openPR:   true,
			})
			if res.code != 0 {
				t.Fatalf("exit code = %d, want 0\noutput:\n%s", res.code, res.out)
			}
			var commits, messages []string
			for _, ev := range res.prEvents {
				switch {
				case strings.HasPrefix(ev, "git commit "):
					commits = append(commits, ev)
				case strings.HasPrefix(ev, "message: "):
					messages = append(messages, ev)
				}
			}
			want := make([]string, 0, len(tc.wantFile))
			for _, i := range tc.wantFile {
				want = append(want, res.paths[i])
			}
			if wantCommit := "git commit -- " + strings.Join(want, " "); !slices.Equal(commits, []string{wantCommit}) {
				t.Errorf("commits = %q, want [%q]", commits, wantCommit)
			}
			body := strings.Join(messages, "\n")
			for i, p := range res.paths {
				if got, wantIn := strings.Contains(body, p), slices.Contains(tc.wantFile, i); got != wantIn {
					t.Errorf("commit message mentions %s = %t, want %t\nmessage:\n%s", p, got, wantIn, body)
				}
			}
			if n := countOf(res.prEvents, "gh pr create"); n != 1 {
				t.Errorf("gh pr create ran %d times, want once; events:\n%s", n, strings.Join(res.prEvents, "\n"))
			}
		})
	}
}
