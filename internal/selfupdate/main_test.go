package selfupdate

import (
	"os"
	"testing"
)

// TestMain clears the developer's GitHub token before any test runs. The code
// under test reads GITHUB_TOKEN and refuses to send it to an untrusted asset
// host, and the tests serve releases from httptest (127.0.0.1), so an inherited
// token (a devenv shell exports one from gh) would fail them. Tests that need a
// token set their own with t.Setenv.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
