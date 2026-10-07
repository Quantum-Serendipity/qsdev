package cigeneration

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// allowedActionOwners mirrors the repository setting Settings > Actions >
// General > "Allow select actions and reusable workflows": actions created by
// GitHub (the actions and github organizations) plus the owner/* patterns
// listed there. GitHub refuses to start a workflow that uses any other action,
// reporting only startup_failure with no log, so a workflow change that adds
// one fails here instead. Change this list only together with that setting.
var allowedActionOwners = []string{
	// Actions created by GitHub.
	"actions",
	"github",
	// Allowed owner/* patterns.
	"anchore",
	"crazy-max",
	"golangci",
	"google",
	"goreleaser",
	"ossf",
	"sigstore",
	"slsa-framework",
	"snyk",
	"softprops",
	"step-security",
}

// actionWorkflow is the part of a workflow that names actions: each job's
// reusable-workflow uses and its steps' uses.
type actionWorkflow struct {
	Jobs map[string]struct {
		Uses  string `yaml:"uses"`
		Steps []struct {
			Name string `yaml:"name"`
			Uses string `yaml:"uses"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

// actionAllowed reports whether a uses: reference may run under the
// repository's Actions allowlist. Local actions and reusable workflows
// (./...) are always allowed; anything else must name an allowed owner.
func actionAllowed(uses string) bool {
	if strings.HasPrefix(uses, "./") {
		return true
	}
	if strings.HasPrefix(uses, "docker://") {
		return false
	}
	owner, _, ok := strings.Cut(uses, "/")
	return ok && slices.Contains(allowedActionOwners, strings.ToLower(owner))
}

// checkWorkflowActions parses one workflow and returns every uses: outside
// the allowlist, and a parse error as a problem of its own.
func checkWorkflowActions(src []byte) []string {
	var wf actionWorkflow
	if err := yaml.Unmarshal(src, &wf); err != nil {
		return []string{fmt.Sprintf("parsing workflow: %v", err)}
	}
	if len(wf.Jobs) == 0 {
		return []string{"has no jobs; the parser or the layout changed"}
	}
	var problems []string
	for name, job := range wf.Jobs {
		if job.Uses != "" && !actionAllowed(job.Uses) {
			problems = append(problems, fmt.Sprintf("job %s uses %s, which the Actions allowlist does not admit", name, job.Uses))
		}
		for i, s := range job.Steps {
			if s.Uses != "" && !actionAllowed(s.Uses) {
				problems = append(problems, fmt.Sprintf("job %s step %d (%s) uses %s, which the Actions allowlist does not admit",
					name, i+1, s.Name, s.Uses))
			}
		}
	}
	slices.Sort(problems)
	return problems
}

// TestWorkflowActionsAllowlisted keeps every workflow startable. Wave 1 added
// cachix/install-nix-action, which the repository's allowlist does not admit,
// and every CI run ended in startup_failure before any job ran. Each file in
// .github/workflows must also parse as YAML.
func TestWorkflowActionsAllowlisted(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(repoWorkflowsDir)
	if err != nil {
		t.Fatalf("reading %s: %v", repoWorkflowsDir, err)
	}
	checked := 0
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if e.IsDir() || (ext != ".yml" && ext != ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(repoWorkflowsDir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		checked++
		for _, p := range checkWorkflowActions(b) {
			t.Errorf(".github/workflows/%s: %s", e.Name(), p)
		}
	}
	if checked == 0 {
		t.Fatal("found no workflows; the layout changed")
	}
}

// TestCheckWorkflowActions proves the guard above is not vacuous.
func TestCheckWorkflowActions(t *testing.T) {
	t.Parallel()

	const sha = "@0123456789abcdef0123456789abcdef01234567"
	tests := []struct {
		name string
		src  string
		want int
	}{
		{"allowed owners", "jobs:\n  a:\n    steps:\n      - uses: actions/checkout" + sha + "\n      - uses: github/codeql-action/init" + sha +
			"\n      - uses: step-security/harden-runner" + sha + "\n      - run: echo hi\n", 0},
		{"local action and reusable workflow", "jobs:\n  a:\n    uses: ./.github/workflows/x.yml\n  b:\n    steps:\n      - uses: ./.github/actions/y\n", 0},
		{"third-party step action", "jobs:\n  a:\n    steps:\n      - uses: cachix/install-nix-action" + sha + "\n", 1},
		{"third-party reusable workflow", "jobs:\n  a:\n    uses: example/workflows/.github/workflows/ci.yml" + sha + "\n", 1},
		{"owner prefix is not an owner", "jobs:\n  a:\n    steps:\n      - uses: actions-evil/checkout" + sha + "\n", 1},
		{"docker image", "jobs:\n  a:\n    steps:\n      - uses: docker://alpine:3\n", 1},
		{"unparsable", "jobs: [\n", 1},
		{"no jobs", "name: x\n", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := checkWorkflowActions([]byte(tt.src)); len(got) != tt.want {
				t.Errorf("checkWorkflowActions() = %q, want %d problems", got, tt.want)
			}
		})
	}
}
