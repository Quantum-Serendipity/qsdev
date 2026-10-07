package container

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// dockerfilePath is the gateway image's Dockerfile, relative to this package.
var dockerfilePath = filepath.Join("..", "..", "..", "build", "docker", "Dockerfile")

// ociSourceLabel links a GHCR package to the repository that publishes it.
const ociSourceLabel = "org.opencontainers.image.source"

// dockerInstruction is one Dockerfile instruction: its upper-cased keyword
// and its arguments, with line continuations joined.
type dockerInstruction struct {
	Cmd  string
	Args string
}

// dockerStage is the instructions of one build stage, FROM first.
type dockerStage []dockerInstruction

// parseDockerfile splits a Dockerfile into stages. It handles comments,
// blank lines and backslash continuations, which is all this Dockerfile
// uses; heredocs are not supported.
func parseDockerfile(src string) []dockerStage {
	var stages []dockerStage
	var cur strings.Builder
	flush := func() {
		line := strings.TrimSpace(cur.String())
		cur.Reset()
		if line == "" {
			return
		}
		cmd, args, _ := strings.Cut(line, " ")
		in := dockerInstruction{Cmd: strings.ToUpper(cmd), Args: strings.Join(strings.Fields(args), " ")}
		if in.Cmd == "FROM" || len(stages) == 0 {
			stages = append(stages, nil)
		}
		stages[len(stages)-1] = append(stages[len(stages)-1], in)
	}
	for _, raw := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if body, ok := strings.CutSuffix(line, `\`); ok {
			cur.WriteString(body + " ")
			continue
		}
		cur.WriteString(line)
		flush()
	}
	flush()
	return stages
}

// find returns the arguments of every instruction cmd in the stage.
func (s dockerStage) find(cmd string) []string {
	var out []string
	for _, in := range s {
		if in.Cmd == cmd {
			out = append(out, in.Args)
		}
	}
	return out
}

// declares reports whether an instruction cmd (ARG, ENV or LABEL, in the
// key=value form or a bare ARG key) in the stage sets key to want, quotes
// ignored; an empty want only checks that key is declared.
func (s dockerStage) declares(cmd, key, want string) bool {
	for _, args := range s.find(cmd) {
		for _, f := range strings.Fields(args) {
			k, v, hasValue := strings.Cut(f, "=")
			if k != key {
				continue
			}
			if want == "" || (hasValue && strings.Trim(v, `"`) == want) {
				return true
			}
		}
	}
	return false
}

// TestDockerfileCrossCompiles is the U26-04 regression test for the
// multi-arch gateway image: the release builds linux/amd64 and linux/arm64
// with plain docker and no emulation, so the builder runs on the build
// platform and cross-compiles for the target, and the runtime stage runs
// nothing. The image also links its GHCR package to the repository.
func TestDockerfileCrossCompiles(t *testing.T) {
	t.Parallel()

	b, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("reading %s: %v", dockerfilePath, err)
	}
	stages := parseDockerfile(string(b))
	if len(stages) < 2 {
		t.Fatalf("%s has %d stages, want a builder and a runtime stage", dockerfilePath, len(stages))
	}
	builder, final := stages[0], stages[len(stages)-1]

	if from := builder[0].Args; !strings.HasPrefix(from, "--platform=$BUILDPLATFORM ") {
		t.Errorf("builder FROM %q does not run on $BUILDPLATFORM; a foreign-arch builder needs emulation", from)
	}
	for _, arg := range []string{"TARGETOS", "TARGETARCH"} {
		if !builder.declares("ARG", arg, "") {
			t.Errorf("builder stage does not declare ARG %s", arg)
		}
	}
	for key, want := range map[string]string{"GOOS": "${TARGETOS}", "GOARCH": "${TARGETARCH}"} {
		if !builder.declares("ENV", key, want) {
			t.Errorf("builder stage does not set ENV %s=%s", key, want)
		}
	}
	if !slices.ContainsFunc(builder.find("RUN"), func(r string) bool { return strings.Contains(r, "go build") }) {
		t.Error("builder stage has no go build")
	}

	if runs := final.find("RUN"); len(runs) > 0 {
		t.Errorf("runtime stage runs %q; it would need emulation for a foreign platform", runs)
	}
	if from := final[0].Args; !strings.Contains(from, "@sha256:") {
		t.Errorf("runtime FROM %q is not pinned by digest", from)
	}
	if !final.declares("LABEL", ociSourceLabel, branding.RepoURL()) {
		t.Errorf("runtime stage has no LABEL %s=%s", ociSourceLabel, branding.RepoURL())
	}
}

// TestParseDockerfile pins the parser TestDockerfileCrossCompiles relies on.
func TestParseDockerfile(t *testing.T) {
	t.Parallel()

	src := "# syntax=docker/dockerfile:1\r\n" +
		"FROM --platform=$BUILDPLATFORM a AS b\n" +
		"# comment\n" +
		"ARG X Y\n" +
		"RUN go build \\\n" +
		"      -o /x \\\n" +
		"      ./cmd\n" +
		"\n" +
		"from c\n" +
		"LABEL k=\"v\"\n"
	stages := parseDockerfile(src)
	if len(stages) != 2 {
		t.Fatalf("parseDockerfile() = %d stages, want 2: %+v", len(stages), stages)
	}
	if got := stages[0].find("RUN"); !slices.Equal(got, []string{"go build -o /x ./cmd"}) {
		t.Errorf("RUN = %q, want the joined continuation", got)
	}
	if !stages[0].declares("ARG", "Y", "") || stages[0].declares("ARG", "Z", "") {
		t.Error("declares(ARG) misreports the builder's ARG list")
	}
	if got := stages[1][0]; got.Cmd != "FROM" || got.Args != "c" {
		t.Errorf("second stage starts with %+v, want FROM c", got)
	}
	if !stages[1].declares("LABEL", "k", "v") {
		t.Error("declares(LABEL) misses k=\"v\"")
	}
}
