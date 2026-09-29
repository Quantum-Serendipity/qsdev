package user

import (
	cat "example.com/s/internal/catalog"
)

type registry struct{}

func (registry) MustDefault() {}

// Use reaches the singleton through an alias.
func Use() {
	_ = cat.MustDefault()
	var catalog registry
	catalog.MustDefault()
}
