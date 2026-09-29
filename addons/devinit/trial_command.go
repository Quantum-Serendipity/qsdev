package devinit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TrialOptions holds the flags for the trial command.
type TrialOptions struct {
	Branch  string
	Path    string
	Profile string
	DryRun  bool
}

func trialCmd() *cobra.Command {
	var opts TrialOptions

	cmd := &cobra.Command{
		Use:   "trial",
		Short: "Create a worktree to safely evaluate " + branding.Get().AppName + " on this project",
		Long: `Creates a git worktree with a full ` + branding.Get().AppName + ` configuration so you can
evaluate the generated environment without modifying your working branch.

Happy with it? Merge the branch. Not for you? Delete the worktree.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTrial(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", branding.Get().AppName+"-trial", "Branch name for the trial worktree")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Worktree path (default: ../<repo>-"+branding.Get().AppName+"-trial)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "Project-type profile to apply")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be done without creating the worktree")

	return cmd
}

var validBranchRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]*$`)

func validateBranchName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("branch name cannot be empty")
	}
	if len(name) > 250 {
		return fmt.Errorf("branch name too long")
	}
	if !validBranchRe.MatchString(name) {
		return fmt.Errorf("invalid branch name %q", name)
	}
	if strings.Contains(name, "..") || strings.Contains(name, "~") ||
		strings.Contains(name, "^") || strings.Contains(name, ":") ||
		strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("invalid branch name %q: contains forbidden sequence", name)
	}
	return nil
}

func runTrial(cmd *cobra.Command, opts TrialOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	if !isGitRepo(projectRoot) {
		return fmt.Errorf("not a git repository; %s trial requires git", branding.Get().AppName)
	}

	// Validate before the dry-run so a preview never reports success for a
	// branch name the real run would reject.
	if err := validateBranchName(opts.Branch); err != nil {
		return fmt.Errorf("validating branch name: %w", err)
	}

	repoName := filepath.Base(projectRoot)
	worktreePath := opts.Path
	if worktreePath == "" {
		worktreePath = filepath.Join(filepath.Dir(projectRoot), repoName+"-"+branding.Get().AppName+"-trial")
	}
	if !filepath.IsAbs(worktreePath) {
		worktreePath = filepath.Join(projectRoot, worktreePath)
	}

	out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()

	if opts.DryRun {
		fmt.Fprintf(out, "Would create:\n")
		fmt.Fprintf(out, "  Worktree: %s\n", worktreePath)
		fmt.Fprintf(out, "  Branch:   %s\n", opts.Branch)
		fmt.Fprintf(out, "  Action:   %s init --yes --force\n", branding.Get().AppName)
		return nil
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("path already exists: %s\nRemove it or use --path to specify a different location", worktreePath)
	}

	// The branch the trial forks from is where "keep it" merges back into.
	baseBranch := currentBranch(projectRoot)

	fmt.Fprintf(out, "Creating worktree at %s (branch: %s)...\n", worktreePath, opts.Branch)
	if err := runGit(projectRoot, out, errOut, "worktree", "add", "-b", opts.Branch, worktreePath); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	if err := populateTrialWorktree(out, errOut, worktreePath, opts); err != nil {
		// Leave nothing behind: a stale worktree and branch would make every
		// retry fail with "path already exists".
		if cleanupErr := removeTrialWorktree(projectRoot, worktreePath, opts.Branch); cleanupErr != nil {
			return fmt.Errorf("%w (cleanup also failed: %w; remove %s and branch %s manually)",
				err, cleanupErr, worktreePath, opts.Branch)
		}
		return err
	}

	appName := branding.Get().AppName
	fmt.Fprintf(out, "\nTrial environment created successfully.\n\n")
	fmt.Fprintf(out, "  cd %s\n\n", worktreePath)
	fmt.Fprintf(out, "Evaluate the configuration, then:\n")
	if baseBranch != "" {
		fmt.Fprintf(out, "  Keep it:    git checkout %s && git merge %s\n", baseBranch, opts.Branch)
	} else {
		fmt.Fprintf(out, "  Keep it:    git merge %s  (from the branch that should receive it)\n", opts.Branch)
	}
	fmt.Fprintf(out, "  Discard it: git worktree remove %s && git branch -D %s\n\n", worktreePath, opts.Branch)
	fmt.Fprintf(out, "Run '%s status' in the worktree to see your security posture.\n", appName)
	return nil
}

// populateTrialWorktree runs init inside the freshly created worktree and
// commits the generated configuration.
func populateTrialWorktree(out, errOut io.Writer, worktreePath string, opts TrialOptions) error {
	initArgs := []string{"init", "--yes", "--force"}
	if opts.Profile != "" {
		initArgs = append(initArgs, "--profile", opts.Profile)
	}
	fmt.Fprintf(out, "Running %s init in worktree...\n", branding.Get().AppName)
	if err := trialInitRunner(worktreePath, out, errOut, initArgs...); err != nil {
		return fmt.Errorf("init in worktree failed: %w", err)
	}

	// Commit generated files so Nix flakes can evaluate them (flakes only see git-tracked files).
	fmt.Fprintf(out, "Committing generated configuration...\n")
	if err := runGit(worktreePath, out, errOut, "add", "."); err != nil {
		return fmt.Errorf("staging generated files: %w", err)
	}
	if err := runGit(worktreePath, out, errOut, "commit", "-m", branding.Get().AppName+" trial: generated configuration"); err != nil {
		return fmt.Errorf("committing generated files: %w", err)
	}
	return nil
}

// removeTrialWorktree best-effort removes a trial worktree and its branch
// after a failed trial, returning the combined cleanup errors.
func removeTrialWorktree(projectRoot, worktreePath, branch string) error {
	var errs []error
	if err := runGit(projectRoot, io.Discard, io.Discard, "worktree", "remove", "--force", worktreePath); err != nil {
		errs = append(errs, fmt.Errorf("removing worktree: %w", err))
	}
	if err := runGit(projectRoot, io.Discard, io.Discard, "branch", "-D", branch); err != nil {
		errs = append(errs, fmt.Errorf("deleting branch: %w", err))
	}
	return errors.Join(errs...)
}

// currentBranch returns the checked-out branch of the repository at dir, or
// "" when HEAD is detached or the branch cannot be determined.
func currentBranch(dir string) string {
	c := exec.Command("git", "symbolic-ref", "--quiet", "--short", "HEAD")
	c.Dir = dir
	out, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func isGitRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// .git can be a directory (normal repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}

func runGit(dir string, stdout, stderr io.Writer, args ...string) error {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}

// trialInitRunner runs the init step inside the trial worktree. It is a
// variable so tests can substitute it: re-executing the test binary with
// init arguments would run the whole test suite instead.
var trialInitRunner = runSelfInDir

func runSelfInDir(dir string, stdout, stderr io.Writer, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	c := exec.Command(exe, args...)
	c.Dir = dir
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}
