package claudesettings

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
)

// UserLabel names the user settings file in Sources and messages.
const UserLabel = "~/.claude/settings.json"

// userSettingsFile is the name of the user settings file in UserDir.
const userSettingsFile = "settings.json"

// Effective is the settings view Claude Code runs a project with: the
// committed file overlaid by the local one, and, when read with
// ReadOptions.UserDir, the user file beneath both.
type Effective struct {
	// Settings is the merged view.
	Settings
	// User, Project and Local are the parsed files, nil when the file is
	// absent or (User) not read. A file whose Unloadable is set is not part
	// of the merged view.
	User, Project, Local *Settings
	// Sources maps KeyDisableAllHooks, KeyDefaultMode,
	// KeyDisableBypassPermissionsMode and EnvSourceKey of each env variable
	// to the RelPath (UserLabel for the user file) of every file whose value
	// is in effect, in read order.
	Sources map[string][]string
}

// ReadOptions selects the settings files ReadWith reads beyond the project's.
type ReadOptions struct {
	// UserDir is the Claude Code user configuration directory holding the
	// user settings.json (canon.ClaudeConfigDir); empty skips it, so the
	// result does not depend on the machine.
	UserDir string
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
// A file Claude Code refuses to load (Settings.Unloadable) contributes
// nothing to the merged view, as in Claude Code; it is still returned in
// Project or Local.
//
// An empty projectRoot yields an empty view. User and managed settings are
// not read, so the result does not depend on the machine.
func Read(projectRoot string) (Effective, error) {
	return ReadWith(projectRoot, ReadOptions{})
}

// ReadWith is Read that also reads the files opts selects. The user file is
// read first, beneath the project files, with the same merge; managed
// settings are never read. An empty projectRoot still yields an empty view.
func ReadWith(projectRoot string, opts ReadOptions) (Effective, error) {
	e := Effective{
		Settings: Settings{Hooks: map[string][]Matcher{}, Env: map[string]string{}},
		Sources:  map[string][]string{},
	}
	if projectRoot == "" {
		return e, nil
	}
	if opts.UserDir != "" {
		u, err := readFile(filepath.Join(opts.UserDir, userSettingsFile), UserLabel)
		if err != nil {
			return Effective{}, err
		}
		if u != nil {
			e.User = u
			e.overlay(UserLabel, *u)
		}
	}
	for _, rel := range []string{ProjectRelPath, LocalRelPath} {
		s, err := readFile(filepath.Join(projectRoot, filepath.FromSlash(rel)), rel)
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

// EnvSourceKey is the Effective.Sources key of the env variable name.
func EnvSourceKey(name string) string {
	return KeyEnv + "." + name
}

// readFile parses the settings file at path, named label in errors; it
// returns nil when the file does not exist.
func readFile(path, label string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", label, err)
	}
	s, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return &s, nil
}

// overlay merges s, read from rel, over the view built so far, unless Claude
// Code refuses to load it.
func (e *Effective) overlay(rel string, s Settings) {
	if s.Unloadable != "" {
		return
	}
	e.Deny = append(e.Deny, s.Deny...)
	for event, matchers := range s.Hooks {
		e.Hooks[event] = append(e.Hooks[event], matchers...)
	}
	maps.Copy(e.Env, s.Env)
	for k := range s.Env {
		e.Sources[EnvSourceKey(k)] = []string{rel}
	}
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
