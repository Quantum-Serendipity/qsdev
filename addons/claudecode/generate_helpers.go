package claudecode

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// loadYAMLManifest reads a YAML file from the embedded template filesystem and
// unmarshals it into the type parameter T. The path is relative to the embed
// root (e.g. "templates/skills/manifest.yaml").
func loadYAMLManifest[T any](path string) (*T, error) {
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest T
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &manifest, nil
}

// skillHeader is the synthesized SKILL.md YAML front-matter. Distinct from
// deny_conflicts' skillFrontmatter, which parses front-matter back out.
type skillHeader struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// DisableModelInvocation makes the skill user-invoked only; omitted when
	// false so model-invocable skills keep the minimal header.
	DisableModelInvocation bool `yaml:"disable-model-invocation,omitempty"`
}

// prependSkillFrontMatter renders a minimal SKILL.md YAML front-matter block
// from hdr and prepends it to a flat template body so Claude Code can load the
// skill. Bodies that already begin with a front-matter delimiter are returned
// unchanged, so it is idempotent and safe if a template later grows its own
// front-matter. Marshaling via yaml.Marshal (rather than string concatenation)
// safely quotes descriptions containing ':' or other YAML metacharacters.
func prependSkillFrontMatter(hdr skillHeader, body []byte) ([]byte, error) {
	if bytes.HasPrefix(bytes.TrimLeft(body, " \t\r\n"), []byte("---")) {
		return body, nil
	}

	y, err := yaml.Marshal(hdr)
	if err != nil {
		return nil, fmt.Errorf("marshaling front-matter for skill %q: %w", hdr.Name, err)
	}

	var b bytes.Buffer
	b.WriteString("---\n")
	b.Write(y)
	b.WriteString("---\n\n")
	b.Write(body)
	return b.Bytes(), nil
}

// hookFileSpec describes a single hook file to be generated from a template.
type hookFileSpec struct {
	enabled      bool
	templatePath string
	outputPath   string
	mode         os.FileMode
	strategy     types.MergeStrategy
	owner        string
}

// generateHookFile reads an embedded template and returns a GeneratedFile when
// the spec is enabled. It returns (nil, nil) when the spec is disabled.
func generateHookFile(spec hookFileSpec) (*types.GeneratedFile, error) {
	if !spec.enabled {
		return nil, nil
	}

	content, err := templateFS.ReadFile(spec.templatePath)
	if err != nil {
		return nil, fmt.Errorf("reading hook template %s: %w", spec.templatePath, err)
	}

	return &types.GeneratedFile{
		Path:     spec.outputPath,
		Content:  content,
		Mode:     spec.mode,
		Strategy: spec.strategy,
		Owner:    spec.owner,
	}, nil
}
