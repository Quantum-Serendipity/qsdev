// Package enumtext implements the String/MarshalText/UnmarshalText triple for
// int-backed enumerations whose values lie in the range 0..n-1, so each enum
// type declares its names once instead of hand-copying the logic.
package enumtext

import "fmt"

// Enum maps the values of an int-backed enumeration to their text names:
// Names[i] is the text form of value i. An empty name marks i as a
// non-member: it is how an enum declares an explicit Unknown zero value that
// must never be marshalled or parsed, so a forgotten field fails closed.
// Validity is decided by index range and non-empty name, never by comparing
// against the invalid sentinel, so a legitimate value may be named "unknown".
type Enum[T ~int] struct {
	typeName string
	label    string
	invalid  string
	names    []string
	byName   map[string]T
}

// New returns an Enum for T. typeName is the Go type name used in marshal
// errors, label the human-readable noun used in unmarshal errors (e.g.
// "hook event"), and invalid the String form of out-of-range values.
func New[T ~int](typeName, label, invalid string, names []string) Enum[T] {
	byName := make(map[string]T, len(names))
	for i, n := range names {
		if n != "" {
			byName[n] = T(i)
		}
	}
	return Enum[T]{typeName: typeName, label: label, invalid: invalid, names: names, byName: byName}
}

// Valid reports whether v is a declared value with a non-empty name.
func (e Enum[T]) Valid(v T) bool {
	return int(v) >= 0 && int(v) < len(e.names) && e.names[v] != ""
}

// String returns the name of v, or the invalid sentinel when v is not a member.
func (e Enum[T]) String(v T) string {
	if !e.Valid(v) {
		return e.invalid
	}
	return e.names[v]
}

// MarshalText returns the name of v, or an error when v is not a member.
func (e Enum[T]) MarshalText(v T) ([]byte, error) {
	if !e.Valid(v) {
		return nil, fmt.Errorf("cannot marshal %s %s value %d", e.invalid, e.typeName, int(v))
	}
	return []byte(e.names[v]), nil
}

// UnmarshalText parses a name into *dst, or returns an error for an unknown name.
func (e Enum[T]) UnmarshalText(text []byte, dst *T) error {
	v, ok := e.byName[string(text)]
	if !ok {
		return fmt.Errorf("unknown %s: %q", e.label, string(text))
	}
	*dst = v
	return nil
}
