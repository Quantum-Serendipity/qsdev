package python

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// pyprojectFile holds the pyproject.toml fields the module reads.
type pyprojectFile struct {
	Project struct {
		RequiresPython string `toml:"requires-python"`
	} `toml:"project"`
	Tool struct {
		Poetry struct {
			Dependencies map[string]any `toml:"dependencies"`
		} `toml:"poetry"`
	} `toml:"tool"`
}

// pyproject is the parsed view of a project's pyproject.toml. The zero value
// describes a missing or unparseable file.
type pyproject struct {
	file pyprojectFile
	meta toml.MetaData
	ok   bool
}

// readPyproject parses projectRoot/pyproject.toml.
func readPyproject(projectRoot string) pyproject {
	var p pyproject
	meta, err := toml.DecodeFile(filepath.Join(projectRoot, "pyproject.toml"), &p.file)
	if err != nil {
		return pyproject{}
	}
	p.meta, p.ok = meta, true
	return p
}

// hasTool reports whether pyproject.toml defines [tool.<key>...].
func (p pyproject) hasTool(key ...string) bool {
	return p.ok && p.meta.IsDefined(append([]string{"tool"}, key...)...)
}

// requiresPython returns the project's Python version constraint and its
// source: [project] requires-python, else Poetry's
// [tool.poetry.dependencies] python.
func (p pyproject) requiresPython() (spec, source string) {
	if s := strings.TrimSpace(p.file.Project.RequiresPython); s != "" {
		return s, "pyproject.toml requires-python"
	}
	if s, ok := p.file.Tool.Poetry.Dependencies["python"].(string); ok && strings.TrimSpace(s) != "" {
		return s, "pyproject.toml tool.poetry.dependencies.python"
	}
	return "", ""
}

// iniHasSection reports whether the INI-style file at path has a [section]
// header.
func iniHasSection(path, section string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "["+section+"]" {
			return true
		}
	}
	return false
}

// tomlDefines reports whether the TOML file at path defines key.
func tomlDefines(path string, key ...string) bool {
	var v map[string]any
	meta, err := toml.DecodeFile(path, &v)
	return err == nil && meta.IsDefined(key...)
}
