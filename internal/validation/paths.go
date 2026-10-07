package validation

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// windowsAbsRe matches a drive-absolute Windows path such as C:\Users or C:/x.
var windowsAbsRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// maxReadPathLen caps a configured read path well above any real directory.
const maxReadPathLen = 1024

// CheckBoundaryReadPath reports why p cannot be a file-boundary extra read
// path (.qsdev.yaml hooks.file_boundary.extra_read_paths), or nil when it can.
//
// The path must be absolute, or relative to the home directory ("~/..."), as
// the hook cannot know which directory a relative path meant. It must not be
// the filesystem root, the home directory or an ancestor of it, which would
// lift the read boundary altogether, nor contain a ".." segment, a comma (the
// separator of the list handed to the hook) or a control character. It must
// not be, contain or lie inside a credential store (~/.ssh, ~/.aws,
// /etc/shadow, the sandbox system deny list and secrets.CredentialPaths).
func CheckBoundaryReadPath(p string) error {
	home, err := projectctx.HomeDir()
	if err != nil {
		home = ""
	}
	return checkBoundaryReadPath(p, home, pathmatch.Platform)
}

// checkBoundaryReadPath is CheckBoundaryReadPath for the given home directory,
// comparing paths as a filesystem with opts does (so on macOS and Windows
// "~/.SSH" is the credential store "~/.ssh"); an empty home skips the checks
// of absolute paths against it.
func checkBoundaryReadPath(p, home string, opts pathmatch.Options) error {
	switch {
	case p == "":
		return errors.New("path is empty")
	case len(p) > maxReadPathLen:
		return errors.New("path is too long")
	case strings.TrimSpace(p) != p:
		return errors.New("path has leading or trailing whitespace")
	case strings.ContainsRune(p, ','):
		return errors.New("path contains a comma")
	case strings.ContainsFunc(p, isControl):
		return errors.New("path contains a control character")
	}

	slashed := toSlash(p)
	var rest string
	homeRelative := false
	switch {
	case strings.HasPrefix(slashed, "~/") || slashed == "~":
		rest = strings.TrimPrefix(slashed, "~")
		homeRelative = true
	case windowsAbsRe.MatchString(slashed):
		rest = slashed[2:]
	case strings.HasPrefix(slashed, "/"):
		rest = slashed
	default:
		return errors.New("path must be absolute or start with ~/")
	}
	for _, seg := range strings.Split(rest, "/") {
		if seg == ".." {
			return errors.New(`path contains a ".." segment`)
		}
	}
	if path.Clean("/"+rest) == "/" {
		return errors.New("path is the filesystem root or the home directory, which would lift the read boundary")
	}
	if homeRelative {
		return checkCredentialOverlap(path.Clean("/"+rest), "/", secrets.CredentialPaths(), opts)
	}
	return checkAbsoluteReadPath(slashed[:len(slashed)-len(rest)]+path.Clean("/"+rest), home, opts)
}

// checkAbsoluteReadPath rejects an absolute read path that contains the home
// directory or overlaps a system or per-user credential store.
func checkAbsoluteReadPath(abs, home string, opts pathmatch.Options) error {
	if err := checkCredentialOverlap(abs, "", denylist.SystemDenyPaths(), opts); err != nil {
		return err
	}
	if home == "" {
		return nil
	}
	home = path.Clean(toSlash(home))
	if opts.Within(home, abs) {
		return errors.New("path is or contains the home directory, which would lift the read boundary")
	}
	return checkCredentialOverlap(abs, home+"/", secrets.CredentialPaths(), opts)
}

// checkCredentialOverlap rejects p when it equals, contains or lies inside
// any deny entry (each prefixed with prefix), since reading it would expose
// the credential store.
func checkCredentialOverlap(p, prefix string, deny []string, opts pathmatch.Options) error {
	for _, d := range deny {
		d = path.Clean(prefix + toSlash(d))
		if opts.Within(p, d) || opts.Within(d, p) {
			return fmt.Errorf("path overlaps the credential store %s", d)
		}
	}
	return nil
}

// toSlash converts Windows separators to slashes on every platform, since the
// committed policy may have been written on either.
func toSlash(p string) string {
	return strings.ReplaceAll(p, `\`, "/")
}

// isControl reports whether r is an ASCII control character.
func isControl(r rune) bool {
	return r < 0x20 || r == 0x7f
}
