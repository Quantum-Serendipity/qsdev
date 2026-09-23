package claudecode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// hookScriptDir is the project-relative directory holding the generated hook
// scripts that settings.json wires into Claude Code. Files under it are
// security guards: a modified or deleted copy is always restored.
const hookScriptDir = ".claude/hooks/"

// regenWriteOptions controls how planRegenWrites treats generated files the
// user changed since qsdev last wrote them.
type regenWriteOptions struct {
	// force replaces user-modified or user-deleted generated files and
	// overwrites a mergeable file whose merge fails.
	force bool
	// keepUserChanges leaves a recorded non-mergeable file that the user
	// modified (or a recorded file the user deleted) alone unless force is set.
	// Hook scripts are exempt (always restored), as are edits to
	// library-managed files, which the skill library owns.
	keepUserChanges bool
}

// regenPlan is the outcome of planning a regeneration: what to write, what to
// record in state, and the counts reported to the user.
type regenPlan struct {
	writes          []types.GeneratedFile // files whose content must be written
	recorded        []types.GeneratedFile // files to record in state (written or already current)
	mergedOriginals map[string][]byte     // un-merged "ours" content for merged files
	present         map[string]bool       // paths holding generated content after the write
	legacyRemovals  []string              // stale flat-layout skill files safe to delete
	created         int
	updated         int
	unchanged       int
	skipped         int
}

// written reports how many files the plan (re)writes on disk.
func (p *regenPlan) written() int { return p.created + p.updated }

// planRegenWrites decides, for every generated file, whether it is merged,
// rewritten, left unchanged or skipped. It performs no writes, so a merge
// conflict aborts the whole regeneration before anything on disk changes.
//
// Any file present on disk with a mergeable strategy (ThreeWayMerge/
// SectionMarker) is ALWAYS merged — whatever the claude state records about
// it. This preserves user-owned top-level keys such as settings.json "env"
// across every regeneration, including the first run after a top-level
// `qsdev init` (which records state under the devinit state file, so the
// claude addon has no record of settings.json).
//
// A merge that fails on non-empty user content is never silently replaced:
// the file is reported as a conflict unless opts.force is set.
//
// A non-mergeable file is skipped only when its on-disk content already equals
// the generated content; a deleted or tampered hook script is always restored
// with a warning.
func planRegenWrites(files []types.GeneratedFile, existing types.GeneratedState, projectRoot string, opts regenWriteOptions, warn io.Writer) (*regenPlan, error) {
	plan := &regenPlan{
		mergedOriginals: make(map[string][]byte),
		present:         make(map[string]bool),
	}
	plan.legacyRemovals = planLegacySkillCleanup(files, existing, projectRoot, warn)

	var conflicts []string
	for _, f := range files {
		if f.Mode == 0 {
			f.Mode = fileutil.ModeReadWrite
		}
		onDisk, exists, err := readIfExists(filepath.Join(projectRoot, f.Path))
		if err != nil {
			return nil, err
		}
		rec, recorded := existing.Files[f.Path]
		userChanged := recorded && (!exists || state.ComputeHash(onDisk) != rec.Hash)

		if isMergeable(f.Strategy) && exists {
			if conflict := planMerge(plan, f, onDisk, rec, recorded, opts, warn); conflict != "" {
				conflicts = append(conflicts, conflict)
			}
			continue
		}

		if exists && bytes.Equal(onDisk, f.Content) && modeMatches(filepath.Join(projectRoot, f.Path), f.Mode) {
			plan.keep(f)
			continue
		}

		if userChanged && opts.keepUserChanges && !opts.force && !alwaysRestored(f, exists) {
			if exists {
				_, _ = fmt.Fprintf(warn, "Warning: %s was modified since it was generated; skipping (use --force to overwrite)\n", f.Path)
			} else {
				_, _ = fmt.Fprintf(warn, "Warning: %s was deleted since it was generated; skipping (use --force to restore)\n", f.Path)
			}
			plan.skipped++
			continue
		}
		if userChanged && isHookScript(f.Path) {
			_, _ = fmt.Fprintf(warn, "Warning: security hook %s was modified or deleted since it was generated; restoring it\n", f.Path)
		}
		plan.write(f, exists)
	}

	if len(conflicts) > 0 {
		return nil, fmt.Errorf("could not merge generated content into %s; fix the file (e.g. its JSON syntax) and retry, or run '%s claude update --force' to replace it with the generated content",
			strings.Join(conflicts, "; "), branding.Get().AppName)
	}
	return plan, nil
}

