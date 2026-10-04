package devinit

import (
	"os"
	"testing"
)

// TestMain removes the shared lifecycle project template once every test has
// run.
func TestMain(m *testing.M) {
	code := m.Run()
	removeLifecycleTemplate()
	os.Exit(code)
}
