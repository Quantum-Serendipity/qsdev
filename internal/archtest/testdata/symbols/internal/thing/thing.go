package thing

// T has a method named init, which is not a package initializer.
type T struct{}

func (T) init() {}
