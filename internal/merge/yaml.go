package merge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"gopkg.in/yaml.v3"
)

// MergeYAML performs a key-level three-way merge of a YAML document whose
// root is a mapping (e.g. a generated docker-compose fragment).
//
// base is the last-generated content (nil when there is no recorded base),
// theirs the current on-disk content and ours the newly generated content.
// Mappings merge key by key, recursively; every other value (scalars and
// sequences) is merged as a whole:
//
//   - a value the user left as generated takes the generator's new value;
//   - a value the user changed keeps the user's version, also when the
//     generator changed it too (the user's edit wins the conflict);
//   - a generated key the user deleted stays deleted;
//   - a new generated key is added, and a generated key the generator no
//     longer produces is removed unless the user changed it;
//   - keys the user added are kept.
//
// Without a base every key present on disk counts as the user's own. The
// output is built from theirs, so the user's key order, comments and anchors
// survive; an unmodified file (theirs equal to base) is replaced by ours
// outright. Each input must be a single YAML document, and a merge result
// that no longer parses (e.g. an alias left without its anchor) is an error
// rather than a broken file.
func MergeYAML(base, theirs, ours []byte) ([]byte, error) {
	if len(base) > 0 && bytes.Equal(theirs, base) {
		if _, err := parseYAMLDocument("ours", ours); err != nil {
			return nil, err
		}
		return ours, nil
	}
	var baseMap *yaml.Node
	if len(base) > 0 {
		b, err := parseYAMLMapping("base", base)
		if err != nil {
			return nil, err
		}
		baseMap = b
	}
	theirsDoc, err := parseYAMLDocument("theirs", theirs)
	if err != nil {
		return nil, err
	}
	oursMap, err := parseYAMLMapping("ours", ours)
	if err != nil {
		return nil, err
	}

	mergeYAMLMapping(baseMap, theirsDoc.Content[0], oursMap)

	out, err := yaml.Marshal(theirsDoc)
	if err != nil {
		return nil, fmt.Errorf("marshaling merged yaml: %w", err)
	}
	if _, err := parseYAMLDocument("merged", out); err != nil {
		return nil, err
	}
	return out, nil
}

// parseYAMLDocument parses data as a single YAML document whose root is a
// mapping, returning the document node.
func parseYAMLDocument(label string, data []byte) (*yaml.Node, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parsing %s yaml: %w", label, err)
	}
	// A second document would be silently dropped on re-marshal.
	var next yaml.Node
	if err := dec.Decode(&next); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple documents are not supported")
		}
		return nil, fmt.Errorf("parsing %s yaml: %w", label, err)
	}
	// Resolve aliases now so a dangling one is reported here.
	var probe any
	if err := doc.Decode(&probe); err != nil {
		return nil, fmt.Errorf("parsing %s yaml: %w", label, err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parsing %s yaml: root is not a mapping", label)
	}
	return &doc, nil
}

// parseYAMLMapping parses data like parseYAMLDocument and returns the root
// mapping node.
func parseYAMLMapping(label string, data []byte) (*yaml.Node, error) {
	doc, err := parseYAMLDocument(label, data)
	if err != nil {
		return nil, err
	}
	return doc.Content[0], nil
}

// mergeYAMLMapping merges ours into theirs in place (see MergeYAML). base may
// be nil when no recorded value exists at this level.
func mergeYAMLMapping(base, theirs, ours *yaml.Node) {
	for i := 0; i+1 < len(ours.Content); i += 2 {
		key, oursVal := ours.Content[i].Value, ours.Content[i+1]
		baseVal := yamlMappingValue(base, key)
		idx := yamlMappingIndex(theirs, key)
		switch {
		case idx < 0 && baseVal != nil:
			// The user deleted a generated key: respect the deletion.
		case idx < 0:
			theirs.Content = append(theirs.Content, ours.Content[i], oursVal)
		case theirs.Content[idx+1].Kind == yaml.MappingNode && oursVal.Kind == yaml.MappingNode:
			if baseVal != nil && baseVal.Kind != yaml.MappingNode {
				baseVal = nil
			}
			mergeYAMLMapping(baseVal, theirs.Content[idx+1], oursVal)
		case baseVal != nil && yamlNodesEqual(theirs.Content[idx+1], baseVal):
			// Unchanged by the user: take the generator's value.
			theirs.Content[idx+1] = keepUserDecor(theirs.Content[idx+1], oursVal)
		}
		// Otherwise the user changed the value (or it has no base): keep it.
	}

	// Drop generated keys the generator no longer produces, unless edited.
	kept := theirs.Content[:0]
	for i := 0; i+1 < len(theirs.Content); i += 2 {
		key, val := theirs.Content[i].Value, theirs.Content[i+1]
		if baseVal := yamlMappingValue(base, key); baseVal != nil &&
			yamlMappingIndex(ours, key) < 0 && yamlNodesEqual(val, baseVal) {
			continue
		}
		kept = append(kept, theirs.Content[i], val)
	}
	theirs.Content = kept
}

// keepUserDecor returns a copy of the generated value gen carrying the
// anchor and comments of the user's node it replaces, so aliases elsewhere in
// the file still resolve and the user's comments survive.
func keepUserDecor(user, gen *yaml.Node) *yaml.Node {
	out := *gen
	if user.Anchor != "" {
		out.Anchor = user.Anchor
	}
	if user.HeadComment != "" {
		out.HeadComment = user.HeadComment
	}
	if user.LineComment != "" {
		out.LineComment = user.LineComment
	}
	if user.FootComment != "" {
		out.FootComment = user.FootComment
	}
	return &out
}

// yamlMappingIndex returns the index of key's key node in mapping m, or -1.
func yamlMappingIndex(m *yaml.Node, key string) int {
	if m == nil {
		return -1
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// yamlMappingValue returns key's value node in mapping m, or nil.
func yamlMappingValue(m *yaml.Node, key string) *yaml.Node {
	if i := yamlMappingIndex(m, key); i >= 0 {
		return m.Content[i+1]
	}
	return nil
}

// yamlNodesEqual reports whether two nodes decode to the same value, ignoring
// comments, quoting and layout. Nodes that fail to decode are unequal.
func yamlNodesEqual(a, b *yaml.Node) bool {
	var av, bv any
	if a.Decode(&av) != nil || b.Decode(&bv) != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}
