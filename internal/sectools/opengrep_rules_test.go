package sectools_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func rulesRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "rules", "core")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("rules/core not found at %s: %v", root, err)
	}
	return root
}

type ruleMetadata struct {
	Category    string `yaml:"category"`
	Subcategory string `yaml:"subcategory"`
	Confidence  string `yaml:"confidence"`
	CWE         string `yaml:"cwe"`
	OWASP       string `yaml:"owasp"`
}

type ruleEntry struct {
	ID             string       `yaml:"id"`
	Languages      []string     `yaml:"languages"`
	Severity       string       `yaml:"severity"`
	Message        string       `yaml:"message"`
	Mode           string       `yaml:"mode"`
	Pattern        string       `yaml:"pattern"`
	Patterns       []any        `yaml:"patterns"`
	PatternSources []any        `yaml:"pattern-sources"`
	PatternSinks   []any        `yaml:"pattern-sinks"`
	PatternRegex   string       `yaml:"pattern-regex"`
	Metadata       ruleMetadata `yaml:"metadata"`
}

type ruleFile struct {
	Rules []ruleEntry `yaml:"rules"`
}

func findRuleFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".yaml" && info.Name() != "manifest.yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking rules directory: %v", err)
	}
	return files
}

func TestRuleFiles_YAMLSyntax(t *testing.T) {
	t.Parallel()
	root := rulesRoot(t)
	files := findRuleFiles(t, root)
	if len(files) == 0 {
		t.Skip("no rule files found")
	}

	for _, path := range files {
		relPath, _ := filepath.Rel(root, path)
		t.Run(relPath, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			var rf ruleFile
			if err := yaml.Unmarshal(data, &rf); err != nil {
				t.Fatalf("YAML parse error: %v", err)
			}
			if len(rf.Rules) == 0 {
				t.Error("file contains no rules")
			}
		})
	}
}

func TestRuleFiles_RequiredFields(t *testing.T) {
	t.Parallel()
	root := rulesRoot(t)
	files := findRuleFiles(t, root)
	if len(files) == 0 {
		t.Skip("no rule files found")
	}

	for _, path := range files {
		relPath, _ := filepath.Rel(root, path)
		t.Run(relPath, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			var rf ruleFile
			if err := yaml.Unmarshal(data, &rf); err != nil {
				t.Fatalf("YAML parse error: %v", err)
			}

			for i, rule := range rf.Rules {
				name := rule.ID
				if name == "" {
					name = fmt.Sprintf("rule[%d]", i)
				}
				t.Run(name, func(t *testing.T) {
					if rule.ID == "" {
						t.Error("missing id")
					}
					if !strings.HasPrefix(rule.ID, "qsdev.core.") {
						t.Errorf("id %q should start with qsdev.core.", rule.ID)
					}
					if len(rule.Languages) == 0 {
						t.Error("missing languages")
					}
					if rule.Severity == "" {
						t.Error("missing severity")
					}
					if rule.Message == "" {
						t.Error("missing message")
					}

					hasPattern := rule.Mode != "" || rule.Pattern != "" ||
						len(rule.Patterns) > 0 || rule.PatternRegex != ""
					if !hasPattern {
						t.Error("rule must have mode, pattern, patterns, or pattern-regex")
					}

					if rule.Mode == "taint" {
						if len(rule.PatternSources) == 0 {
							t.Error("taint rule missing pattern-sources")
						}
						if len(rule.PatternSinks) == 0 {
							t.Error("taint rule missing pattern-sinks")
						}
					}
				})
			}
		})
	}
}

func TestRuleFiles_MetadataSchema(t *testing.T) {
	t.Parallel()
	root := rulesRoot(t)
	files := findRuleFiles(t, root)
	if len(files) == 0 {
		t.Skip("no rule files found")
	}

	for _, path := range files {
		relPath, _ := filepath.Rel(root, path)
		t.Run(relPath, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			var rf ruleFile
			if err := yaml.Unmarshal(data, &rf); err != nil {
				t.Fatalf("YAML parse error: %v", err)
			}

			for _, rule := range rf.Rules {
				t.Run(rule.ID, func(t *testing.T) {
					m := rule.Metadata
					if m.Category != "security" {
						t.Errorf("metadata.category = %q, want %q", m.Category, "security")
					}
					if m.Subcategory == "" {
						t.Error("missing metadata.subcategory")
					}
					if m.Confidence == "" {
						t.Error("missing metadata.confidence")
					}
					if m.CWE == "" {
						t.Error("missing metadata.cwe")
					}
					if !strings.HasPrefix(m.CWE, "CWE-") {
						t.Errorf("metadata.cwe = %q, should start with CWE-", m.CWE)
					}
					if m.OWASP == "" {
						t.Error("missing metadata.owasp")
					}
				})
			}
		})
	}
}

func TestRuleFiles_IDUniqueness(t *testing.T) {
	t.Parallel()
	root := rulesRoot(t)
	files := findRuleFiles(t, root)
	if len(files) == 0 {
		t.Skip("no rule files found")
	}

	seen := make(map[string]string)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		var rf ruleFile
		if err := yaml.Unmarshal(data, &rf); err != nil {
			t.Fatalf("YAML parse error in %s: %v", path, err)
		}

		relPath, _ := filepath.Rel(root, path)
		for _, rule := range rf.Rules {
			if prev, ok := seen[rule.ID]; ok {
				t.Errorf("duplicate rule ID %q in %s (first seen in %s)", rule.ID, relPath, prev)
			}
			seen[rule.ID] = relPath
		}
	}
}

func TestRulesValidateWithOpengrep(t *testing.T) {
	if _, err := exec.LookPath("opengrep"); err != nil {
		t.Skip("opengrep not available")
	}
	root := rulesRoot(t)

	cmd := exec.Command("opengrep", "scan", "--validate", "--config", root)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("opengrep scan --validate failed: %v\noutput: %s", err, output)
	}
}
