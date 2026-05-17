package cmdutil

import (
	"fmt"
	"os"
)

// ProjectRoot returns the current working directory, wrapping any error
// with a consistent message.
func ProjectRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determining project root: %w", err)
	}
	return root, nil
}
