package procexec

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// guardPanic builds a command under the forbid-exec guard and returns the
// panic message, or "" when the guard let it through.
func guardPanic(t *testing.T, name string, args ...string) (msg string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			msg, _ = r.(string)
			if msg == "" {
				msg = "non-string panic"
			}
		}
	}()
	_ = Command(name, args...)
	_ = CommandContext(context.Background(), name, args...)
	return ""
}

func TestForbidExec(t *testing.T) {
	tests := []struct {
		name   string
		forbid bool
		argv   []string
		panics bool
	}{
		{"env unset allows anything", false, []string{"npx", "-y", "some-server"}, false},
		{"docker --version", true, []string{"/usr/bin/docker", "--version"}, false},
		{"docker compose version", true, []string{"docker", "compose", "version"}, false},
		{"git rev-parse", true, []string{"git", "rev-parse", "--show-toplevel"}, false},
		{"git -C dir rev-parse", true, []string{"git", "-C", "/p", "rev-parse", "--git-path", "hooks"}, false},
		{"windows git.exe upper case", true, []string{`C:\Program Files\Git\bin\GIT.EXE`, "-C", "d", "rev-parse"}, false},
		{"nix-instantiate parse stdin", true, []string{"/nix/store/x/bin/nix-instantiate", "--parse", "-"}, false},
		{"bash -n", true, []string{"/bin/bash", "-n"}, false},
		{"ps -p", true, []string{"ps", "-p", "42", "-o", "comm="}, false},
		{"cmd /c ver", true, []string{"cmd", "/c", "ver"}, false},
		{"xcode-select -p", true, []string{"xcode-select", "-p"}, false},
		{"ghc --numeric-version", true, []string{"ghc", "--numeric-version"}, false},
		{"ghc running a file forbidden", true, []string{"ghc", "Setup.hs"}, true},
		{"sw_vers", true, []string{"sw_vers", "-productVersion"}, false},
		{"uname -r", true, []string{"uname", "-r"}, false},
		{"sysctl -n", true, []string{"sysctl", "-n", "sysctl.proc_translated"}, false},
		{"podman info format", true, []string{"podman", "info", "--format", "{{.Host.Security.Rootless}}"}, false},
		{"npx launcher forbidden", true, []string{"npx", "-y", "@socketsecurity/mcp"}, true},
		{"version with extra arg forbidden", true, []string{"node", "--version", "x"}, true},
		{"git fetch forbidden", true, []string{"git", "-C", "d", "fetch"}, true},
		{"git -C without dir forbidden", true, []string{"git", "-C"}, true},
		{"go list forbidden", true, []string{"go", "list", "-m", "-u", "all"}, true},
		{"devenv update forbidden", true, []string{"devenv", "update"}, true},
		{"bash script forbidden", true, []string{"bash", "-c", "curl x"}, true},
		{"docker compose up forbidden", true, []string{"docker", "compose", "up"}, true},
		{"bare binary forbidden", true, []string{"qsdev"}, true},
		// No argv shape is safe for an arbitrary binary: these fetch, run
		// project code, mutate the project or start an interpreter.
		{"npx version forbidden", true, []string{"npx", "version"}, true},
		{"uvx version forbidden", true, []string{"uvx", "version"}, true},
		{"bunx version forbidden", true, []string{"bunx", "version"}, true},
		{"gradlew --version forbidden", true, []string{"./gradlew", "--version"}, true},
		{"mvnw --version forbidden", true, []string{"./mvnw", "--version"}, true},
		{"make version forbidden", true, []string{"make", "version"}, true},
		{"gradle version forbidden", true, []string{"gradle", "version"}, true},
		{"mix version forbidden", true, []string{"mix", "version"}, true},
		{"just version forbidden", true, []string{"just", "version"}, true},
		{"task version forbidden", true, []string{"task", "version"}, true},
		{"yarn version forbidden", true, []string{"yarn", "version"}, true},
		{"sh -v forbidden", true, []string{"sh", "-v"}, true},
		{"bash -v forbidden", true, []string{"bash", "-v"}, true},
		{"python3 -v forbidden", true, []string{"python3", "-v"}, true},
		{"node_modules bin --version forbidden", true, []string{"/repo/node_modules/.bin/x", "--version"}, true},
		{"go version forbidden", true, []string{"go", "version"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.forbid {
				t.Setenv(ForbidExecEnv, "1")
			} else {
				t.Setenv(ForbidExecEnv, "")
			}
			msg := guardPanic(t, tt.argv[0], tt.argv[1:]...)
			if got := msg != ""; got != tt.panics {
				t.Fatalf("argv %q: panicked=%v (%q), want %v", tt.argv, got, msg, tt.panics)
			}
			if tt.panics && !strings.Contains(msg, strings.Join(tt.argv, " ")) {
				t.Errorf("panic message %q does not name argv %q", msg, tt.argv)
			}
		})
	}
}

func TestForbidExec_VersionProbe(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join("bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		path    string
		flag    string
		wantErr bool
	}{
		{"absolute --version", abs, "--version", false},
		{"absolute version", abs, "version", false},
		{"absolute -v", abs, "-v", false},
		{"relative dot path", "./gradlew", "--version", true},
		{"bare name", "npx", "version", true},
		{"relative dir path", filepath.Join("node_modules", ".bin", "x"), "--version", true},
		{"other flag", abs, "--help", true},
		{"flag with value", abs, "--version=1", true},
		{"empty flag", abs, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(ForbidExecEnv, "1")
			cmd := VersionProbe(context.Background(), tt.path, tt.flag)
			if got := cmd.Err != nil; got != tt.wantErr {
				t.Fatalf("VersionProbe(%q, %q).Err = %v, want error %v", tt.path, tt.flag, cmd.Err, tt.wantErr)
			}
			if tt.wantErr {
				if err := cmd.Run(); err == nil {
					t.Error("rejected version probe ran without error")
				}
				return
			}
			if want := []string{tt.path, tt.flag}; !slices.Equal(cmd.Args, want) {
				t.Errorf("Args = %q, want %q", cmd.Args, want)
			}
			// An empty Dir means the caller's working directory, the
			// project, whose pins a toolchain shim would follow.
			if cmd.Dir == "" || cmd.Dir != NeutralDir() {
				t.Errorf("Dir = %q, want NeutralDir %q", cmd.Dir, NeutralDir())
			}
		})
	}
}

func TestLocalProbes_DeclaredOnce(t *testing.T) {
	t.Parallel()
	if len(localProbes) == 0 {
		t.Fatal("localProbes is empty")
	}
	for i, p := range localProbes {
		if p.Name == "" {
			t.Errorf("localProbes[%d] (%q) names no binary: it would allow that argv for any binary", i, p.Args)
		}
		if len(p.Args) == 0 {
			t.Errorf("localProbes[%d] (%q) has no args: it would allow a bare binary", i, p.Name)
		}
		for _, a := range p.Args {
			if a == "" {
				t.Errorf("localProbes[%d] (%q) has an empty arg", i, p.Name)
			}
		}
	}

	// The package declares exactly one []localProbe literal: localProbes.
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	literals := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, f, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if arr, ok := lit.Type.(*ast.ArrayType); ok {
				if id, ok := arr.Elt.(*ast.Ident); ok && id.Name == "localProbe" {
					literals++
				}
			}
			return true
		})
	}
	if literals != 1 {
		t.Errorf("found %d []localProbe literals, want exactly 1 (localProbes)", literals)
	}
}
