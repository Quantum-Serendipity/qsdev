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
