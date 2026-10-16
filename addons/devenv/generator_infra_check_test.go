package devenv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGenerate_InfraProfileConfigsPassCheck is the regression test for
// `qsdev check` expecting the raw .qsdev.yaml registry proxy while
// generation wrote the infra profile's repository layout (Nexus
// npm-group/pypi-group, Artifactory api/npm/npm-virtual): the package-manager
// security configs the generator writes for every infra profile, with and
// without one selected, must pass the check that reads the same .qsdev.yaml.
func TestGenerate_InfraProfileConfigsPassCheck(t *testing.T) {
	t.Parallel()
	infra := types.InfraConfig{
		RegistryProxy:     "https://repo.corp.internal/base",
		NixCache:          "corp",
		NixCachePublicKey: testNixCacheKey,
	}
	langs := []types.LanguageChoice{
		{Name: "javascript", PackageManager: "npm"},
		{Name: "javascript", PackageManager: "pnpm"},
		{Name: "javascript", PackageManager: "yarn"},
		{Name: "javascript", PackageManager: "bun"},
		{Name: "python", PackageManager: "pip"},
		{Name: "python"},
		{Name: "php"},
	}
	for _, profileName := range []string{"enterprise", "consulting-default", "startup-github", ""} {
		for _, lang := range langs {
			t.Run(profileName+"/"+lang.Name+"/"+lang.PackageManager, func(t *testing.T) {
				t.Parallel()
				answers := infraAnswers(profileName, infra, lang)
				files, err := generateInfra(t, answers)
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}
				dir := t.TempDir()
				for path, content := range files {
					p := filepath.Join(dir, filepath.FromSlash(path))
					if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				qcfg := config.AnswersToConfig(answers, "0.0.0")
				var checked int
				for _, r := range check.CheckSecurityHardening(check.CheckContext{ProjectRoot: dir, QsdevConfig: &qcfg}) {
					if !strings.HasPrefix(r.Name, "security_config_") {
						continue
					}
					checked++
					if r.Status != check.StatusPass {
						t.Errorf("%s %s: Status = %s, want PASS (%s)", r.Name, r.FilePath, r.Status, r.Message)
					}
				}
				if checked == 0 {
					t.Fatal("no security_config results; the check verified nothing")
				}
			})
		}
	}
}
