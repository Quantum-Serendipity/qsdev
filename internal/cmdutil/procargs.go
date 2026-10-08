package cmdutil

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// The process-table parsers have no build constraint, so their tests run on
// every OS; humangate_linux.go and humangate_darwin.go feed them.

// parseStatPPID returns the ppid field of a Linux /proc/<pid>/stat line. The
// command name before it is parenthesised and may hold spaces and
// parentheses, so the fields start after the last ')'.
func parseStatPPID(stat []byte) (int, error) {
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 {
		return 0, fmt.Errorf("malformed /proc stat line %q", stat)
	}
	fields := bytes.Fields(stat[end+1:])
	if len(fields) < 2 {
		return 0, fmt.Errorf("malformed /proc stat line %q", stat)
	}
	ppid, err := strconv.Atoi(string(fields[1]))
	if err != nil {
		return 0, fmt.Errorf("parsing the ppid in %q: %w", stat, err)
	}
	return ppid, nil
}

// procArgs is what a process started with: its executable's path, its argv
// and its environment.
type procArgs struct {
	exe  string
	argv []string
	env  []string
}

// parseProcArgs2 reads a macOS kern.procargs2 buffer: a native-endian int32
// argc, the executable path, NUL padding, argc argument strings, then the
// environment strings, each NUL-terminated and ended by an empty string.
func parseProcArgs2(data []byte) (procArgs, error) {
	if len(data) < 4 {
		return procArgs{}, fmt.Errorf("short kern.procargs2 buffer (%d bytes)", len(data))
	}
	argc := int(int32(binary.NativeEndian.Uint32(data[:4])))
	if argc < 0 {
		return procArgs{}, fmt.Errorf("negative argc %d in kern.procargs2", argc)
	}
	exe := bytes.IndexByte(data[4:], 0)
	if exe < 0 {
		return procArgs{}, errors.New("kern.procargs2 has no executable path")
	}
	p := procArgs{exe: string(data[4 : 4+exe])}
	rest := bytes.TrimLeft(data[4+exe:], "\x00")
	for range argc {
		end := bytes.IndexByte(rest, 0)
		if end < 0 {
			return procArgs{}, fmt.Errorf("kern.procargs2 holds fewer than %d arguments", argc)
		}
		p.argv = append(p.argv, string(rest[:end]))
		rest = rest[end+1:]
	}
	for {
		end := bytes.IndexByte(rest, 0)
		if end <= 0 {
			return p, nil
		}
		p.env = append(p.env, string(rest[:end]))
		rest = rest[end+1:]
	}
}

// splitNUL splits a NUL-separated list, such as a Linux /proc/<pid>/environ
// or cmdline, dropping empty entries.
func splitNUL(data []byte) []string {
	return strings.FieldsFunc(string(data), func(r rune) bool { return r == 0 })
}
