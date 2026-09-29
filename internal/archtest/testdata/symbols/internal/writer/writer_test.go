package writer

import (
	"os"
	"testing"
)

func TestWrite(t *testing.T) {
	_ = os.WriteFile("f", nil, 0o644)
	_ = os.Chdir("a")
	_ = os.Chdir("b")
	panic("tests may panic")
}
