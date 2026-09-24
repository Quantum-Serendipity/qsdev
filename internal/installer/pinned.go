package installer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Minimum release ages for packages qsdev installs itself, outside any
// project. They mirror the policy qsdev generates for project dependencies
// (min-release-age=3 in .npmrc, the package guard's default MIN_AGE_DAYS=3,
// and --exclude-newer=7d for uv), which a global install would otherwise
// bypass: npm does not read the project .npmrc in global mode, and the
// package-guard hook only sees the agent's own commands, not qsdev's
// subprocesses.
const (
	NpmMinReleaseAge = 3 * 24 * time.Hour
	UvMinReleaseAge  = 7 * 24 * time.Hour
)

// ErrUnpinned marks a package that names no exact release. Installing it
// would take whatever release the registry serves that day.
var ErrUnpinned = errors.New("no exact pinned version")

// exactSemverPattern matches an exact semver-style version (1.2.3,
// 1.2.3-rc.1) but not a range, tag or dist-tag (^1.2, latest, next).
var exactSemverPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$`)

// IsExactSemver reports whether v names exactly one semver release.
func IsExactSemver(v string) bool {
	return exactSemverPattern.MatchString(v)
}

// ReleaseCutoff returns the RFC 3339 instant age before now: only releases
// published before it are eligible for installation.
func ReleaseCutoff(now time.Time, age time.Duration) string {
	return now.Add(-age).UTC().Format(time.RFC3339)
}

// NpmPackage is one exact npm release to install globally.
type NpmPackage struct {
	Name    string
	Version string
	// RunInstallScripts lets npm run lifecycle scripts (preinstall,
	// install, postinstall) during the install. It is off unless the package
	// verifiably cannot work without them.
	RunInstallScripts bool
}

// NpmGlobalInstallCmd returns the npm command that installs exactly
// pkg.Name@pkg.Version globally. The install is age-gated: npm resolves
// the package and all of its dependencies only among releases published
// before now minus [NpmMinReleaseAge], so a pinned release younger than that
// fails to install. Lifecycle scripts are disabled with --ignore-scripts
// unless pkg.RunInstallScripts is set. A package without an exact version is
// refused with [ErrUnpinned], and a name npm would read as an option is
// refused.
func NpmGlobalInstallCmd(pkg NpmPackage, now time.Time) ([]string, error) {
	if pkg.Name == "" {
		return nil, errors.New("npm package has no name")
	}
	if strings.HasPrefix(pkg.Name, "-") {
		// npm would parse it as an option (--registry=..., --prefix=...).
		return nil, fmt.Errorf("npm package name %q starts with '-'", pkg.Name)
	}
	if !IsExactSemver(pkg.Version) {
		return nil, fmt.Errorf("%w: %s version %q; refusing to install an unpinned package", ErrUnpinned, pkg.Name, pkg.Version)
	}
	cmd := []string{"npm", "install", "-g"}
	if !pkg.RunInstallScripts {
		cmd = append(cmd, "--ignore-scripts")
	}
	return append(cmd, "--before="+ReleaseCutoff(now, NpmMinReleaseAge), pkg.Name+"@"+pkg.Version), nil
}

// commitRevPattern matches a full 40-hex-digit git commit hash.
var commitRevPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

// nixAttrPathPattern matches a Nix attribute path such as devenv or
// python3Packages.black.
var nixAttrPathPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_'-]*(\.[A-Za-z_][A-Za-z0-9_'-]*)*$`)

// forgeFlakeSchemes are the flake reference types whose path may carry the
// revision as its third segment (github:owner/repo/<rev>).
var forgeFlakeSchemes = []string{"github:", "gitlab:", "sourcehut:"}

// gitFlakeSchemePrefix prefixes the git flake reference types (git+https:,
// git+ssh:, git+file:, ...). Nix checks a git fetch out at the rev it names,
// so a rev query parameter on one of these pins it.
const gitFlakeSchemePrefix = "git+"

// IsPinnedFlakeRef reports whether ref names exactly one commit: a forge
// reference whose revision is a full commit hash, either as its third path
// segment (github:NixOS/nixpkgs/<rev>) or as a rev=<commit> query parameter
// (github:NixOS/nixpkgs?rev=<rev>), or a git+ reference with a rev=<commit>
// query parameter. A registry name (nixpkgs, flake:nixpkgs), branch or tag
// resolves to whatever that name points at when the install runs, so it is
// not pinned. Neither is an indirect reference with a rev (nixpkgs?rev=...),
// which the registry can map to a path: or tarball flake that ignores the
// rev, nor any path:, tarball or file reference.
func IsPinnedFlakeRef(ref string) bool {
	if strings.ContainsAny(ref, "# \t\n") {
		return false
	}
	path, query, _ := strings.Cut(ref, "?")
	hasRev := false
	for param := range strings.SplitSeq(query, "&") {
		if rev, ok := strings.CutPrefix(param, "rev="); ok && commitRevPattern.MatchString(rev) {
			hasRev = true
		}
	}
	if strings.HasPrefix(path, gitFlakeSchemePrefix) {
		return hasRev
	}
	for _, scheme := range forgeFlakeSchemes {
		rest, ok := strings.CutPrefix(path, scheme)
		if !ok {
			continue
		}
		segs := strings.Split(rest, "/")
		if len(segs) < 2 || segs[0] == "" || segs[1] == "" {
			return false
		}
		switch len(segs) {
		case 2:
			return hasRev
		case 3:
			return commitRevPattern.MatchString(segs[2])
		}
		return false
	}
	return false
}

// NixPackage is one package to install into the user's Nix profile from a
// flake pinned to an exact commit.
type NixPackage struct {
	// Flake is the pinned flake reference, e.g. github:NixOS/nixpkgs/<rev>.
	Flake string
	// Attribute is the package's attribute path within Flake, e.g. devenv.
	Attribute string
}

// NixProfileInstallCmd returns the command that installs pkg.Attribute from
// pkg.Flake into the user's Nix profile. The flake must be pinned to a
// commit (see [IsPinnedFlakeRef]) so the install cannot follow the mutable
// user or system registry, and the command sets accept-flake-config to
// false so a nixConfig the flake declares (extra substituters, trusted
// public keys) is ignored even when the user's nix.conf would accept it.
// An unpinned flake is refused with [ErrUnpinned].
func NixProfileInstallCmd(pkg NixPackage) ([]string, error) {
	if strings.HasPrefix(pkg.Flake, "-") {
		// nix would parse it as an option.
		return nil, fmt.Errorf("flake reference %q starts with '-'", pkg.Flake)
	}
	if !IsPinnedFlakeRef(pkg.Flake) {
		return nil, fmt.Errorf("%w: flake %q is not pinned to a commit; refusing to install %s from it",
			ErrUnpinned, pkg.Flake, pkg.Attribute)
	}
	if !nixAttrPathPattern.MatchString(pkg.Attribute) {
		return nil, fmt.Errorf("invalid Nix attribute path %q", pkg.Attribute)
	}
	return []string{
		"nix", "profile", "install",
		"--option", "accept-flake-config", "false",
		pkg.Flake + "#" + pkg.Attribute,
	}, nil
}
