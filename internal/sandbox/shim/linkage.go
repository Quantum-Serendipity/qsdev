package shim

import (
	"debug/elf"
	"fmt"
	"strings"
)

// nixStore is the one host tree every namespace sandbox mounts.
const nixStore = "/nix/store/"

// maxInterpLen bounds the PT_INTERP read; a real interpreter path is far
// shorter than PATH_MAX.
const maxInterpLen = 4096

// LinkageHint explains why the binary at exe may fail to start inside a
// namespace sandbox: a dynamic ELF whose interpreter lies outside /nix/store
// cannot be loaded there. It returns "" when exe is static, uses a
// /nix/store interpreter, or is not a readable ELF file. It is a diagnostic
// for the not-started path only.
func LinkageHint(exe string) string {
	interp, err := elfInterp(exe)
	if err != nil {
		return ""
	}
	return linkageHint(exe, interp)
}

// linkageHint decides the hint for exe given its ELF interpreter ("" for a
// static binary).
func linkageHint(exe, interp string) string {
	if interp == "" || strings.HasPrefix(interp, nixStore) {
		return ""
	}
	return fmt.Sprintf("qsdev at %s is dynamically linked against %s, which is not visible inside the sandbox; rebuild with CGO_ENABLED=0", exe, interp)
}

// elfInterp returns the PT_INTERP path of the ELF file at path, or "" when
// it has none.
func elfInterp(path string) (string, error) {
	f, err := elf.Open(path)
	if err != nil {
		return "", fmt.Errorf("reading ELF %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	for _, p := range f.Progs {
		if p.Type != elf.PT_INTERP {
			continue
		}
		if p.Filesz > maxInterpLen {
			return "", fmt.Errorf("interpreter of %s is %d bytes", path, p.Filesz)
		}
		buf := make([]byte, p.Filesz)
		if _, err := p.ReadAt(buf, 0); err != nil {
			return "", fmt.Errorf("reading interpreter of %s: %w", path, err)
		}
		return strings.TrimRight(string(buf), "\x00"), nil
	}
	return "", nil
}
