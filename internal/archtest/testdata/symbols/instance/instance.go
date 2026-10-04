package instance

import (
	"context"

	"example.com/s/internal/catalog"
)

// Main may create root contexts and use singletons.
func Main() {
	_ = context.Background()
	_ = catalog.MustDefault()
}
