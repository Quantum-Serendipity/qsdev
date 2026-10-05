package mcpserve

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// httpTokenBytes is the size of the per-launch bearer token, before hex
// encoding.
const httpTokenBytes = 32

// httpTokenDirMode keeps the token directory private to the user.
const httpTokenDirMode os.FileMode = 0o700

// httpToken is the bearer token a plain loopback HTTP server requires of every
// caller (U21-04). It is generated per launch, and published to a file only
// the user can read, so a client running as the user can authenticate while
// another local account, or a browser page, cannot.
type httpToken struct {
	// value is the token, hex-encoded.
	value string
	// path is the file it is published to; empty until publish when it is
	// the default, which names the port the listener actually bound.
	path string
}

// newHTTPToken generates a token to be published at path, or at the default
// path for the bound port (see defaultHTTPTokenPath) when path is empty.
func newHTTPToken(path string) (*httpToken, error) {
	b := make([]byte, httpTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generating the HTTP bearer token: %w", err)
	}
	return &httpToken{value: hex.EncodeToString(b), path: path}, nil
}

// defaultHTTPTokenPath is where the token of a server on port is published
// when --http-token-file is unset: mcp/<port>.token in the per-user state
// directory, which self-protection guards.
func defaultHTTPTokenPath(port int) (string, error) {
	dirs, err := projectctx.UserDirs()
	if err != nil {
		return "", fmt.Errorf("resolving the state directory for the HTTP bearer token: %w", err)
	}
	return filepath.Join(dirs.State, "mcp", strconv.Itoa(port)+".token"), nil
}

// publish writes the token to its file, mode 0600, once the listener is
// bound to addr. Only the path is logged, never the token.
//
// Without --http-token-file the file is the default one for addr's port (so
// --port 0 publishes under the port the kernel chose, and two such servers
// never share a file), and its directory, which qsdev owns, is made 0700
// even when it already existed. A custom file's directory is the operator's:
// it is created 0700 when missing, and a warning names one that other users
// can write, since they could replace the file.
func (t *httpToken) publish(addr net.Addr) error {
	ownDir := t.path == ""
	if ownDir {
		tcp, ok := addr.(*net.TCPAddr)
		if !ok {
			return fmt.Errorf("resolving the HTTP bearer token path: listener address %v is not TCP", addr)
		}
		path, err := defaultHTTPTokenPath(tcp.Port)
		if err != nil {
			return err
		}
		t.path = path
	}
	dir := filepath.Dir(t.path)
	if err := os.MkdirAll(dir, httpTokenDirMode); err != nil {
		return fmt.Errorf("creating the HTTP bearer token directory: %w", err)
	}
	if err := checkTokenDir(dir, ownDir); err != nil {
		return err
	}
	if err := fileutil.WriteFileAtomic(t.path, []byte(t.value+"\n"), fileutil.ModePrivate); err != nil {
		return fmt.Errorf("writing the HTTP bearer token: %w", err)
	}
	// The attribute is "path", not a name with "token" in it, which the log
	// redaction would hide.
	slog.Info("plain HTTP requires the per-launch bearer credential; clients read it from the file at path",
		"path", t.path)
	return nil
}

// checkTokenDir makes the token directory dir private when qsdev owns it
// (ownDir), or warns when another user could write to it. Windows permissions
// are ACLs, which a mode cannot express; there the per-user state directory's
// ACL keeps the file private.
func checkTokenDir(dir string, ownDir bool) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if ownDir {
		if err := os.Chmod(dir, httpTokenDirMode); err != nil {
			return fmt.Errorf("restricting the HTTP bearer token directory: %w", err)
		}
		return nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("checking the HTTP bearer token directory: %w", err)
	}
	if fi.Mode().Perm()&0o022 != 0 {
		slog.Warn("the HTTP bearer credential file is in a directory other users can write to; they could replace it",
			"dir", dir, "mode", fmt.Sprintf("%#o", fi.Mode().Perm()))
	}
	return nil
}

// remove deletes the published token file. A file that is already gone is
// fine; any other failure is logged, since the server is stopping anyway.
func (t *httpToken) remove() {
	if err := os.Remove(t.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		slog.Warn("could not remove the HTTP bearer credential file", "path", t.path, "error", err)
	}
}
