package generate

import (
	"encoding/json"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// DeterministicJSON marshals v to JSON with sorted map keys, 2-space indent,
// and trailing newline.
func DeterministicJSON(v any) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON: %w", err)
	}
	return append(data, '\n'), nil
}

// DeterministicYAML marshals v to YAML with sorted map keys.
func DeterministicYAML(v any) ([]byte, error) {
	sorted := sortMapKeys(v)
	return yaml.Marshal(sorted)
}

// sortMapKeys recursively sorts map[string]any keys so YAML output
// is deterministic regardless of Go map iteration order.
func sortMapKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		node := &yaml.Node{
			Kind: yaml.MappingNode,
		}
		for _, k := range keys {
			node.Content = append(node.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: k},
				valueToNode(sortMapKeys(val[k])),
			)
		}
		return node

	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = sortMapKeys(elem)
		}
		return out

	default:
		return v
	}
}

// valueToNode converts a Go value to a yaml.Node via marshal round-trip.
// Returns an empty scalar on error as a defensive fallback.
func valueToNode(v any) *yaml.Node {
	if n, ok := v.(*yaml.Node); ok {
		return n
	}

	node := &yaml.Node{}
	// Let yaml.v3 figure out the encoding by round-tripping through Marshal
	// then decoding the node. For simple scalars this is the safest approach.
	data, err := yaml.Marshal(v)
	if err != nil {
		node.Kind = yaml.ScalarNode
		node.Value = ""
		return node
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Content) == 0 {
		node.Kind = yaml.ScalarNode
		node.Value = ""
		return node
	}
	return doc.Content[0]
}