// planMerge merges generated content into an existing mergeable file. It
// returns a non-empty conflict description when the merge fails on user
// content that must not be overwritten.
func planMerge(plan *regenPlan, f types.GeneratedFile, onDisk []byte, rec types.FileState, recorded bool, opts regenWriteOptions, warn io.Writer) string {
	merged, mergeErr := merge.Dispatch(f.Path, f.Strategy, rec.BaseContent, onDisk, f.Content)
	if mergeErr != nil {
		// Overwriting is safe only when the file holds nothing the user owns:
		// it is blank, or it is byte-identical to what qsdev last wrote.
		disposable := len(bytes.TrimSpace(onDisk)) == 0 || (recorded && state.ComputeHash(onDisk) == rec.Hash)
		switch {
		case disposable:
			_, _ = fmt.Fprintf(warn, "Warning: merge failed for %s: %v (no user content to preserve; overwriting)\n", f.Path, mergeErr)
		case opts.force:
			_, _ = fmt.Fprintf(warn, "Warning: merge failed for %s: %v (overwriting: --force)\n", f.Path, mergeErr)
		default:
			return fmt.Sprintf("%s (%v)", f.Path, mergeErr)
		}
		plan.write(f, true)
		return ""
	}

	plan.mergedOriginals[f.Path] = f.Content // remember "ours" for BaseContent
	f.Content = merged
	if bytes.Equal(onDisk, merged) {
		// Merge is a no-op on disk, but still record the file so its
		// BaseContent tracks "ours"; without this a later three-way merge can
		// never prune managed keys the generator removed.
		plan.keep(f)
		return ""
	}
	plan.write(f, true)
	return ""
}

// keep records a file whose on-disk content already matches.
func (p *regenPlan) keep(f types.GeneratedFile) {
	p.recorded = append(p.recorded, f)
	p.present[f.Path] = true
	p.unchanged++
}

// write schedules a file to be written.
func (p *regenPlan) write(f types.GeneratedFile, exists bool) {
	p.writes = append(p.writes, f)
	p.recorded = append(p.recorded, f)
	p.present[f.Path] = true
	if exists {
		p.updated++
	} else {
		p.created++
	}
}

// applyRegenPlan performs the planned legacy cleanup and file writes.
func applyRegenPlan(plan *regenPlan, projectRoot string, warn io.Writer) error {
	for _, legacy := range plan.legacyRemovals {
		if err := os.Remove(filepath.Join(projectRoot, legacy)); err != nil && !errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(warn, "Warning: could not remove stale skill file %s: %v\n", legacy, err)
		}
	}
	for _, f := range plan.writes {
		if err := fileutil.WriteFileAtomic(filepath.Join(projectRoot, f.Path), f.Content, f.Mode); err != nil {
			return fmt.Errorf("writing %s: %w", f.Path, err)
		}
	}
	return nil
}

// planLegacySkillCleanup finds stale flat .claude/skills/<name>.md files left
// over from the pre-<name>/SKILL.md layout. A flat file is deleted only when
// its content matches what qsdev recorded generating (in the claude state or
// the top-level devinit state); anything else may be user-authored and is left
// in place with a warning. It also drops the legacy entries from existing so
// they are not carried forward.
func planLegacySkillCleanup(files []types.GeneratedFile, existing types.GeneratedState, projectRoot string, warn io.Writer) []string {
	var removals []string
	var initState *types.GeneratedState
	for _, f := range files {
		legacy, ok := legacyFlatSkillPath(f.Path)
		if !ok {
			continue
		}
		onDisk, exists, err := readIfExists(filepath.Join(projectRoot, legacy))
		switch {
		case err != nil:
			_, _ = fmt.Fprintf(warn, "Warning: could not read stale skill file %s: %v\n", legacy, err)
			continue
		case !exists:
			delete(existing.Files, legacy)
			continue
		}
		if initState == nil {
			initState = loadInitState(projectRoot)
		}
		hash := state.ComputeHash(onDisk)
		if recordedHash(existing, legacy) == hash || recordedHash(*initState, legacy) == hash {
			removals = append(removals, legacy)
			delete(existing.Files, legacy)
			continue
		}
		_, _ = fmt.Fprintf(warn, "Warning: %s uses the old flat skill layout and was not generated by %s (or was edited); leaving it in place. Claude Code loads %s instead; move any custom content there and delete %s.\n",
			legacy, branding.Get().AppName, f.Path, legacy)
	}
	return removals
}

// loadInitState loads the top-level (devinit) state, which records files
// generated by a top-level init. A missing or unreadable file yields an empty
// state: the caller then treats every legacy file as unrecorded, which only
// ever preserves files.
func loadInitState(projectRoot string) *types.GeneratedState {
	st, err := state.LoadStateFromFile(filepath.Join(projectRoot, state.StateFilePaths()[0]))
	if err != nil {
		st = types.GeneratedState{}
	}
	return &st
}

// recordedHash returns the hash recorded for path, or "" when unrecorded.
func recordedHash(st types.GeneratedState, path string) string {
	if fs, ok := st.Files[path]; ok {
		return fs.Hash
	}
	return ""
}

// readIfExists reads path, reporting whether it exists. Errors other than
// not-exist are returned.
func readIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("reading %s: %w", path, err)
}

// modeMatches reports whether path's permission bits equal mode. Windows does
// not carry POSIX permission bits, so it always matches there.
func modeMatches(path string, mode os.FileMode) bool {
	if runtime.GOOS == "windows" {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm() == mode.Perm()
}

func isMergeable(s types.MergeStrategy) bool {
	return s == types.ThreeWayMerge || s == types.SectionMarker
}

func isHookScript(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), hookScriptDir)
}

// alwaysRestored reports whether a user-changed file is restored even when
// user changes are otherwise kept: hook scripts are security guards, and an
// edited library-managed file is owned by the skill library. A deleted
// library-managed file stays deleted.
func alwaysRestored(f types.GeneratedFile, exists bool) bool {
	return isHookScript(f.Path) || (exists && f.Strategy == types.LibraryManaged)
}
