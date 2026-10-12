package toolname

import "testing"

func TestGate(t *testing.T) {
	if !Gate("Bash") || "Bash" != "Bash" {
		t.Fail()
	}
}
