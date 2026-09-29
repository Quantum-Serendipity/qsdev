package types

import (
	osx "os"

	"example.com/m/internal/enumtext"
)

// Name mentions os.WriteFile in a comment only.
func Name() string { _ = osx.Args; return enumtext.Name }
