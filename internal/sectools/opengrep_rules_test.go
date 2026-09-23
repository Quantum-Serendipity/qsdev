package sectools_test

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/rules"
)

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

// coreRuleFiles returns the rule library exactly as it is embedded and
// delivered into projects (rules.CoreRuleFiles), so these checks follow the
// library wherever it lives. An empty library is a failure, not a skip: the
// validation below must never silently stop running.
func coreRuleFiles(t *testing.T) []rules.RuleFile {
	t.Helper()
	files, err := rules.CoreRuleFiles()
	if err != nil {
		t.Fatalf("loading embedded rule library: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("embedded rule library contains no rule files")
	}
	return files
}

func TestRuleFiles_YAMLSyntax(t *testing.T) {
	t.Parallel()
	for _, file := range coreRuleFiles(t) {
		t.Run(file.RelPath, func(t *testing.T) {
			t.Parallel()
			var rf ruleFile
			if err := yaml.Unmarshal(file.Content, &rf); err != nil {
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
	for _, file := range coreRuleFiles(t) {
		t.Run(file.RelPath, func(t *testing.T) {
			t.Parallel()
			var rf ruleFile
			if err := yaml.Unmarshal(file.Content, &rf); err != nil {
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
	for _, file := range coreRuleFiles(t) {
		t.Run(file.RelPath, func(t *testing.T) {
			t.Parallel()
			var rf ruleFile
			if err := yaml.Unmarshal(file.Content, &rf); err != nil {
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
	seen := make(map[string]string)
	for _, file := range coreRuleFiles(t) {
		var rf ruleFile
		if err := yaml.Unmarshal(file.Content, &rf); err != nil {
			t.Fatalf("YAML parse error in %s: %v", file.RelPath, err)
		}

		for _, rule := range rf.Rules {
			if prev, ok := seen[rule.ID]; ok {
				t.Errorf("duplicate rule ID %q in %s (first seen in %s)", rule.ID, file.RelPath, prev)
			}
			seen[rule.ID] = file.RelPath
		}
	}
}

// Engine validation of the whole library (every rule parses, none is skipped)
// lives in the rules package: TestCoreRules_Validate.
