package haskell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ltsCompiler records that Stackage LTS series major moved to GHC ghc at
// minor release minor (lts-<major>.<minor>).
type ltsCompiler struct {
	major, minor int
	ghc          string
}

// ltsCompilers lists, in order, every LTS release from lts-12.0 on that
// changed GHC version: a snapshot uses the GHC of the last entry of its series
// at or below its minor release. The data is the resolver.compiler field of
// commercialhaskell/stackage-snapshots lts/<major>/<minor>.yaml, verified
// through ltsVerifiedMajor.ltsVerifiedMinor. Snapshots outside that range
// resolve to no GHC version rather than a guessed one.
var ltsCompilers = []ltsCompiler{
	{12, 0, "8.4.3"}, {12, 15, "8.4.4"},
	{13, 0, "8.6.3"}, {13, 12, "8.6.4"}, {13, 20, "8.6.5"},
	{14, 0, "8.6.5"},
	{15, 0, "8.8.2"}, {15, 4, "8.8.3"},
	{16, 0, "8.8.3"}, {16, 12, "8.8.4"},
	{17, 0, "8.10.3"}, {17, 3, "8.10.4"},
	{18, 0, "8.10.4"}, {18, 7, "8.10.6"}, {18, 9, "8.10.7"},
	{19, 0, "9.0.2"},
	{20, 0, "9.2.5"}, {20, 12, "9.2.6"}, {20, 13, "9.2.7"}, {20, 25, "9.2.8"},
	{21, 0, "9.4.5"}, {21, 8, "9.4.6"}, {21, 12, "9.4.7"}, {21, 22, "9.4.8"},
	{22, 0, "9.6.3"}, {22, 7, "9.6.4"}, {22, 21, "9.6.5"}, {22, 28, "9.6.6"}, {22, 44, "9.6.7"},
	{23, 0, "9.8.4"},
	{24, 0, "9.10.2"}, {24, 12, "9.10.3"},
}

// The newest LTS release ltsCompilers was verified against. Its series may
// still receive releases, so later ones (and a bare "lts-24") are unknown.
const (
	ltsVerifiedMajor = 24
	ltsVerifiedMinor = 60
)

var (
	// ghcVersionRe matches a full GHC release version, which is what Stack's
	// default compiler-check (match-minor) requires the shell's GHC to equal.
	ghcVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	// ltsRe matches an LTS snapshot name: "lts-22.44", or "lts-22" for the
	// series' latest release.
	ltsRe = regexp.MustCompile(`^lts-([0-9]+)(?:\.([0-9]+))?$`)
	// ltsURLRe matches a snapshot URL into the stackage-snapshots layout.
	ltsURLRe = regexp.MustCompile(`/lts/([0-9]+)/([0-9]+)\.yaml$`)
)

// stackCompiler is the GHC a Stack project builds with.
type stackCompiler struct {
	// source describes where the version came from, for messages:
	// `snapshot "lts-22.44"` or `compiler "ghc-9.6.7"`.
	source string
	// ghc is the full GHC version, or "" when source names no GHC that qsdev
	// can determine (a nightly or custom snapshot, or an LTS release newer
	// than ltsCompilers).
	ghc string
}

// errNoStackSnapshot reports a stack.yaml without a snapshot or resolver.
var errNoStackSnapshot = errors.New("stack.yaml sets no snapshot or resolver")

// readStackCompiler returns the GHC that projectRoot's stack.yaml builds
// with: its compiler override when set, else its snapshot's (the `snapshot`
// key, or the older `resolver`).
func readStackCompiler(projectRoot string) (stackCompiler, error) {
	data, err := os.ReadFile(filepath.Join(projectRoot, "stack.yaml"))
	if err != nil {
		return stackCompiler{}, fmt.Errorf("reading stack.yaml: %w", err)
	}
	var doc struct {
		Snapshot any    `yaml:"snapshot"`
		Resolver any    `yaml:"resolver"`
		Compiler string `yaml:"compiler"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return stackCompiler{}, fmt.Errorf("parsing stack.yaml: %w", err)
	}
	if doc.Compiler != "" {
		return stackCompiler{
			source: fmt.Sprintf("compiler %q", doc.Compiler),
			ghc:    ghcFromCompiler(doc.Compiler),
		}, nil
	}
	snapshot := doc.Snapshot
	if snapshot == nil {
		snapshot = doc.Resolver
	}
	switch s := snapshot.(type) {
	case nil:
		return stackCompiler{}, errNoStackSnapshot
	case string:
		return stackCompiler{source: fmt.Sprintf("snapshot %q", s), ghc: snapshotGHC(s)}, nil
	default:
		return stackCompiler{source: "a custom snapshot"}, nil
	}
}

// snapshotGHC returns the GHC version a Stack snapshot name uses, or "" when
// it cannot be determined without fetching the snapshot.
func snapshotGHC(snapshot string) string {
	snapshot = strings.TrimSpace(snapshot)
	if v := ghcFromCompiler(snapshot); v != "" {
		return v
	}
	if m := ltsRe.FindStringSubmatch(snapshot); m != nil {
		return ltsGHC(m[1], m[2])
	}
	if m := ltsURLRe.FindStringSubmatch(snapshot); m != nil {
		return ltsGHC(m[1], m[2])
	}
	return ""
}

// ghcFromCompiler returns the version of a "ghc-X.Y.Z" compiler name, or ""
// for anything else (ghcjs, ghc-git, a malformed version).
func ghcFromCompiler(compiler string) string {
	v, ok := strings.CutPrefix(strings.TrimSpace(compiler), "ghc-")
	if !ok || !ghcVersionRe.MatchString(v) {
		return ""
	}
	return v
}

// ltsGHC returns the GHC of LTS release major.minor; an empty minor means
// the series' latest release, which is only known for a closed series.
func ltsGHC(majorStr, minorStr string) string {
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return ""
	}
	minor := -1
	if minorStr != "" {
		if minor, err = strconv.Atoi(minorStr); err != nil {
			return ""
		}
	}
	switch {
	case major > ltsVerifiedMajor,
		major == ltsVerifiedMajor && (minor < 0 || minor > ltsVerifiedMinor):
		return ""
	}
	ghc := ""
	for _, c := range ltsCompilers {
		if c.major == major && (minor < 0 || c.minor <= minor) {
			ghc = c.ghc
		}
	}
	return ghc
}

// compilerAttr returns the nixpkgs haskell.compiler attribute for GHC
// version v ("9.6.7" -> "ghc967", "9.6" -> "ghc96").
func compilerAttr(v string) string {
	return "ghc" + strings.ReplaceAll(v, ".", "")
}
