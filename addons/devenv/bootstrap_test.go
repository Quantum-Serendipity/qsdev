package devenv_test

import (
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devenv"
)

func TestInstallDevenvStep_NotNil(t *testing.T) {
	step := devenv.InstallDevenvStep()
	if step == nil {
		t.Fatal("InstallDevenvStep() returned nil")
	}
}

func TestInstallDirenvStep_NotNil(t *testing.T) {
	step := devenv.InstallDirenvStep()
	if step == nil {
		t.Fatal("InstallDirenvStep() returned nil")
	}
}
