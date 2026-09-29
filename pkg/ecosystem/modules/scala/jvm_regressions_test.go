package scala_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// writeTree creates each file (slash-separated path -> content) under dir.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestDetect_Markers covers the Scala markers: a bare project/ directory (a
// common folder name in Maven and other repos) is not Scala, while sbt
// metadata under project/ and every Mill build file name are.
func TestDetect_Markers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		files      map[string]string
		detected   bool
		confidence ecosystem.Confidence
		buildTool  string
		evidence   string
	}{
		{name: "bare project dir", files: map[string]string{"pom.xml": "<project/>", "project/README.md": "docs"}},
		{name: "project dir with java sources", files: map[string]string{"project/src/Main.java": "class Main {}"}},
		{
			name:     "sbt build.properties",
			files:    map[string]string{"project/build.properties": "sbt.version=1.10.0\n"},
			detected: true, confidence: ecosystem.ConfidenceProbable, buildTool: "sbt", evidence: "sbt version 1.10.0",
		},
		{
			name:     "sbt plugins file",
			files:    map[string]string{"project/plugins.sbt": "addSbtPlugin(\"a\" % \"b\" % \"1\")\n"},
			detected: true, confidence: ecosystem.ConfidenceProbable, buildTool: "sbt",
		},
		{
			name:     "mill 1.x build.mill",
			files:    map[string]string{"build.mill": "package build\n", ".mill-version": "1.0.4\n"},
			detected: true, confidence: ecosystem.ConfidenceCertain, buildTool: "mill", evidence: "Mill version 1.0.4",
		},
		{
			name:     "mill yaml build",
			files:    map[string]string{"build.mill.yaml": "extends: ScalaModule\n"},
			detected: true, confidence: ecosystem.ConfidenceCertain, buildTool: "mill", evidence: "build.mill.yaml found",
		},
		{
			name:     "legacy build.sc warns",
			files:    map[string]string{"build.sc": "import mill._\n"},
			detected: true, confidence: ecosystem.ConfidenceCertain, buildTool: "mill", evidence: "rename it",
		},
		{
			name:     "sbt preferred over mill",
			files:    map[string]string{"build.sbt": "", "build.mill": ""},
			detected: true, confidence: ecosystem.ConfidenceCertain, buildTool: "sbt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTree(t, dir, tt.files)
			res := newModule().Detect(dir)
			if res.Detected != tt.detected {
				t.Fatalf("Detected = %v, want %v (evidence %v)", res.Detected, tt.detected, res.Evidence)
			}
			if !tt.detected {
				return
			}
			if res.Confidence != tt.confidence {
				t.Errorf("Confidence = %v, want %v", res.Confidence, tt.confidence)
			}
			if got := res.SuggestedConfig.Extra("build_tool", ""); got != tt.buildTool {
				t.Errorf("build_tool = %q, want %q", got, tt.buildTool)
			}
			if tt.evidence != "" && !slices.ContainsFunc(res.Evidence, func(e string) bool { return strings.Contains(e, tt.evidence) }) {
				t.Errorf("evidence %v should mention %q", res.Evidence, tt.evidence)
			}
		})
	}
}

// TestDevenvNixFragment_JavaCoexistence guards the Java+Scala devenv.nix: the
// Java module owns languages.java.enable and jdk.package, so the Scala
// fragment must not define either leaf in the shared attribute set. Its JDK
// is a lib.mkDefault in an imported module, which the Java choice overrides.
func TestDevenvNixFragment_JavaCoexistence(t *testing.T) {
	t.Parallel()

	for _, bt := range []string{"sbt", "mill"} {
		frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{
			Extras: map[string]string{"build_tool": bt, "jdk_version": "17"},
		})
		if err != nil {
			t.Fatalf("%s: %v", bt, err)
		}
		for _, bad := range []string{"languages.java = {", "languages.java.enable", "\n  languages.java.jdk.package"} {
			if strings.Contains(frag, bad) {
				t.Errorf("%s fragment defines %q in the shared attribute set:\n%s", bt, bad, frag)
			}
		}
		const want = "imports = [ { languages.java.jdk.package = lib.mkDefault pkgs.jdk17; } ];"
		if !strings.Contains(frag, want) {
			t.Errorf("%s fragment missing %q:\n%s", bt, want, frag)
		}
	}

	_, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Extras: map[string]string{"jdk_version": "23"}})
	if !errors.Is(err, ecosystem.ErrUnsupportedJDKVersion) {
		t.Errorf("jdk_version 23: err = %v, want ErrUnsupportedJDKVersion", err)
	}
}

