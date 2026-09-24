package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// allowlistPOM declares a repository the project allowlists (confluent) and
// one it does not (jitpack).
const allowlistPOM = `<project xmlns="http://maven.apache.org/POM/4.0.0">
  <repositories>
    <repository><id>confluent</id><url>https://packages.confluent.io/maven/</url></repository>
  </repositories>
  <pluginRepositories>
    <pluginRepository><id>jitpack</id><url>https://jitpack.io</url></pluginRepository>
  </pluginRepositories>
</project>
`

const allowlistConfig = `version: 2
languages:
  - name: java
    package_manager: maven
java:
  repository_allowlist: [confluent]
`

// writeAllowlistProject writes the given files (relative path -> content)
// under a new project directory.
func writeAllowlistProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// assertAllowlistApplied checks the warning names only the non-allowlisted
// repository and the generated settings.xml mirror excludes the allowlisted
// one.
func assertAllowlistApplied(t *testing.T, dir, output string) {
	t.Helper()
	if !strings.Contains(output, "Warning: Java/Kotlin (JVM): ") || !strings.Contains(output, "jitpack (https://jitpack.io)") {
		t.Errorf("output lacks the redirected-repository warning for jitpack:\n%s", output)
	}
	if strings.Contains(output, "confluent (") {
		t.Errorf("allowlisted confluent reported as redirected:\n%s", output)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".mvn", "settings.xml"))
	if err != nil {
		t.Fatalf("reading generated settings.xml: %v", err)
	}
	if !strings.Contains(string(data), "<mirrorOf>*,!confluent</mirrorOf>") {
		t.Errorf("settings.xml mirror does not exclude confluent:\n%s", data)
	}
}

// TestJoin_AppliesJavaRepositoryAllowlist guards W089 for join: the committed
// java.repository_allowlist reaches settings.xml generation and the
// redirected-repository warning.
func TestJoin_AppliesJavaRepositoryAllowlist(t *testing.T) {
	t.Setenv("QSDEV_SKIP_SETUP", "1")
	dir := writeAllowlistProject(t, map[string]string{".qsdev.yaml": allowlistConfig, "pom.xml": allowlistPOM})
	cmd, buf := newJoinTestCmd()
	if err := runJoin(cmd, InitOptions{Yes: true}, dir); err != nil {
		t.Fatalf("runJoin: %v\n%s", err, buf.String())
	}
	assertAllowlistApplied(t, dir, buf.String())
}

// TestUpdate_AppliesJavaRepositoryAllowlist guards W089 for `init --update`:
// an allowlist added to the committed .qsdev.yaml after init is honoured, and
// kept in the file.
func TestUpdate_AppliesJavaRepositoryAllowlist(t *testing.T) {
	dir := writeAllowlistProject(t, map[string]string{"pom.xml": allowlistPOM})
	out, err := executeInitCmd(t, dir, "--lang", "java", "--java-build-tool", "maven", "--yes")
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "confluent (https://packages.confluent.io/maven/)") {
		t.Errorf("init without an allowlist does not warn about confluent:\n%s", out)
	}
	cfgPath := filepath.Join(dir, ".qsdev.yaml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg = append(cfg, "java:\n  repository_allowlist: [confluent]\n"...)
	if err := os.WriteFile(cfgPath, cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	// settings.xml is never overwritten, so the update reports the stale
	// mirror until it is edited as the warning says.
	out, err = executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `needs "*,!confluent"`) {
		t.Errorf("update does not report the stale settings.xml mirror:\n%s", out)
	}
	settingsPath := filepath.Join(dir, ".mvn", "settings.xml")
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	settings = []byte(strings.Replace(string(settings), "<mirrorOf>*</mirrorOf>", "<mirrorOf>*,!confluent</mirrorOf>", 1))
	if err := os.WriteFile(settingsPath, settings, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = executeInitCmd(t, dir, "--update")
	if err != nil {
		t.Fatalf("update failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "needs \"*,!confluent\"") {
		t.Errorf("update still reports the edited settings.xml mirror:\n%s", out)
	}
	assertAllowlistApplied(t, dir, out)
	cfg, err = os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "repository_allowlist:") {
		t.Errorf("update dropped java.repository_allowlist from .qsdev.yaml:\n%s", cfg)
	}
}

// TestUpdate_RejectsInvalidRepositoryAllowlist checks that an allowlist id
// that would change the mirrorOf expression stops update instead of being
// silently dropped.
func TestUpdate_RejectsInvalidRepositoryAllowlist(t *testing.T) {
	dir := writeAllowlistProject(t, map[string]string{"pom.xml": allowlistPOM})
	if out, err := executeInitCmd(t, dir, "--lang", "java", "--java-build-tool", "maven", "--yes"); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}
	cfgPath := filepath.Join(dir, ".qsdev.yaml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg = append(cfg, "java:\n  repository_allowlist: [\"confluent,*\"]\n"...)
	if err := os.WriteFile(cfgPath, cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := executeInitCmd(t, dir, "--update", "--dry-run")
	if err == nil || !strings.Contains(err.Error(), "java.repository_allowlist") {
		t.Fatalf("update error = %v, want a java.repository_allowlist error\n%s", err, out)
	}
}
