package types

import (
	"encoding/base64"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// blobLineLen is the base64 line length of an encoded Blob (the MIME limit).
const blobLineLen = 76

// Blob is file content kept in a state file. yaml.v3 encodes a plain []byte
// as a sequence of integers, one line per byte, which made state files tens
// of thousands of lines long and slow to load. A Blob encodes as one base64
// !!binary scalar instead. Base64 is used even for text because yaml.v3's
// emitter does not round-trip every string exactly (a literal block that
// starts with blank lines loses one), and the three-way merge base must come
// back byte for byte. Decoding also accepts the legacy integer sequence, so
// state files written before the change still load.
//
// Blob has []byte as its underlying type, so a []byte assigns to it and
// back without conversion, and encoding/json still encodes it as base64.
type Blob []byte

// MarshalYAML implements yaml.Marshaler. The base64 text is wrapped into
// lines of blobLineLen in a literal block; the decoder ignores line breaks.
func (b Blob) MarshalYAML() (any, error) {
	enc := base64.StdEncoding.EncodeToString(b)
	if len(enc) <= blobLineLen {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: enc}, nil
	}
	var sb strings.Builder
	sb.Grow(len(enc) + len(enc)/blobLineLen + 1)
	for len(enc) > 0 {
		n := min(blobLineLen, len(enc))
		sb.WriteString(enc[:n])
		sb.WriteByte('\n')
		enc = enc[n:]
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!binary", Value: sb.String(), Style: yaml.LiteralStyle}, nil
}

// UnmarshalYAML implements yaml.Unmarshaler. It accepts a !!binary scalar, a
// plain string scalar (taken as the content itself), null, and the legacy
// sequence of byte values.
func (b *Blob) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.SequenceNode:
		var legacy []byte
		if err := node.Decode(&legacy); err != nil {
			return fmt.Errorf("decoding legacy byte list: %w", err)
		}
		*b = legacy
	case yaml.ScalarNode:
		if node.ShortTag() == "!!null" {
			*b = nil
			return nil
		}
		// yaml.v3 base64-decodes a !!binary scalar into a string target.
		var s string
		if err := node.Decode(&s); err != nil {
			return fmt.Errorf("decoding content: %w", err)
		}
		*b = Blob(s)
	default:
		return fmt.Errorf("line %d: content must be a !!binary scalar or a byte list, got YAML node kind %d", node.Line, node.Kind)
	}
	return nil
}