// TestManifestFiles_Detected checks that sbt build plugins (project/*.sbt)
// and Mill build files are listed when present.
func TestManifestFiles_Detected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"sbt with plugins", map[string]string{"build.sbt": "", "project/plugins.sbt": "", "project/build.properties": ""}, []string{"build.sbt", "project/plugins.sbt"}},
		{"mill", map[string]string{"build.mill": ""}, []string{"build.mill"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTree(t, dir, tt.files)
			m := newModule()
			var got []string
			for _, mf := range m.ManifestFiles(m.Detect(dir).SuggestedConfig) {
				got = append(got, mf.Path)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ManifestFiles = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestScalafmtHook_Script runs the scalafmt hook script against a stub
// scalafmt. A project without .scalafmt.conf is skipped (scalafmt exits 16
// without one), the staged files are checked with the provisioned scalafmt,
// and a config pinning another scalafmt version fails instead of letting
// scalafmt download and run that release from Maven Central.
func TestScalafmtHook_Script(t *testing.T) {
	t.Parallel()

	bash := shelltest.Bash(t)
	var hook ecosystem.HookConfig
	for _, h := range newModule().PreCommitHooks(ecosystem.ModuleConfig{}) {
		if h.ID == "scalafmt" {
			hook = h
		}
	}
	if hook.Script == "" || !hook.PassFilenames || hook.NixPackage != "scalafmt" {
		t.Fatalf("scalafmt hook = %+v, want a script hook on the staged files using the scalafmt package", hook)
	}

	tests := []struct {
		name     string
		config   *string
		wantExit int
		wantRun  string // arguments the stub must receive; "" means never run
		wantMsg  string
	}{
		{name: "no config skips", config: nil, wantExit: 0, wantMsg: "skipping"},
		{name: "matching version", config: ptr("version = \"3.11.4\"\nrunner.dialect = scala3\n"), wantRun: "--check --non-interactive --respect-project-filters A.scala B.scala"},
		{name: "hocon colon unquoted", config: ptr("maxColumn = 100\nversion: 3.11.4\n"), wantRun: "--check --non-interactive --respect-project-filters A.scala B.scala"},
		{name: "other version refused", config: ptr("version = \"3.7.17\"\n"), wantExit: 1, wantMsg: "download scalafmt 3.7.17"},
		{name: "missing version refused", config: ptr("maxColumn = 100\n"), wantExit: 1, wantMsg: "does not set the required version"},
		{name: "commented version ignored", config: ptr("# version = \"3.11.4\"\nversion = \"3.7.17\"\n"), wantExit: 1, wantMsg: "download scalafmt 3.7.17"},
		{name: "crlf config", config: ptr("version = \"3.11.4\"\r\n"), wantRun: "--check --non-interactive --respect-project-filters A.scala B.scala"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			repo := filepath.Join(dir, "repo")
			log := filepath.Join(dir, "calls.log")
			writeTree(t, bin, map[string]string{"scalafmt": bash.Shebang() +
				"if [ \"$1\" = --version ]; then echo 'scalafmt 3.11.4'; exit 0; fi\n" +
				"echo \"$*\" >> " + shelltest.QuotePath(log) + "\n"})
			if err := os.Chmod(filepath.Join(bin, "scalafmt"), 0o755); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{"A.scala": "object A\n", "B.scala": "object B\n"}
			if tt.config != nil {
				files[".scalafmt.conf"] = *tt.config
			}
			writeTree(t, repo, files)

			cmd := exec.Command(bash.Path, "-c", hook.Script, "scalafmt-hook", "A.scala", "B.scala")
			cmd.Dir = repo
			cmd.Env = []string{"PATH=" + bin}
			out, err := cmd.CombinedOutput()
			exit := 0
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exit = exitErr.ExitCode()
			} else if err != nil {
				t.Fatalf("running hook: %v", err)
			}
			if exit != tt.wantExit {
				t.Errorf("exit = %d, want %d; output:\n%s", exit, tt.wantExit, out)
			}
			if !strings.Contains(string(out), tt.wantMsg) {
				t.Errorf("output %q should contain %q", out, tt.wantMsg)
			}
			calls, _ := os.ReadFile(log)
			if got := strings.TrimSpace(string(calls)); got != tt.wantRun {
				t.Errorf("scalafmt ran with %q, want %q", got, tt.wantRun)
			}
		})
	}
}

func ptr(s string) *string { return &s }
