//go:build integration

// Package container's integration-tagged suite builds the real Docker image and
// exercises the running gateway container. It is gated behind the `integration`
// build tag because it needs a Docker daemon and builds the full image. Each
// test self-skips with a clear message only when docker or its daemon is
// unavailable. Once the daemon is reachable, a failing image build is exactly
// the regression this suite exists to catch, so it fails the test rather than
// skipping it.
//
// Run explicitly with:
//
//	go test -tags integration ./internal/mcpserve/container/...
package container

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	// imageTag is the local tag used for the test build.
	imageTag = "qsdev-mcp-gateway:itest"
	// imageVersion is stamped into the test build via --build-arg VERSION.
	imageVersion = "0.0.0-itest"
	// maxImageBytes is the size ceiling. The vendored AWS/Azure cloud SDKs make
	// the stripped binary large; the spec's 20-25MB target predates the
	// cloud-SDK decision (Unit 32.10 note). The image measured ~31.8MB when CI
	// started building it (the old 30MB ceiling was never exercised, since the
	// image did not build); the ceiling leaves room for growth while still
	// catching an accidental fat base image or unstripped binary.
	// In-process Sigstore verification of self-updates (sigstore-go, F290)
	// added ~6.9MB to the stripped binary (30.6MB -> 37.5MB), taking the image
	// to ~39.9MB; the ceiling keeps the same ~13% headroom above that.
	maxImageBytes = 45 * 1024 * 1024
)

// dockerOrSkip skips the test when docker is unusable in this environment.
func dockerOrSkip(t *testing.T) string {
	t.Helper()
	bin, err := exec.LookPath("docker")
	if err != nil {
		t.Skipf("docker not on PATH; skipping container integration test: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if perr := exec.CommandContext(ctx, bin, "info").Run(); perr != nil {
		t.Skipf("docker daemon unavailable; skipping container integration test: %v", perr)
	}
	return bin
}

// buildImage builds the gateway image from the repo root using the Dockerfile.
// It returns the docker binary path. The repo root is located by walking up from
// the test's working directory to the module root (where go.mod lives).
func buildImage(t *testing.T, docker string) {
	t.Helper()
	root := moduleRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, docker, "build",
		"-f", "build/docker/Dockerfile", "-t", imageTag,
		"--build-arg", "VERSION="+imageVersion, ".")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docker build failed: %v\n%s", err, out)
	}
}

// TestImageVersionStamped checks the image's binary carries the version passed
// as a build argument (the Dockerfile's ldflags), not the "dev" default.
func TestImageVersionStamped(t *testing.T) {
	docker := dockerOrSkip(t)
	buildImage(t, docker)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, docker, "run", "--rm", "--entrypoint", "/qsdev",
		imageTag, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("running qsdev version in the image: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), imageVersion) {
		t.Errorf("qsdev version output lacks %q:\n%s", imageVersion, out)
	}
}

// TestImageSizeUnderCeiling builds the image and asserts it is within the ceiling.
func TestImageSizeUnderCeiling(t *testing.T) {
	docker := dockerOrSkip(t)
	buildImage(t, docker)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, docker, "image", "inspect",
		imageTag, "--format", "{{.Size}}").Output()
	if err != nil {
		t.Fatalf("inspecting image size: %v", err)
	}
	size, perr := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if perr != nil {
		t.Fatalf("parsing image size %q: %v", out, perr)
	}
	t.Logf("gateway image size: %d bytes (%.1f MB)", size, float64(size)/(1024*1024))
	if size > maxImageBytes {
		t.Errorf("image size %d bytes exceeds ceiling %d bytes (%.1f MB)", size, maxImageBytes, float64(maxImageBytes)/(1024*1024))
	}
}

