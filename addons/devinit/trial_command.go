package devinit

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	cmd.Flags().StringVarP(&opts.Branch, "branch", "b", "qsdev-trial", "Branch name for the trial worktree")
	cmd.Flags().StringVarP(&opts.Path, "path", "p", "", "Worktree path (default: ../<repo>-qsdev-trial)")
	cmd.Flags().StringVar(&opts.Profile, "profile", "", "Project-type profile to apply")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Show what would be done without creating the worktree")

	return cmd
}

func runTrial(cmd *cobra.Command, opts TrialOptions) error {
	projectRoot, err := cmdutil.ProjectRoot()
	if err != nil {
		return err
	}

	if !isGitRepo(projectRoot) {
		return fmt.Errorf("not a git repository; %s trial requires git", branding.Get().AppName)
	}

	repoName := filepath.Base(projectRoot)
	worktreePath := opts.Path
	if worktreePath == "" {
		worktreePath = filepath.Join(filepath.Dir(projectRoot), repoName+"-qsdev-trial")
	}
	if !filepath.IsAbs(worktreePath) {
		worktreePath = filepath.Join(projectRoot, worktreePath)
	}

	if opts.DryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Would create:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Worktree: %s\n", worktreePath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Branch:   %s\n", opts.Branch)
		fmt.Fprintf(cmd.OutOrStdout(), "  Action:   %s init --yes --force\n", branding.Get().AppName)
		return nil
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("path already exists: %s\nRemove it or use --path to specify a different location", worktreePath)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Creating worktree at %s (branch: %s)...\n", worktreePath, opts.Branch)
	if err := runGit(projectRoot, "worktree", "add", "-b", opts.Branch, worktreePath); err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	initArgs := []string{"init", "--yes", "--force"}
	if opts.Profile != "" {
		initArgs = append(initArgs, "--profile", opts.Profile)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Running %s init in worktree...\n", branding.Get().AppName)
	if err := runSelfInDir(worktreePath, initArgs...); err != nil {
		return fmt.Errorf("init in worktree failed: %w", err)
	}

	// Commit generated files so Nix flakes can evaluate them (flakes only see git-tracked files).
	fmt.Fprintf(cmd.OutOrStdout(), "Committing generated configuration...\n")
	if err := runGit(worktreePath, "add", "."); err != nil {
		return fmt.Errorf("staging generated files: %w", err)
	}
	if err := runGit(worktreePath, "commit", "-m", branding.Get().AppName+" trial: generated configuration"); err != nil {
		return fmt.Errorf("committing generated files: %w", err)
	}

	appName := branding.Get().AppName
	fmt.Fprintf(cmd.OutOrStdout(), "\nTrial environment created successfully.\n\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  cd %s\n\n", worktreePath)
	fmt.Fprintf(cmd.OutOrStdout(), "Evaluate the configuration, then:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  Keep it:    git checkout main && git merge %s\n", opts.Branch)
	fmt.Fprintf(cmd.OutOrStdout(), "  Discard it: git worktree remove %s && git branch -D %s\n\n", worktreePath, opts.Branch)
	fmt.Fprintf(cmd.OutOrStdout(), "Run '%s status' in the worktree to see your security posture.\n", appName)
	return nil
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

func runGit(dir string, args ...string) error {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runSelfInDir(dir string, args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}
	c := exec.Command(exe, args...)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
