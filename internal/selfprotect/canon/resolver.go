package canon

import (
	"io/fs"
	"os"
	"path/filepath"
)

// Resolver answers filesystem questions about paths — Canonicalize,
// EvalSymlinks, Stat — remembering each answer and every lookup behind it. A
// caller that resolves many paths for one decision (the words of one shell
// command, which repeat a path or share the directories above it) then
// queries the filesystem once per distinct path or directory rather than
// once per component of every word, which matters where each query is slow
// (Windows).
//
// A Resolver sees each path as it was at its first lookup, so it serves one
// decision and is then discarded. The zero value is ready to use, and a nil
// *Resolver answers every call afresh, exactly as the os and filepath
// functions do. It is not safe for concurrent use.
type Resolver struct {
	canonical map[string]lookup[string]
	evaluated map[string]lookup[string]
	stats     map[string]lookup[fs.FileInfo]
	lstats    map[string]lookup[fs.FileInfo]
	links     map[string]lookup[string]
}

// lookup is one remembered answer.
type lookup[T any] struct {
	value T
	err   error
}

// remember returns f(key), calling f only the first time key is seen.
func remember[T any](m *map[string]lookup[T], key string, f func(string) (T, error)) (T, error) {
	if res, ok := (*m)[key]; ok {
		return res.value, res.err
	}
	value, err := f(key)
	if *m == nil {
		*m = make(map[string]lookup[T])
	}
	(*m)[key] = lookup[T]{value: value, err: err}
	return value, err
}

// Canonicalize is the package-level Canonicalize, remembered per path.
func (r *Resolver) Canonicalize(path string) (string, error) {
	if r == nil {
		return canonicalize(nil, path)
	}
	return remember(&r.canonical, path, func(p string) (string, error) { return canonicalize(r, p) })
}

// EvalSymlinks is filepath.EvalSymlinks, remembered per path.
func (r *Resolver) EvalSymlinks(path string) (string, error) {
	if r == nil {
		return filepath.EvalSymlinks(path)
	}
	return remember(&r.evaluated, path, filepath.EvalSymlinks)
}

// Stat is os.Stat, remembered per path.
func (r *Resolver) Stat(path string) (fs.FileInfo, error) {
	if r == nil {
		return os.Stat(path)
	}
	return remember(&r.stats, path, os.Stat)
}

// lstat is os.Lstat, remembered per path.
func (r *Resolver) lstat(path string) (fs.FileInfo, error) {
	if r == nil {
		return os.Lstat(path)
	}
	return remember(&r.lstats, path, os.Lstat)
}

// readlink is os.Readlink, remembered per path.
func (r *Resolver) readlink(path string) (string, error) {
	if r == nil {
		return os.Readlink(path)
	}
	return remember(&r.links, path, os.Readlink)
}
