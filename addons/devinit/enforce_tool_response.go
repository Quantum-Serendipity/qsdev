package devinit

import (
	"encoding/json"
	"fmt"
)

// hardenToolResponse passes each text a PostToolUse tool_response carries
// through harden and rebuilds the response in the shape the tool produced:
//
//   - a bare string stays a string;
//   - an MCP content array (bare, or under the "content" key of a result
//     object) has every text block hardened on its own, in place, keeping the
//     block's other fields, every non-text block, and the result object's
//     other fields such as structuredContent;
//   - a single {"type":"text"} block keeps its other fields;
//   - any other JSON is hardened as its JSON text and returned as a string.
//
// Hardening each text block separately keeps a block that is a JSON document
// apart from its neighbours, so the datamark transform can recognize it and
// keep it parseable. changed is false when harden altered nothing, including
// when the response carries no text, such as an image-only MCP result.
func hardenToolResponse(raw json.RawMessage, harden func(string) string) (out json.RawMessage, changed bool, err error) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return hardenString(s, harden)
	}

	if blocks, wrapper, ok := responseContentBlocks(raw); ok {
		return hardenContentBlocks(blocks, wrapper, harden)
	}

	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) == nil {
		if _, isText := textBlockText(obj); isText {
			changed, err := hardenTextBlock(obj, harden)
			if err != nil || !changed {
				return nil, false, err
			}
			out, err := json.Marshal(obj)
			if err != nil {
				return nil, false, fmt.Errorf("encoding hardened text block: %w", err)
			}
			return out, true, nil
		}
	}

	return hardenString(string(raw), harden)
}

// hardenString hardens s and encodes the result as a JSON string.
func hardenString(s string, harden func(string) string) (json.RawMessage, bool, error) {
	hardened := harden(s)
	if hardened == s {
		return nil, false, nil
	}
	out, err := json.Marshal(hardened)
	if err != nil {
		return nil, false, fmt.Errorf("encoding hardened text: %w", err)
	}
	return out, true, nil
}

// hardenContentBlocks hardens every text block of an MCP content array in
// place and re-encodes the array, under wrapper's "content" key when the
// array came from a result object.
func hardenContentBlocks(blocks []map[string]json.RawMessage, wrapper map[string]json.RawMessage, harden func(string) string) (json.RawMessage, bool, error) {
	changed := false
	for _, b := range blocks {
		c, err := hardenTextBlock(b, harden)
		if err != nil {
			return nil, false, err
		}
		changed = changed || c
	}
	if !changed {
		return nil, false, nil
	}

	content, err := json.Marshal(blocks)
	if err != nil {
		return nil, false, fmt.Errorf("encoding hardened content blocks: %w", err)
	}
	if wrapper == nil {
		return content, true, nil
	}
	wrapper["content"] = content
	out, err := json.Marshal(wrapper)
	if err != nil {
		return nil, false, fmt.Errorf("encoding hardened tool result: %w", err)
	}
	return out, true, nil
}

// hardenTextBlock replaces the text of block, when it is a text block, with its
// hardened form. It reports whether the text changed; a non-text block is left
// alone.
func hardenTextBlock(block map[string]json.RawMessage, harden func(string) string) (bool, error) {
	text, isText := textBlockText(block)
	if !isText {
		return false, nil
	}
	hardened := harden(text)
	if hardened == text {
		return false, nil
	}
	encoded, err := json.Marshal(hardened)
	if err != nil {
		return false, fmt.Errorf("encoding hardened text block: %w", err)
	}
	block["text"] = encoded
	return true, nil
}

// responseContentBlocks decodes raw as an MCP content-block array, either bare
// or under the "content" key of an object. wrapper is that object (nil for a
// bare array). An array holding an element that is not an MCP content block is
// not a content array, so arbitrary JSON rows are hardened as JSON rather than
// skipped as blocks without text.
func responseContentBlocks(raw json.RawMessage) (blocks []map[string]json.RawMessage, wrapper map[string]json.RawMessage, ok bool) {
	if err := json.Unmarshal(raw, &blocks); err == nil {
		return blocks, nil, allContentBlocks(blocks)
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, nil, false
	}
	content, has := wrapper["content"]
	if !has {
		return nil, nil, false
	}
	if err := json.Unmarshal(content, &blocks); err != nil || !allContentBlocks(blocks) {
		return nil, nil, false
	}
	return blocks, wrapper, true
}

// mcpContentTypes are the content-block types the MCP specification defines
// for a tool result.
var mcpContentTypes = map[string]bool{
	"text":          true,
	"image":         true,
	"audio":         true,
	"resource":      true,
	"resource_link": true,
}

// allContentBlocks reports whether every block is a well-formed MCP content
// block: its type is one the specification defines, and a text block carries
// its text as a string. A malformed text block would otherwise be skipped as
// a non-text block and reach the model unhardened.
func allContentBlocks(blocks []map[string]json.RawMessage) bool {
	for _, b := range blocks {
		var typ string
		if json.Unmarshal(b["type"], &typ) != nil || !mcpContentTypes[typ] {
			return false
		}
		if _, isText := textBlockText(b); typ == "text" && !isText {
			return false
		}
	}
	return true
}

// textBlockText returns the text of an MCP {"type":"text"} content block and
// reports whether block is one.
func textBlockText(block map[string]json.RawMessage) (string, bool) {
	var typ, text string
	if json.Unmarshal(block["type"], &typ) != nil || typ != "text" {
		return "", false
	}
	if json.Unmarshal(block["text"], &text) != nil {
		return "", false
	}
	return text, true
}
