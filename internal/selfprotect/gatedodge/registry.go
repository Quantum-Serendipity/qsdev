package gatedodge

import (
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
)

// GuardedFile is a file whose content guards the project's security settings.
// A Write/Edit/MultiEdit of it is checked by Introduced, Change or both; a
// shell command that rewrites it skips those checks, so selfprotect denies
// it.
type GuardedFile struct {
	// Name is the file's lower-cased base name; it is matched in any case.
	Name string
	// RuleID is the gate-dodge rule ID reported when a change is blocked.
	RuleID string
	// Introduced, when set, describes how the text a call introduces (the
	// Write content or an Edit's new_string) weakens the file ("" when it
	// does not).
	Introduced func(content string) string
	// Change, when set, describes how changing the whole file from before to
	// after weakens it ("" when it does not). It sees what a call removes,
	// which Introduced cannot. An error means the change cannot be verified;
	// callers must block it.
	Change func(before, after string) (string, error)
}

// guardedFiles is the one list of content-guarded files. canon keeps its
// GuardedConfigFiles list for the hook sandbox; every name on it has a Change
// check here (TestGuardedFileRegistryConsistency).
var guardedFiles = []GuardedFile{
	{Name: ".qsdev.yaml", RuleID: "GD-001", Change: qsdevConfigDowngrade},
	{Name: "devenv.nix", RuleID: "GD-002", Change: devenvNixDowngrade},
	{Name: ".pre-commit-config.yaml", RuleID: "GD-003", Introduced: detectPrecommitConfig, Change: checkPrecommitResult},
	{Name: ".npmrc", RuleID: "GD-004", Introduced: detectNpmrc, Change: checkNpmrcResult},
	{Name: "pnpm-workspace.yaml", RuleID: "GD-004", Change: pnpmWorkspaceConfig.check},
	{Name: ".yarnrc.yml", RuleID: "GD-004", Change: yarnBerryConfig.check},
	{Name: ".yarnrc", RuleID: "GD-004", Change: yarnClassicConfig.check},
	{Name: "bunfig.toml", RuleID: "GD-004", Change: bunConfig.check},
	{Name: "claude.md", RuleID: "GD-005", Introduced: detectClaudeMD},
}

// Lookup returns the distinct entries guarding paths, in the order of paths:
// those whose Name is the lower-cased base name of a non-empty path. A Write
// hook passes the resolved target first and then the path the call names, so
// the checks of the file actually written run (claude.md linked to
// .qsdev.yaml gets GD-001) and a guarded name linked elsewhere stays guarded.
// The case is ignored because on a case-insensitive filesystem (macOS,
// Windows) .NPMRC opens .npmrc, and so are the name aliases Windows strips
// when it opens a file (a trailing dot or space, an alternate-data-stream
// suffix such as `::$DATA`), so `.qsdev.yaml.` names .qsdev.yaml. Both / and
// \ separate path components on every OS, so a Windows path is matched
// wherever it is checked.
func Lookup(paths ...string) []GuardedFile {
	var found []GuardedFile
	for _, p := range paths {
		if p == "" {
			continue
		}
		name := lookupKey.Key(p[strings.LastIndexAny(p, `/\`)+1:])
		i := slices.IndexFunc(guardedFiles, func(g GuardedFile) bool { return g.Name == name })
		if i < 0 || slices.ContainsFunc(found, func(g GuardedFile) bool { return g.Name == name }) {
			continue
		}
		found = append(found, guardedFiles[i])
	}
	return found
}

// lookupKey compares base names as a case-insensitive Windows filesystem
// does, on every OS (see Lookup).
var lookupKey = pathmatch.Options{FoldCase: true, WindowsAliases: true}

// ChangeGuardedNames returns the sorted names of the guarded files that have
// a Change check: the files whose shell rewrite would skip a before/after
// comparison.
func ChangeGuardedNames() []string {
	var names []string
	for _, g := range guardedFiles {
		if g.Change != nil {
			names = append(names, g.Name)
		}
	}
	slices.Sort(names)
	return names
}
