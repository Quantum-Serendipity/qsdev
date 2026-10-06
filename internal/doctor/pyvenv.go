package doctor

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxPyvenvCfg bounds how much of a pyvenv.cfg is read.
const maxPyvenvCfg = 64 << 10

// pyvenvVersion returns the Python version the virtual environment holding
// the interpreter at binPath records in its pyvenv.cfg, or "" when there is
// none. Python looks for the file beside the interpreter and one directory
// up (bin/ or Scripts/ sits in the environment root); venv writes
// "version", and virtualenv, uv and devenv "version_info". Nothing is run.
func pyvenvVersion(binPath string) string {
	dir := filepath.Dir(binPath)
	for _, cfg := range []string{filepath.Join(dir, "pyvenv.cfg"), filepath.Join(filepath.Dir(dir), "pyvenv.cfg")} {
		if fields, ok := readPyvenvCfg(cfg); ok {
			for _, key := range []string{"version", "version_info"} {
				if v := numericVersion(fields[key]); v != "" {
					return v
				}
			}
			return ""
		}
	}
	return ""
}

// readPyvenvCfg parses the "key = value" lines of the pyvenv.cfg at path,
// with keys lower-cased. It reports false when path is not a regular file
// that can be read.
func readPyvenvCfg(path string) (map[string]string, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	f, err := os.Open(path) //nolint:gosec // a fixed file name beside a binary found on PATH; only parsed
	if err != nil {
		return nil, false
	}
	defer func() { _ = f.Close() }()

	fields := make(map[string]string)
	sc := bufio.NewScanner(io.LimitReader(f, maxPyvenvCfg))
	for sc.Scan() {
		if key, val, ok := strings.Cut(sc.Text(), "="); ok {
			fields[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(val)
		}
	}
	return fields, true
}

// numericVersion returns the leading dot-separated numeric components of v
// ("3.12.4.final.0" → "3.12.4"), or "" when v does not start with one. The
// file is repository content, so nothing else of it is kept.
func numericVersion(v string) string {
	var parts []string
	for p := range strings.SplitSeq(v, ".") {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			break
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, ".")
}
