package devenv

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const (
	nixpkgsURL     = "github:NixOS/nixpkgs/nixpkgs-unstable"
	gitHooksURL    = "github:cachix/git-hooks.nix"
	requireVersion = ">=2.1"
	yamlHeaderFmt  = "# %s init — security-hardened devenv configuration.\n# See https://devenv.sh/reference/yaml-options/ for all options.\n"
)

// DevenvYaml is the top-level structure that marshals to devenv.yaml.
type DevenvYaml struct {
	RequireVersion string                     `yaml:"require_version"`
	Inputs         map[string]DevenvYamlInput `yaml:"inputs"`
	Impure         bool                       `yaml:"impure"`       // no omitempty: false is security-critical
	AllowUnfree    bool                       `yaml:"allow_unfree"` // no omitempty
	AllowBroken    bool                       `yaml:"allow_broken"` // no omitempty
	Clean          DevenvClean                `yaml:"clean"`

	// These must always appear in output even when empty.
	PermittedUnfreePackages   []string `yaml:"permitted_unfree_packages"`
	PermittedInsecurePackages []string `yaml:"permitted_insecure_packages"`
}

// DevenvYamlInput represents a single flake input entry in devenv.yaml.
type DevenvYamlInput struct {
	URL    string                        `yaml:"url"`
	Inputs map[string]DevenvYamlSubInput `yaml:"inputs,omitempty"`
}

// DevenvYamlSubInput represents a sub-input override within a flake input.
// Used for follows declarations (e.g. inputs.nixpkgs.follows = "nixpkgs").
type DevenvYamlSubInput struct {
	Follows string `yaml:"follows,omitempty"`
}

// DevenvClean represents the clean section of devenv.yaml.
type DevenvClean struct {
	Enabled bool     `yaml:"enabled"` // no omitempty
	Keep    []string `yaml:"keep"`
}

// needsGitHooks determines whether the git-hooks input should be added.
// Security hooks (ripsecrets, check-added-large-files, etc.) are always
// enabled in devenv.nix, so the git-hooks input is always required.
func needsGitHooks(_ types.WizardAnswers, _ *ecosystem.Registry) bool {
	return true
}

