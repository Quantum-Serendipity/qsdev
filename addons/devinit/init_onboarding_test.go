package devinit

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/generate"
)

// tipRe extracts the next-tier preview command init prints.
var tipRe = regexp.MustCompile(`Tip: Run '` + regexp.QuoteMeta(branding.Get().AppName) + ` ([^']+)' to preview the next tier`)

// TestInitCmd_NextTierTipPreviews is the regression test for the next-tier
// tip being a no-op: on a set-up project `init --tier X --dry-run` printed
// "Nothing to do" and previewed nothing. The advertised command must actually
// print a preview, and the forced path must not claim "Nothing to do".
func TestInitCmd_NextTierTipPreviews(t *testing.T) {
	dir := createGoFixture(t)
	out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "supply-chain-only")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	m := tipRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no next-tier tip in output:\n%s", out)
	}
	before := readProjectFile(t, dir, "devenv.nix")

	out, err = executeInitCmd(t, dir, strings.Fields(m[1])...)
	if err != nil {
		t.Fatalf("tip command %q: %v\n%s", m[1], err, out)
	}
	if strings.Contains(out, "Nothing to do") {
		t.Errorf("forced re-init still announces \"Nothing to do\":\n%s", out)
	}
	if !strings.Contains(out, "--force regenerates") || !strings.Contains(out, "Action") {
		t.Errorf("tip command did not explain the forced path and print a preview:\n%s", out)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != before {
		t.Error("the --dry-run tip command modified devenv.nix")
	}
}

// TestInitCmd_IgnoredFlagsNoteNamesWorkingCommand checks the note for flags a
// set-up project ignores names a command that works without a terminal.
func TestInitCmd_IgnoredFlagsNoteNamesWorkingCommand(t *testing.T) {
	dir := createGoFixture(t)
	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	out, err := executeInitCmd(t, dir, "--tier", "full", "--dry-run")
	if err != nil {
		t.Fatalf("re-run: %v\n%s", err, out)
	}
	for _, want := range []string{"ignoring --tier", "nothing was generated or previewed", "--yes --force"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// TestInitCmd_ReviewCLAUDEmdOnlyWhenGenerated is the regression test for the
// supply-chain-only next steps telling the user to review a CLAUDE.md that
// tier never generates.
func TestInitCmd_ReviewCLAUDEmdOnlyWhenGenerated(t *testing.T) {
	tests := []struct {
		tier       string
		wantReview bool
	}{
		{"supply-chain-only", false},
		{"standard", true},
	}
	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			dir := createGoFixture(t)
			out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", tt.tier)
			if err != nil {
				t.Fatalf("init: %v\n%s", err, out)
			}
			_, statErr := os.Stat(filepath.Join(dir, claudeMDPath))
			if exists := statErr == nil; exists != tt.wantReview {
				t.Fatalf("CLAUDE.md exists = %v, want %v", exists, tt.wantReview)
			}
			if got := strings.Contains(out, "Review CLAUDE.md"); got != tt.wantReview {
				t.Errorf("\"Review CLAUDE.md\" step shown = %v, want %v:\n%s", got, tt.wantReview, out)
			}
		})
	}
}

// createHandWrittenConfigFixture is a Go project that already has its own
// CLAUDE.md and devenv.nix, as when onboarding an existing repository.
func createHandWrittenConfigFixture(t *testing.T) (dir, claudeMD, devenvNix string) {
	t.Helper()
	dir = createGoFixture(t)
	claudeMD = "# Team notes\n\nHand-written guidance.\n"
	devenvNix = "{ pkgs, ... }: { packages = [ pkgs.hello ]; }\n"
	if err := os.WriteFile(filepath.Join(dir, claudeMDPath), []byte(claudeMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "devenv.nix"), []byte(devenvNix), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, claudeMD, devenvNix
}

// TestInitCmd_MergeOnboardsExistingConfig covers the non-destructive
// onboarding path the qsdev-onboard skill documents (`init --yes --merge`),
// which previously did not exist: --yes refused existing configuration and
// --force overwrote devenv.nix.
func TestInitCmd_MergeOnboardsExistingConfig(t *testing.T) {
	dir, claudeMD, devenvNix := createHandWrittenConfigFixture(t)

	_, err := executeInitCmd(t, dir, "--yes", "--lang", "go")
	if err == nil || !strings.Contains(err.Error(), "--merge") {
		t.Fatalf("refusal should point at --merge, got %v", err)
	}

	out, err := executeInitCmd(t, dir, "--yes", "--merge", "--lang", "go")
	if err != nil {
		t.Fatalf("init --merge: %v\n%s", err, out)
	}
	if got := readProjectFile(t, dir, claudeMDPath); !strings.HasPrefix(got, claudeMD) || !strings.Contains(got, merge.BeginMarkerPrefix) {
		t.Errorf("CLAUDE.md should keep the hand-written text and gain the generated block:\n%s", got)
	}
	if got := readProjectFile(t, dir, "devenv.nix"); got != devenvNix {
		t.Errorf("--merge overwrote devenv.nix:\n%s", got)
	}
	requireFileExists(t, dir, "devenv.nix"+generate.SidecarSuffix)
	requireFileExists(t, dir, branding.Get().ConfigFile)

	if _, err := executeInitCmd(t, dir, "--yes", "--merge", "--force"); err == nil {
		t.Error("--merge and --force together should be rejected")
	}
}

// TestInitCmd_PartialWriteLeavesRecoverableProject is the regression test for
// a partial init write returning before .gitignore and the answers were
// written: the state directories were left unignored and repair had no
// inputs. The error now names the re-run that finishes the setup, and it
// does.
func TestInitCmd_PartialWriteLeavesRecoverableProject(t *testing.T) {
	dir := createGoFixture(t)
	// A directory where the generated .envrc goes makes that one write fail.
	envrc := filepath.Join(dir, ".envrc")
	if err := os.Mkdir(envrc, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := executeInitCmd(t, dir, "--yes", "--lang", "go")
	if err == nil || !strings.Contains(err.Error(), "partial write") || !strings.Contains(err.Error(), "same init command with --merge") {
		t.Fatalf("want a partial-write error naming the re-run, got %v\n%s", err, out)
	}
	gitignore := readProjectFile(t, dir, ".gitignore")
	for _, entry := range []string{branding.Get().StateDir + "/", "." + branding.Get().AppName + "/"} {
		if !strings.Contains(gitignore, entry) {
			t.Errorf(".gitignore missing %s after a partial write:\n%s", entry, gitignore)
		}
	}
	if _, err := os.Stat(answers.PrimaryPath(dir)); err != nil {
		t.Errorf("answers not saved after a partial write: %v", err)
	}
	requireFileNotExists(t, dir, branding.Get().ConfigFile)

	if err := os.Remove(envrc); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--yes", "--merge", "--lang", "go"); err != nil {
		t.Fatalf("re-run after fixing the failure: %v\n%s", err, out)
	}
	requireFileExists(t, dir, branding.Get().ConfigFile)
	requireFileExists(t, dir, ".envrc")
}
