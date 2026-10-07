package mcpserve

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// readTokenFile returns the token a client would read from the published file.
func readTokenFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the token file: %v", err)
	}
	return strings.TrimSpace(string(raw))
}

// postStatus sends POST url with an optional bearer token and returns the status.
func postStatus(t *testing.T, h http.Handler, token string) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, mcpEndpointPath, strings.NewReader("{}"))
	req.Host = "127.0.0.1:8765"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// TestHTTPLoopback_RequiresToken (U21-04): plain loopback HTTP answers 401 to
// a request without the per-launch token or with a wrong one, and lets the
// token read from the published file through. The file is private to the
// user: 0600 in a 0700 directory (Windows relies on the per-user state
// directory's ACL instead).
func TestHTTPLoopback_RequiresToken(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "mcp", "8765.token")
	tok, err := newHTTPToken(path)
	if err != nil {
		t.Fatalf("newHTTPToken: %v", err)
	}
	if err := tok.publish(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8765}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	t.Cleanup(tok.remove)

	stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := httpHandler(stub, nil, tok.value)

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"no token", "", http.StatusUnauthorized},
		{"wrong token", strings.Repeat("0", len(tok.value)), http.StatusUnauthorized},
		{"short token", tok.value[:8], http.StatusUnauthorized},
		{"token from file", readTokenFile(t, path), http.StatusOK},
	}
	for _, c := range cases {
		if got := postStatus(t, h, c.token); got != c.want {
			t.Errorf("%s: status = %d, want %d", c.name, got, c.want)
		}
	}

	req := httptest.NewRequest(http.MethodPost, mcpEndpointPath, nil)
	req.Host = "127.0.0.1:8765"
	req.Header.Set("Authorization", "Basic "+tok.value)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("non-bearer scheme: status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer") {
		t.Errorf("401 WWW-Authenticate = %q, want a Bearer challenge", got)
	}

	if runtime.GOOS == "windows" {
		return
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("token file mode = %o, want 600", perm)
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("token directory mode = %o, want 700", perm)
	}
}

// TestHTTPToken_RemovedOnShutdown: a real plain-HTTP server publishes its
// token once it listens, enforces it end to end, and removes the file when it
// shuts down.
func TestHTTPToken_RemovedOnShutdown(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "mcp", "srv.token")
	tok, err := newHTTPToken(path)
	if err != nil {
		t.Fatalf("newHTTPToken: %v", err)
	}
	srv := New(WithProjectRoot(t.TempDir()))
	addr := freeLoopbackAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ServeHTTP(ctx, addr, nil, tok) }()

	url := "http://" + addr + mcpEndpointPath
	waitForHTTP(t, url)
	client := &http.Client{Timeout: 10 * time.Second}
	resp := mcpPost(t, client, url, "", map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("POST without token: status = %d, want 401", resp.StatusCode)
	}
	authed := &http.Client{Timeout: 10 * time.Second, Transport: bearerTransport{token: readTokenFile(t, path)}}
	mcpInitialize(t, authed, url) // fails the test unless 200

	cancel()
	select {
	case <-serveErr:
	case <-time.After(httpShutdownTimeout + 5*time.Second):
		t.Fatal("ServeHTTP did not return after cancellation")
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("token file still present after shutdown (stat err %v)", err)
	}
}

// bearerTransport adds a bearer token to every request.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// TestHTTPToken_DefaultPathUnderUserState: without --http-token-file the token
// lives in the per-user state directory, under mcp/<port>.token.
func TestHTTPToken_DefaultPathUnderUserState(t *testing.T) {
	state := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	t.Setenv("XDG_STATE_HOME", state)

	got, err := defaultHTTPTokenPath(8765)
	if err != nil {
		t.Fatalf("defaultHTTPTokenPath: %v", err)
	}
	want := filepath.Join(state, branding.Get().AppName, "mcp", "8765.token")
	if got != want {
		t.Errorf("defaultHTTPTokenPath = %q, want %q", got, want)
	}
}

// userStateEnv points the per-user directories at fresh temp dirs and
// returns the state directory.
func userStateEnv(t *testing.T) string {
	t.Helper()
	state := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("LOCALAPPDATA", state)
	return state
}

// TestHTTPToken_DefaultPathNamesBoundPort: without --http-token-file, a server
// asked for --port 0 publishes under the port the kernel bound, not 0, so a
// client finds the file from the address it connects to, and two such
// servers never share (and remove) one file.
func TestHTTPToken_DefaultPathNamesBoundPort(t *testing.T) {
	state := userStateEnv(t)
	tok, err := resolveHTTPToken(container.DeployNative, TransportHTTP, false, serveOptions{port: 0})
	if err != nil {
		t.Fatalf("resolveHTTPToken: %v", err)
	}
	srv := New(WithProjectRoot(t.TempDir()))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ServeHTTP(ctx, "127.0.0.1:0", nil, tok) }()

	var path string
	deadline := time.Now().Add(10 * time.Second)
	for {
		// tok.path is the serving goroutine's until it returns, so the file
		// is found by listing the directory instead.
		matches, _ := filepath.Glob(filepath.Join(state, branding.Get().AppName, "mcp", "*.token"))
		if len(matches) == 1 {
			path = matches[0]
			break
		}
		select {
		case err := <-serveErr:
			t.Fatalf("ServeHTTP returned early: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("the token file was never published")
		}
		time.Sleep(10 * time.Millisecond)
	}
	port := strings.TrimSuffix(filepath.Base(path), ".token")
	if port == "0" {
		t.Fatalf("token published at %s, which names the requested port 0, not the bound one", path)
	}
	url := "http://127.0.0.1:" + port + mcpEndpointPath
	authed := &http.Client{Timeout: 10 * time.Second, Transport: bearerTransport{token: readTokenFile(t, path)}}
	mcpInitialize(t, authed, url) // the file names the port the server answers on
	cancel()
	<-serveErr
}

// TestHTTPToken_DefaultDirTightened: the default token directory is qsdev's,
// so publish makes it 0700 even when it already existed with a looser mode.
func TestHTTPToken_DefaultDirTightened(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows file permissions are ACLs, not modes")
	}
	state := userStateEnv(t)
	dir := filepath.Join(state, branding.Get().AppName, "mcp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tok, err := newHTTPToken("")
	if err != nil {
		t.Fatal(err)
	}
	if err := tok.publish(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8765}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	t.Cleanup(tok.remove)
	if want := filepath.Join(dir, "8765.token"); tok.path != want {
		t.Errorf("token path = %q, want %q", tok.path, want)
	}
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := di.Mode().Perm(); perm != 0o700 {
		t.Errorf("pre-existing token directory mode = %o after publish, want 700", perm)
	}
}

// TestHTTPToken_CustomDirLeftAlone: a --http-token-file directory is the
// operator's, so publish neither fails on nor changes a looser one (it
// warns), and the file itself is still 0600.
func TestHTTPToken_CustomDirLeftAlone(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows file permissions are ACLs, not modes")
	}
	dir := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o775); err != nil {
		t.Fatal(err)
	}
	tok, err := newHTTPToken(filepath.Join(dir, "mcp.token"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tok.publish(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8765}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	t.Cleanup(tok.remove)
	if di, _ := os.Stat(dir); di.Mode().Perm() != 0o775 {
		t.Errorf("custom directory mode = %o, want it left at 775", di.Mode().Perm())
	}
	if fi, _ := os.Stat(tok.path); fi.Mode().Perm() != 0o600 {
		t.Errorf("token file mode = %o, want 600", fi.Mode().Perm())
	}
}
