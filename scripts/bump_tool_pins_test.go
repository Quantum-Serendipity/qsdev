package scripts_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These tests run scripts/bump-tool-pins.sh against stub gh and curl
// commands that serve canned release metadata, so no real GitHub or Go
// module proxy request is made.

// fakeGH answers `gh api repos/<owner>/<repo>/releases?per_page=100` from
// $FAKE_UPSTREAM/github/<owner>_<repo>.json and fails like a 404 otherwise.
const fakeGH = `#!/bin/sh
[ "$1" = "api" ] || { echo "unexpected gh call: $*" >&2; exit 2; }
path="${2#repos/}"
repo="${path%/releases?per_page=100}"
[ "$repo" != "$path" ] || { echo "unexpected gh api path: $2" >&2; exit 2; }
f="$FAKE_UPSTREAM/github/$(echo "$repo" | tr / _).json"
[ -f "$f" ] || { echo "HTTP 404" >&2; exit 1; }
cat "$f"
`

// fakeProxyCurl answers https://proxy.golang.org/<module>/@v/list and
// .../@v/<version>.info from $FAKE_UPSTREAM/goproxy/<module with / as _>/
// and fails like ` + "`curl -f`" + ` on a 404 (exit 22) otherwise.
const fakeProxyCurl = `#!/bin/sh
url=""
for a in "$@"; do
  case "$a" in https://*) url="$a" ;; esac
done
case "$url" in
  https://proxy.golang.org/*/@v/*) ;;
  *) echo "unexpected curl url: $url" >&2; exit 6 ;;
esac
rest="${url#https://proxy.golang.org/}"
mod="${rest%/@v/*}"
file="${rest##*/@v/}"
f="$FAKE_UPSTREAM/goproxy/$(echo "$mod" | tr / _)/$file"
[ -f "$f" ] || exit 22
cat "$f"
`

// bumpTools are the real binaries bump-tool-pins.sh may use.
var bumpTools = []string{"bash", "sh", "jq", "mktemp", "mv", "rm", "cat", "tr"}

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
func runBump(t *testing.T, envFile string, gh, proxy map[string][]upstreamRelease) (int, string, string) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("bump-tool-pins.sh runs on the Linux CI runner; it needs bash")
	}
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	upstream := filepath.Join(root, "upstream")
	for _, d := range []string{binDir, filepath.Join(upstream, "github"), filepath.Join(upstream, "goproxy")} {
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
	writeExec(t, filepath.Join(binDir, "curl"), fakeProxyCurl)

	now := time.Now().UTC()
	for repo, rels := range gh {
		items := make([]string, 0, len(rels))
		for _, r := range rels {
			items = append(items, fmt.Sprintf(`{"tag_name": %q, "published_at": %q, "draft": false, "prerelease": %t}`,
				r.version, r.published(now), r.prerelease))
		}
		writeFile(t, filepath.Join(upstream, "github", strings.ReplaceAll(repo, "/", "_")+".json"),
			"["+strings.Join(items, ",")+"]")
	}
	for mod, rels := range proxy {
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
	envPath := filepath.Join(root, "tool-versions.env")
	writeFile(t, envPath, envFile)

	cmd := exec.Command(filepath.Join(binDir, "bash"), "bump-tool-pins.sh", envPath)
	cmd.Env = []string{"PATH=" + binDir, "HOME=" + root, "TMPDIR=" + root, "FAKE_UPSTREAM=" + upstream}
	out, err := cmd.CombinedOutput()
	code := 0
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("running bump-tool-pins.sh: %v", err)
	}
	after, rerr := os.ReadFile(envPath)
	if rerr != nil {
		t.Fatal(rerr)
	}
	return code, string(out), string(after)
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
			code, out, got := runBump(t, tc.envFile, tc.gh, tc.proxy)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d\noutput:\n%s", code, tc.wantCode, out)
			}
			if got != tc.wantEnv {
				t.Errorf("env file after run:\n%s\nwant:\n%s\noutput:\n%s", got, tc.wantEnv, out)
			}
		})
	}
}