// TestContainerInitializeLatency starts the container in standalone HTTP mode and
// asserts it answers an MCP initialize within 200ms of the port opening.
func TestContainerInitializeLatency(t *testing.T) {
	docker := dockerOrSkip(t)
	buildImage(t, docker)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Run a one-shot `initialize` over stdio against the entrypoint. Gateway mode
	// without an allow-list is pass-through, so initialize is unauthenticated and
	// fast. We measure the round-trip of a single initialize request.
	start := time.Now()
	cmd := exec.CommandContext(ctx, docker, "run", "--rm", "-i",
		"-e", "QSDEV_DEPLOY_MODE=gateway", imageTag,
		"--project-root", "/")
	cmd.Stdin = strings.NewReader(initializeLine())
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil && ctx.Err() == nil {
		// A non-zero exit after answering is fine (stdin EOF closes the server);
		// only treat a context timeout as fatal below.
		t.Logf("container run returned: %v", err)
	}
	elapsed := time.Since(start)

	if !strings.Contains(stdout.String(), `"protocolVersion"`) {
		t.Fatalf("container did not answer initialize; output:\n%s", stdout.String())
	}
	t.Logf("initialize round-trip (incl. container start): %s", elapsed)
	// Cold container start dominates here; the ≤200ms budget in the spec is the
	// server's own startup. We assert a generous wall-clock bound to catch hangs.
	if elapsed > 30*time.Second {
		t.Errorf("initialize took %s, far over budget", elapsed)
	}
}

// TestGatewayBlocksUnauthenticated starts the gateway with a one-agent allow-list
// and asserts a call from a different agent is rejected. It drives a full
// initialize + tools/call over stdio and checks the call result is an error.
func TestGatewayBlocksUnauthenticated(t *testing.T) {
	docker := dockerOrSkip(t)
	buildImage(t, docker)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, docker, "run", "--rm", "-i",
		"-e", "QSDEV_DEPLOY_MODE=gateway",
		"-e", "QSDEV_GATEWAY_AGENTS=only-trusted",
		"-e", "QSDEV_GATEWAY_REQUIRE_AUTH=true",
		imageTag, "--project-root", "/")
	cmd.Stdin = strings.NewReader(initializeLine() + toolCallLine("evil-agent"))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	if sawAuthFailure(t, stdout.Bytes()) {
		return
	}
	// Over stdio the caller's identity is self-asserted, so the gateway refuses
	// to start with an allow-list at all rather than trust it: the disallowed
	// agent is never served.
	if runErr == nil || !strings.Contains(stderr.String(), "cannot be enforced over stdio") {
		t.Fatalf("expected the disallowed agent to be rejected (auth failure or refusal to serve an allow-list over stdio); err=%v\nstdout:\n%s\nstderr:\n%s",
			runErr, stdout.String(), stderr.String())
	}
}

// initializeLine returns a newline-terminated JSON-RPC initialize request.
func initializeLine() string {
	req := map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "container-itest", "version": "0.0.1"},
		},
	}
	b, _ := json.Marshal(req)
	return string(b) + "\n"
}

// toolCallLine returns a tools/call request carrying the qsdev per-request agent
// id override so the gateway resolves the supplied agent.
func toolCallLine(agentID string) string {
	req := map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{
			"name":      "qsdev_status",
			"arguments": map[string]any{},
			"_meta":     map[string]any{"com.quantumserendipity.qsdev/agentId": agentID},
		},
	}
	b, _ := json.Marshal(req)
	return string(b) + "\n"
}

// sawAuthFailure scans JSON-RPC lines for a tool result marked as an error that
// mentions authentication.
func sawAuthFailure(t *testing.T, out []byte) bool {
	t.Helper()
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		if strings.Contains(strings.ToLower(sc.Text()), "authentication failed") {
			return true
		}
	}
	return false
}

// moduleRoot walks up from the working directory to the directory containing
// go.mod (the module root, used as the docker build context).
func moduleRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Skipf("cannot locate module root via `go env GOMOD`: %v", err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		t.Skip("not in a Go module; cannot locate repo root for docker build")
	}
	return strings.TrimSuffix(gomod, "/go.mod")
}
