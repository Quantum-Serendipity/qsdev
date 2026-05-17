package cmdutil

import (
	"os"
	"testing"
)

func TestProjectRoot(t *testing.T) {
	got, err := ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot() error = %v", err)
	}

	want, _ := os.Getwd()
	if got != want {
		t.Errorf("ProjectRoot() = %q, want %q", got, want)
	}
}
