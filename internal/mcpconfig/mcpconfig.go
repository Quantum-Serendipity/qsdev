// Package mcpconfig is the single schema for a project's .mcp.json, the file
// Claude Code reads its project MCP servers from. It only parses: deciding
// whether an entry may be started or dialed is mcpregistry's job. The
// registry reader, the three-way merge and the claudecode generator all use
// these types.
//
// It is a foundation package and imports only the standard library.
package mcpconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// FileName is the project-relative name of the MCP server config file.
const FileName = ".mcp.json"

// Transport values Claude Code assigns when an entry's type is implicit.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// File is the parsed .mcp.json document.
type File struct {
	MCPServers map[string]Server `json:"mcpServers"`
}

// Server is one .mcp.json entry. Stdio servers use Command, Args and Env;
// remote servers use Type ("http" or "sse") with URL and optional Headers.
type Server struct {
	Type    string            `json:"type,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Transport returns the entry's transport as Claude Code resolves it: the
// explicit type when set, otherwise "http" for an entry with a URL and
// "stdio" for one without.
func (s Server) Transport() string {
	switch {
	case s.Type != "":
		return s.Type
	case s.URL != "":
		return TransportHTTP
	default:
		return TransportStdio
	}
}

// Read parses root/.mcp.json. A missing file yields an empty File and no
// error; any other read or parse failure is returned.
func Read(root string) (File, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return File{}, nil
		}
		return File{}, fmt.Errorf("reading %s: %w", FileName, err)
	}
	return Parse(data)
}

// Parse decodes .mcp.json content, such as generated output not yet on disk.
//
// encoding/json matches object keys to fields case-insensitively, while
// Claude Code matches them exactly and ignores the rest. A document that
// spells a field two ways ({"command":"true","Command":"x"}) would therefore
// mean one launch to qsdev and another to Claude Code, so Parse rejects any
// key that differs from a field name only by case (ErrKeyCase) instead of
// guessing which spelling wins.
//
// Likewise, encoding/json merges the objects of a key given twice (two
// "mcpServers" objects, or two "env" maps of one entry) while JSON.parse keeps
// only the last, so Parse rejects a repeated key at the top level, among the
// server names and within an entry (ErrDuplicateKey).
func Parse(data []byte) (File, error) {
	if err := checkKeys(data); err != nil {
		return File{}, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parsing %s: %w", FileName, err)
	}
	return f, nil
}

// ErrKeyCase reports a .mcp.json key that differs from a field name only by
// case, which Claude Code would ignore but encoding/json would read.
var ErrKeyCase = errors.New("key differs from a field name only by case")

// ErrDuplicateKey reports a .mcp.json object that repeats a key, whose values
// encoding/json would merge but Claude Code would replace.
var ErrDuplicateKey = errors.New("key appears more than once")

// Field names of File and Server, read from their json tags so the check
// follows the schema.
var (
	fileFields   = jsonFieldNames(reflect.TypeFor[File]())
	serverFields = jsonFieldNames(reflect.TypeFor[Server]())
)

// member is one key and raw value of a JSON object, in document order.
type member struct {
	key   string
	value json.RawMessage
}

// checkKeys returns ErrKeyCase or ErrDuplicateKey when the document's top
// level, its server names or any server entry has a key that case-folds to a
// field name without being it, or a repeated key. Content that is not the
// expected shape is left for json.Unmarshal to reject or accept exactly as
// before.
func checkKeys(data []byte) error {
	top, ok := objectMembers(data)
	if !ok {
		return nil
	}
	if err := checkMembers(top, fileFields, ""); err != nil {
		return err
	}
	i := slices.IndexFunc(top, func(m member) bool { return m.key == "mcpServers" })
	if i < 0 {
		return nil
	}
	servers, ok := objectMembers(top[i].value)
	if !ok {
		return nil
	}
	if err := checkMembers(servers, nil, "mcpServers"); err != nil {
		return err
	}
	for _, s := range servers {
		entry, ok := objectMembers(s.value)
		if !ok {
			continue
		}
		if err := checkMembers(entry, serverFields, "server "+strconv.Quote(s.key)); err != nil {
			return err
		}
	}
	return nil
}

// objectMembers returns the members of the JSON object data in document
// order, keeping repeated keys, or false when data is not a valid object.
func objectMembers(data []byte) ([]member, bool) {
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, false
	}
	var out []member
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, false
		}
		key, ok := tok.(string)
		if !ok {
			return nil, false
		}
		var value json.RawMessage
		if dec.Decode(&value) != nil {
			return nil, false
		}
		out = append(out, member{key: key, value: value})
	}
	return out, true
}

// checkMembers returns ErrDuplicateKey for the first repeated key of obj,
// else ErrKeyCase for the first key that equals one of fields under Unicode
// case folding but is not spelled exactly as it. where prefixes the error.
func checkMembers(obj []member, fields []string, where string) error {
	seen := make(map[string]bool, len(obj))
	for _, m := range obj {
		if seen[m.key] {
			return locate(where, fmt.Errorf("%w: %q", ErrDuplicateKey, m.key))
		}
		seen[m.key] = true
	}
	for _, m := range obj {
		for _, field := range fields {
			if m.key != field && strings.EqualFold(m.key, field) {
				return locate(where, fmt.Errorf("%w: %q is not %q", ErrKeyCase, m.key, field))
			}
		}
	}
	return nil
}

// locate prefixes err with where, the object it was found in, when set.
func locate(where string, err error) error {
	if where == "" {
		return err
	}
	return fmt.Errorf("%s: %w", where, err)
}

// jsonFieldNames returns the json names of t's fields.
func jsonFieldNames(t reflect.Type) []string {
	names := make([]string, 0, t.NumField())
	for f := range t.Fields() {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" {
			name = f.Name
		}
		names = append(names, name)
	}
	return names
}
