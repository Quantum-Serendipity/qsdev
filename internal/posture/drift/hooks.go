package drift

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const categoryHookDrift = "Pre-Commit Hook Drift"

// gitHooksDirTimeout bounds the git invocation used to locate the hooks
// directory, so a wedged git never stalls a posture assessment.
const gitHooksDirTimeout = 5 * time.Second

// detectHookDrift checks that git hooks are installed and properly configured.
// The hooks directory is resolved the way git resolves it, so worktrees,
// submodules (whose .git is a "gitdir:" file) and core.hooksPath are honoured.
func detectHookDrift(projectDir string, enabledTools map[string]bool) Category {
	cat := Category{Name: categoryHookDrift}

	if _, err := os.Stat(filepath.Join(projectDir, ".git")); err != nil {
		if os.IsNotExist(err) {
			cat.Findings = append(cat.Findings, Finding{
				Category:    categoryHookDrift,
				Severity:    Info,
				Subject:     ".git",
				Description: "Project is not a git repository",
			})
		} else {
			cat.Findings = append(cat.Findings, Finding{
				Category:    categoryHookDrift,
				Severity:    Warning,
				Subject:     ".git",
				Description: fmt.Sprintf("Unable to inspect .git: %v", err),
			})
		}
		return cat
	}

	hooksDir, err := resolveHooksDir(projectDir)
	if err != nil {
		cat.Findings = append(cat.Findings, Finding{
			Category:    categoryHookDrift,
			Severity:    Warning,
			Subject:     ".git",
			Description: fmt.Sprintf("Unable to locate the git hooks directory: %v", err),
		})
		return cat
	}

	// Check pre-commit hook.
	if f, ok := checkHook(hooksDir, "pre-commit",
		"Git pre-commit hook is not installed",
		"Run qsdev update to install the pre-commit hook"); !ok {
		cat.Findings = append(cat.Findings, f)
	}

	// If commitlint is enabled, check commit-msg hook.
	if enabledTools["commitlint"] {
		if f, ok := checkHook(hooksDir, "commit-msg",
			"Commitlint is enabled but the commit-msg hook is not installed",
			"Run qsdev update to install the commit-msg hook"); !ok {
			cat.Findings = append(cat.Findings, f)
		}
	}

	return cat
}

// checkHook verifies that the named hook exists in hooksDir and, off Windows,
// is executable. It returns ok=false with the finding to report otherwise. A
// stat failure other than not-exist is reported rather than dropped, so an
// unreadable hooks directory never reads as a clean result.
func checkHook(hooksDir, name, missingDesc, missingRemediation string) (Finding, bool) {
	hookPath := filepath.Join(hooksDir, name)
	info, err := os.Stat(hookPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return Finding{
			Category:    categoryHookDrift,
			Severity:    Warning,
			Subject:     name,
			Description: missingDesc,
			Remediation: missingRemediation,
		}, false
	case err != nil:
		return Finding{
			Category:    categoryHookDrift,
			Severity:    Warning,
			Subject:     name,
			Description: fmt.Sprintf("Unable to inspect the git %s hook: %v", name, err),
			Remediation: missingRemediation,
		}, false
	case runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0:
		return Finding{
			Category:    categoryHookDrift,
			Severity:    Warning,
			Subject:     name,
			Description: fmt.Sprintf("Git %s hook exists but is not executable", name),
			Remediation: fmt.Sprintf("Run: chmod +x %s", hookPath),
		}, false
	}
	return Finding{}, true
}

// resolveHooksDir returns the directory git runs hooks from for the working
// tree at projectDir. It asks git itself, which follows worktree and submodule
// gitdir links and honours core.hooksPath; when git is unavailable or does not
// recognise the repository it falls back to reading the .git link directly.
func resolveHooksDir(projectDir string) (string, error) {
	if dir, err := gitHooksDir(projectDir); err == nil {
		return dir, nil
	}
	return hooksDirFromDotGit(projectDir)
}

// gitHooksDir asks git for the hooks path of the repository rooted exactly at
// projectDir. GIT_CEILING_DIRECTORIES stops git from walking up into an
// unrelated enclosing repository when projectDir's own .git is unusable.
func gitHooksDir(projectDir string) (string, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("resolving project dir: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitHooksDirTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", absDir,
		"rev-parse", "--path-format=absolute", "--git-path", "hooks")
	cmd.Env = append(repoScopedEnv(os.Environ()), "GIT_CEILING_DIRECTORIES="+filepath.Dir(absDir))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-path hooks: %w", err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" || !filepath.IsAbs(filepath.FromSlash(dir)) {
		return "", fmt.Errorf("git returned unusable hooks path %q", dir)
	}
	return filepath.Clean(filepath.FromSlash(dir)), nil
}

// repoScopedEnv drops the variables that pin git to a particular repository
// (set, for example, while qsdev runs inside a git hook), so git discovers
// the repository from the -C directory instead.
func repoScopedEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		switch name {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE":
			continue
		}
		out = append(out, kv)
	}
	return out
}

// hooksDirFromDotGit resolves the hooks directory from projectDir/.git
// without git: a directory holds hooks/ directly, while a worktree's or
// submodule's "gitdir: <path>" file points at its git dir, whose commondir
// (for worktrees) holds the shared hooks/.
func hooksDirFromDotGit(projectDir string) (string, error) {
	dotGit := filepath.Join(projectDir, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return "", fmt.Errorf("inspecting .git: %w", err)
	}
	if info.IsDir() {
		return filepath.Join(dotGit, "hooks"), nil
	}

	data, err := os.ReadFile(dotGit)
	if err != nil {
		return "", fmt.Errorf("reading .git file: %w", err)
	}
	content := strings.TrimSpace(string(data))
	gitDir, ok := strings.CutPrefix(content, "gitdir:")
	if !ok {
		return "", fmt.Errorf(".git file has no gitdir: line")
	}
	gitDir = filepath.FromSlash(strings.TrimSpace(gitDir))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(projectDir, gitDir)
	}

	commonDir := gitDir
	if raw, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		cd := filepath.FromSlash(strings.TrimSpace(string(raw)))
		if !filepath.IsAbs(cd) {
			cd = filepath.Join(gitDir, cd)
		}
		commonDir = cd
	}
	return filepath.Join(commonDir, "hooks"), nil
}
