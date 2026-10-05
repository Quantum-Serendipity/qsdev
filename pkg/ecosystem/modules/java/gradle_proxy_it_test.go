//go:build gradleit

package java_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestGradleProxyScriptKeepsProxy runs a real Gradle, offline, on a fixture
// that declares Maven Central, JitPack and the Gradle Plugin Portal at the
// project and settings level, with the generated proxy init script. Every
// repository list must end up holding exactly the proxy: the script must
// remove the others and keep the one it adds. Opt in with -tags gradleit; it
// skips when gradle is not on PATH.
func TestGradleProxyScriptKeepsProxy(t *testing.T) {
	gradle, err := exec.LookPath("gradle")
	if err != nil {
		t.Skip("gradle not on PATH")
	}
	const proxy = "https://proxy.example.com/maven/"

	dir := t.TempDir()
	for _, name := range []string{"settings.gradle", "build.gradle"} {
		data, err := os.ReadFile(filepath.Join("testdata", "gradleit", name))
		if err != nil {
			t.Fatal(err)
		}
		writeProjectFile(t, dir, name, string(data))
	}
	f, ok := gradleProxyFile(t, ecosystem.ModuleConfig{
		RegistryProxy: proxy,
		Extras:        map[string]string{"build_tool": "gradle"},
	})
	if !ok {
		t.Fatalf("no %s generated", gradleProxyScript)
	}
	writeProjectFile(t, dir, f.Path, string(f.Content))

	cmd := exec.Command(gradle, "--offline", "--no-daemon", "-q",
		"-I", filepath.Join(dir, filepath.FromSlash(f.Path)), "-p", dir, "showRepositories")
	cmd.Env = append(os.Environ(), "GRADLE_USER_HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gradle: %v\n%s", err, out)
	}

	got := map[string][]string{}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "QSDEV_REPO" {
			got[fields[1]] = append(got[fields[1]], strings.TrimSuffix(fields[2], "/"))
		}
	}
	want := []string{strings.TrimSuffix(proxy, "/")}
	for _, scope := range []string{"plugins", "settings", "project"} {
		if !slices.Equal(got[scope], want) {
			t.Errorf("%s repositories = %q, want exactly the proxy %q\n%s", scope, got[scope], want, out)
		}
	}
}
