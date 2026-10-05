// Package pathmatch is the single definition of how the host filesystem
// compares path names: whether case folds, which Windows name aliases are
// ignored, and when one path lies within another. Every check that compares a
// path against a protected, denied or allowed location builds its keys here,
// so two spellings of one file can never compare unequal in one place and
// equal in another.
//
// It is a foundation package: it imports only the standard library and does
// no I/O. Symlink resolution belongs to the callers (selfprotect/canon,
// sandbox/denylist), which pass the paths they resolved.
package pathmatch

import (
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Options describe how a filesystem compares names.
type Options struct {
	// FoldCase compares names case-insensitively (macOS and Windows defaults).
	FoldCase bool
	// WindowsAliases strips the name aliases Windows ignores when it opens a
	// file: trailing dots and spaces, and an alternate-data-stream suffix.
	WindowsAliases bool
}

// Platform is the Options of the host's default filesystem.
var Platform = Options{
	FoldCase:       runtime.GOOS == "darwin" || runtime.GOOS == "windows",
	WindowsAliases: runtime.GOOS == "windows",
}

// CaseInsensitiveFS reports whether the host's default filesystem folds case,
// in which case `.SSH/id_rsa` names the same file as `.ssh/id_rsa`.
func CaseInsensitiveFS() bool {
	return Platform.FoldCase
}

// Key returns p in the form path comparisons use on this platform
// (Platform.Key). Two spellings of one file have equal keys.
func Key(p string) string {
	return Platform.Key(p)
}

// Within reports whether p is root or lies below it on this platform
// (Platform.Within).
func Within(p, root string) bool {
	return Platform.Within(p, root)
}

// StrictlyWithin reports whether p lies below root, and is not root itself, on
// this platform (Platform.StrictlyWithin).
func StrictlyWithin(p, root string) bool {
	return Platform.StrictlyWithin(p, root)
}

// Key returns p in the form path comparisons use: slash-separated, with the
// filesystem's name aliases normalized away. Folding case before stripping
// aliases gives the same key, since folding never adds or removes the ':', '.'
// and ' ' that stripping looks at; it lets the separator conversion and the
// fold share one copy of p.
func (o Options) Key(p string) string {
	var s string
	if o.FoldCase {
		s = toSlashLower(p)
	} else {
		s = filepath.ToSlash(p)
	}
	if o.WindowsAliases {
		s = stripWindowsAliases(s)
	}
	return s
}

// Within reports whether p is root or lies below it, comparing keys at whole
// path components: "/a/bc" is not within "/a/b". A root ending in a separator
// (the filesystem root "/", a drive "C:/") contains every path that starts
// with it. Both paths are compared as given; callers clean or resolve them
// first.
func (o Options) Within(p, root string) bool {
	return within(o.Key(p), o.Key(root))
}

// StrictlyWithin reports whether p lies below root and is not root itself:
// root is a proper ancestor directory of p.
func (o Options) StrictlyWithin(p, root string) bool {
	pk, rk := o.Key(p), o.Key(root)
	return pk != rk && within(pk, rk)
}

// within is Within on keys. It slices rather than building root+"/", so a
// check that needs no key copy allocates nothing.
func within(pk, rk string) bool {
	if !strings.HasPrefix(pk, rk) {
		return false
	}
	return len(pk) == len(rk) || strings.HasSuffix(rk, "/") || pk[len(rk)] == '/'
}

// toSlashLower is strings.ToLower(filepath.ToSlash(p)) in a single copy, and
// none when p is already slash-separated lower case.
func toSlashLower(p string) string {
	return sepToSlashLower(p, filepath.Separator)
}

// sepToSlashLower is toSlashLower for an explicit separator, so the Windows
// form can be tested on any platform.
func sepToSlashLower(p string, sep rune) string {
	if sep == '/' {
		return strings.ToLower(p)
	}
	// Paths are almost always ASCII: convert them byte by byte, as
	// strings.ToLower's own fast path does, instead of decoding runes.
	if sep < utf8.RuneSelf {
		if s, ok := asciiSepToSlashLower(p, byte(sep)); ok {
			return s
		}
	}
	return strings.Map(func(r rune) rune {
		if r == sep {
			return '/'
		}
		return unicode.ToLower(r)
	}, p)
}

// asciiSepToSlashLower is sepToSlashLower for an all-ASCII p; ok is false when
// p holds a non-ASCII byte. It copies p only when a byte changes.
func asciiSepToSlashLower(p string, sep byte) (string, bool) {
	changed := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		if c >= utf8.RuneSelf {
			return "", false
		}
		changed = changed || c == sep || ('A' <= c && c <= 'Z')
	}
	if !changed {
		return p, true
	}
	var b strings.Builder
	b.Grow(len(p))
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c == sep:
			c = '/'
		case 'A' <= c && c <= 'Z':
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	return b.String(), true
}

// stripWindowsAliases removes, from each component of a slash-separated path,
// an alternate-data-stream suffix ("settings.json::$DATA" -> "settings.json")
// and trailing dots and spaces (".claude." -> ".claude"), which Windows
// discards when it opens the file. A drive component ("C:") and "."/".." are
// left alone. Every path a rule checks on Windows goes through it, so it
// copies s only from the first component it changes, and not at all when
// there is none (the usual case).
func stripWindowsAliases(s string) string {
	var b strings.Builder
	copying := false
	for start := 0; start <= len(s); {
		end := strings.IndexByte(s[start:], '/')
		if end < 0 {
			end = len(s)
		} else {
			end += start
		}
		part := stripComponentAliases(s[start:end])
		if !copying && len(part) != end-start {
			copying = true
			b.Grow(len(s))
			b.WriteString(s[:start])
		}
		if copying {
			b.WriteString(part)
			if end < len(s) {
				b.WriteByte('/')
			}
		}
		start = end + 1
	}
	if !copying {
		return s
	}
	return b.String()
}

// stripComponentAliases is stripWindowsAliases for one path component. The
// result is always a prefix of part.
func stripComponentAliases(part string) string {
	if part == "." || part == ".." || (len(part) == 2 && part[1] == ':') {
		return part
	}
	if j := strings.IndexByte(part, ':'); j >= 0 {
		part = part[:j]
	}
	end := len(part)
	for end > 0 && (part[end-1] == '.' || part[end-1] == ' ') {
		end--
	}
	return part[:end]
}
