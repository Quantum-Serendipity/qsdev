package claudesettings

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
)

// Effective is the settings view Claude Code runs a project with: the
// committed file overlaid by the local one.
type Effective struct {
	// Settings is the merged view.
	Settings
	// Project and Local are the parsed files, nil when the file is absent.
	Project, Local *Settings
	// Sources maps KeyDisableAllHooks, KeyDefaultMode and
	// KeyDisableBypassPermissionsMode to the RelPath of every file whose
	// value is in effect, in read order.
	Sources map[string][]string
}

// Read returns the effective settings of the project at projectRoot. It
// reads ProjectRelPath, then LocalRelPath; a missing file is skipped. The
// merge follows Claude Code precedence, except that disableAllHooks is
// fail-safe:
//   - hooks and deny rules are the union of both files;
//   - an env variable set locally wins;
//   - defaultMode and disableBypassPermissionsMode set locally win;
//   - hooks are disabled when either file disables them.
//
// An empty projectRoot yields an empty view. User and managed settings are
// not read, so the result does not depend on the machine.
func Read(projectRoot string) (Effective, error) {
	e := Effective{
		Settings: Settings{Hooks: map[string][]Matcher{}, Env: map[string]string{}},
		Sources:  map[string][]string{},
	}
	if projectRoot == "" {
		return e, nil
	}
	for _, rel := range []string{ProjectRelPath, LocalRelPath} {
		s, err := readFile(projectRoot, rel)
		if err != nil {
			return Effective{}, err
		}
		if s == nil {
			continue
		}
		if rel == ProjectRelPath {
			e.Project = s
		} else {
			e.Local = s
		}
		e.overlay(rel, *s)
	}
	return e, nil
}

// readFile parses the settings file rel under root; it returns nil when the
// file does not exist.
func readFile(root, rel string) (*Settings, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", rel, err)
	}
	s, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	return &s, nil
}

// overlay merges s, read from rel, over the view built so far.
func (e *Effective) overlay(rel string, s Settings) {
	e.Deny = append(e.Deny, s.Deny...)
	for event, matchers := range s.Hooks {
		e.Hooks[event] = append(e.Hooks[event], matchers...)
	}
	maps.Copy(e.Env, s.Env)
	if s.DisableAllHooks {
		e.DisableAllHooks = true
		e.Sources[KeyDisableAllHooks] = append(e.Sources[KeyDisableAllHooks], rel)
	}
	if s.DefaultMode != "" {
		e.DefaultMode = s.DefaultMode
		e.Sources[KeyDefaultMode] = []string{rel}
	}
	if s.DisableBypassPermissionsMode != "" {
		e.DisableBypassPermissionsMode = s.DisableBypassPermissionsMode
		e.Sources[KeyDisableBypassPermissionsMode] = []string{rel}
	}
}