// collectEcosystemInputs gathers DevenvYamlInputs from every selected module.
func collectEcosystemInputs(answers types.WizardAnswers, registry *ecosystem.Registry) map[string]DevenvYamlInput {
	if registry == nil {
		return nil
	}
	merged := make(map[string]DevenvYamlInput)
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		yip, ok := mod.(ecosystem.DevenvYamlInputProvider)
		if !ok {
			continue
		}
		cfg := ecosystem.ToModuleConfig(lang)
		for _, inp := range yip.DevenvYamlInputs(cfg) {
			key := inputKeyFromURL(inp.URL)
			entry := DevenvYamlInput{
				URL: inp.URL,
			}
			if inp.Follows != "" {
				entry.Inputs = map[string]DevenvYamlSubInput{
					"nixpkgs": {Follows: inp.Follows},
				}
			}
			merged[key] = entry
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// GenerateDevenvYaml produces a security-hardened devenv.yaml from the wizard
// answers and ecosystem registry.
func GenerateDevenvYaml(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	dy := DevenvYaml{
		RequireVersion: requireVersion,
		Inputs: map[string]DevenvYamlInput{
			"nixpkgs": {URL: nixpkgsURL},
		},
		Impure:                    false,
		AllowUnfree:               true,
		AllowBroken:               false,
		PermittedUnfreePackages:   []string{},
		PermittedInsecurePackages: []string{},
		Clean: DevenvClean{
			Enabled: true,
			Keep:    defaultCleanKeep(),
		},
	}

	// Add git-hooks input when needed.
	if needsGitHooks(answers, registry) {
		dy.Inputs["git-hooks"] = DevenvYamlInput{
			URL: gitHooksURL,
			Inputs: map[string]DevenvYamlSubInput{
				"nixpkgs": {Follows: "nixpkgs"},
			},
		}
	}

	// Merge ecosystem module inputs.
	ecoInputs := collectEcosystemInputs(answers, registry)
	for k, v := range ecoInputs {
		if _, exists := dy.Inputs[k]; !exists {
			dy.Inputs[k] = v
		}
	}

	out, err := marshalDevenvYaml(dy)
	if err != nil {
		return nil, fmt.Errorf("marshaling devenv.yaml: %w", err)
	}

	content := fmt.Sprintf(yamlHeaderFmt, branding.GeneratedBy()) + string(out)

	return &types.GeneratedFile{
		Path:     "devenv.yaml",
		Content:  []byte(content),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}, nil
}

// marshalDevenvYaml marshals the DevenvYaml struct into ordered YAML bytes.
// We marshal inputs in a deterministic order: nixpkgs first, then git-hooks,
// then remaining inputs sorted alphabetically.
func marshalDevenvYaml(dy DevenvYaml) ([]byte, error) {
	// Build ordered inputs as a yaml.Node to control key order.
	inputsMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	// nixpkgs always first.
	if inp, ok := dy.Inputs["nixpkgs"]; ok {
		addInputNode(inputsMapping, "nixpkgs", inp)
	}
	// git-hooks second, if present.
	if inp, ok := dy.Inputs["git-hooks"]; ok {
		addInputNode(inputsMapping, "git-hooks", inp)
	}
	// Remaining inputs in sorted order.
	var remaining []string
	for k := range dy.Inputs {
		if k != "nixpkgs" && k != "git-hooks" {
			remaining = append(remaining, k)
		}
	}
	sort.Strings(remaining)
	for _, k := range remaining {
		addInputNode(inputsMapping, k, dy.Inputs[k])
	}

	// Build the top-level document as an ordered mapping node.
	doc := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	addScalarPair(doc, "require_version", dy.RequireVersion)

	doc.Content = append(doc.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "inputs"},
		inputsMapping,
	)

	addBoolPair(doc, "impure", dy.Impure)
	addBoolPair(doc, "allow_unfree", dy.AllowUnfree)
	addBoolPair(doc, "allow_broken", dy.AllowBroken)

	addStringSeqPair(doc, "permitted_unfree_packages", dy.PermittedUnfreePackages)
	addStringSeqPair(doc, "permitted_insecure_packages", dy.PermittedInsecurePackages)

	// Clean section.
	cleanMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	addBoolPair(cleanMapping, "enabled", dy.Clean.Enabled)
	addStringSeqPair(cleanMapping, "keep", dy.Clean.Keep)
	doc.Content = append(doc.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "clean"},
		cleanMapping,
	)

	root := &yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{doc},
	}

	return yaml.Marshal(root)
}

func addScalarPair(mapping *yaml.Node, key, value string) {
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func addBoolPair(mapping *yaml.Node, key string, value bool) {
	v := "false"
	if value {
		v = "true"
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: v, Tag: "!!bool"},
	)
}

func addStringSeqPair(mapping *yaml.Node, key string, values []string) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, v := range values {
		seq.Content = append(seq.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: v},
		)
	}
	// Set flow style for empty sequences so they render as [].
	if len(values) == 0 {
		seq.Style = yaml.FlowStyle
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		seq,
	)
}

func addInputNode(mapping *yaml.Node, key string, inp DevenvYamlInput) {
	inputMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	addScalarPair(inputMapping, "url", inp.URL)
	if len(inp.Inputs) > 0 {
		subInputsMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		keys := make([]string, 0, len(inp.Inputs))
		for k := range inp.Inputs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sub := inp.Inputs[k]
			subMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			if sub.Follows != "" {
				addScalarPair(subMapping, "follows", sub.Follows)
			}
			subInputsMapping.Content = append(subInputsMapping.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				subMapping,
			)
		}
		inputMapping.Content = append(inputMapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "inputs"},
			subInputsMapping,
		)
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		inputMapping,
	)
}
