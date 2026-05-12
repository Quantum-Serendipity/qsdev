package claudecode

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// ComputeTemplateVersion returns a content-hash version string for the
// non-skill, non-rule templates embedded in the binary.
func ComputeTemplateVersion() string {
	return computeEmbedHash(func(path string) bool {
		return !strings.HasPrefix(path, "templates/skills/") &&
			!strings.HasPrefix(path, "templates/rules/")
	})
}

// ComputeSkillLibraryVersion returns a content-hash version string for the
// skill and rules libraries.
func ComputeSkillLibraryVersion() string {
	return computeEmbedHash(func(path string) bool {
		return strings.HasPrefix(path, "templates/skills/") ||
			strings.HasPrefix(path, "templates/rules/")
	})
}

// computeEmbedHash walks the embedded FS under "templates", filters paths
// with accept, sorts entries for determinism, concatenates "path\x00content"
// for each file, and returns "sha256:<64-char-hex>".
func computeEmbedHash(accept func(path string) bool) string {
	type entry struct {
		path    string
		content []byte
	}

	var entries []entry

	_ = fs.WalkDir(templateFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) == ".gitkeep" {
			return nil
		}
		if !accept(path) {
			return nil
		}
		data, readErr := templateFS.ReadFile(path)
		if readErr != nil {
			return nil
		}
		entries = append(entries, entry{path: path, content: data})
		return nil
	})

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	h := sha256.New()
	for _, e := range entries {
		h.Write([]byte(e.path))
		h.Write([]byte{0x00})
		h.Write(e.content)
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
