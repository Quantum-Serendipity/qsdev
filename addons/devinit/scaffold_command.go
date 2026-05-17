package devinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// ScaffoldOptions holds the flags for the scaffold-instance command.
type ScaffoldOptions struct {
	GitHubOwner string
	GitHubRepo  string
	OutputDir   string
	Module      string
}

// ScaffoldData is the template data passed to scaffold templates.
type ScaffoldData struct {
	AppName      string
	AppNameUpper string
	Module       string
	GitHubOwner  string
	GitHubRepo   string
}

var validAppName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func scaffoldCmd() *cobra.Command {
	var opts ScaffoldOptions

	cmd := &cobra.Command{
		Use:   "scaffold-instance <appname>",
		Short: "Generate a new branded dev tool built on the " + branding.Get().AppName + " framework",
		Long: `Scaffolds a complete Go project for a custom branded *dev tool.

The generated project imports ` + branding.Get().AppName + ` as a framework and includes:
  • cmd/<appname>/main.go with your branding
  • go.mod with correct module path
  • Makefile with build/test/lint targets
  • .goreleaser.yaml for cross-platform releases
  • README.md starter
  • .gitignore`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScaffold(cmd, args[0], opts)
		},
	}

	cmd.Flags().StringVarP(&opts.GitHubOwner, "github-owner", "o", "", "GitHub organization or user (required)")
	cmd.Flags().StringVarP(&opts.GitHubRepo, "github-repo", "r", "", "GitHub repository name (defaults to appname)")
	cmd.Flags().StringVarP(&opts.OutputDir, "output-dir", "d", "", "Output directory (defaults to ./<appname>)")
	cmd.Flags().StringVar(&opts.Module, "module", "", "Go module path (defaults to github.com/<owner>/<repo>)")

	return cmd
}

func runScaffold(cmd *cobra.Command, appName string, opts ScaffoldOptions) error {
	if !validAppName.MatchString(appName) {
		return fmt.Errorf("invalid app name %q: must start with a lowercase letter and contain only lowercase letters, digits, and hyphens", appName)
	}

	if opts.GitHubOwner == "" {
		return fmt.Errorf("--github-owner is required")
	}

	if opts.GitHubRepo == "" {
		opts.GitHubRepo = appName
	}
	if opts.OutputDir == "" {
		opts.OutputDir = appName
	}
	if opts.Module == "" {
		opts.Module = fmt.Sprintf("github.com/%s/%s", opts.GitHubOwner, opts.GitHubRepo)
	}

	if _, err := os.Stat(opts.OutputDir); err == nil {
		return fmt.Errorf("output directory already exists: %s", opts.OutputDir)
	}

	data := ScaffoldData{
		AppName:      appName,
		AppNameUpper: strings.ToUpper(strings.ReplaceAll(appName, "-", "_")),
		Module:       opts.Module,
		GitHubOwner:  opts.GitHubOwner,
		GitHubRepo:   opts.GitHubRepo,
	}

	files := []struct {
		path    string
		tmpl    string
		mode    os.FileMode
	}{
		{filepath.Join("cmd", appName, "main.go"), scaffoldMainGoTmpl, 0o644},
		{"go.mod", scaffoldGoModTmpl, 0o644},
		{"Makefile", scaffoldMakefileTmpl, 0o644},
		{".goreleaser.yaml", scaffoldGoreleaserTmpl, 0o644},
		{"README.md", scaffoldReadmeTmpl, 0o644},
		{".gitignore", scaffoldGitignoreTmpl, 0o644},
	}

	for _, f := range files {
		content, err := renderTemplate(f.tmpl, data)
		if err != nil {
			return fmt.Errorf("rendering %s: %w", f.path, err)
		}
		fullPath := filepath.Join(opts.OutputDir, f.path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", f.path, err)
		}
		if err := fileutil.WriteFileAtomic(fullPath, content, f.mode); err != nil {
			return fmt.Errorf("writing %s: %w", f.path, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Scaffolded %s at %s/\n\n", appName, opts.OutputDir)
	fmt.Fprintf(cmd.OutOrStdout(), "Next steps:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  cd %s\n", opts.OutputDir)
	fmt.Fprintf(cmd.OutOrStdout(), "  go mod tidy\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  go build ./cmd/%s\n", appName)
	fmt.Fprintf(cmd.OutOrStdout(), "  ./%s --help\n", appName)
	return nil
}

func renderTemplate(tmplStr string, data ScaffoldData) ([]byte, error) {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
