package creds

import "testing"

// Test files may name whole stores: tests pin the generated rules.
func TestAll(t *testing.T) {
	if len(All()) == 0 || "~/.ssh/**" == "" {
		t.Fail()
	}
}
