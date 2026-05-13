package sysinfo

import (
	"os"
	"strings"
)

// parseOSRelease reads an os-release format file and returns a key-value map.
// It handles KEY=value, KEY="quoted value", and KEY='single quoted' formats.
// Comment lines (starting with #) and empty lines are skipped.
// Returns an empty map on read error.
func parseOSRelease(path string) map[string]string {
	result := make(map[string]string)

	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}

		key := line[:idx]
		value := line[idx+1:]

		// Strip matching quotes (double or single).
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}

	return result
}

// determineFamily maps a distro ID and ID_LIKE string to a family name.
func determineFamily(id, idLike string) string {
	// Direct ID matches first.
	switch id {
	case "nixos":
		return "nixos"
	case "alpine":
		return "alpine"
	case "void":
		return "void"
	case "gentoo":
		return "gentoo"
	case "debian", "ubuntu":
		return "debian"
	case "fedora":
		return "rhel"
	case "arch", "manjaro", "endeavouros", "garuda":
		return "arch"
	case "opensuse-tumbleweed", "opensuse-leap":
		return "suse"
	}

	// ID_LIKE chain fallback.
	if idLike != "" {
		if strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu") {
			return "debian"
		}
		if strings.Contains(idLike, "fedora") || strings.Contains(idLike, "rhel") {
			return "rhel"
		}
		if strings.Contains(idLike, "suse") || strings.Contains(idLike, "opensuse") {
			return "suse"
		}
		if strings.Contains(idLike, "arch") {
			return "arch"
		}
	}

	return "unknown"
}
