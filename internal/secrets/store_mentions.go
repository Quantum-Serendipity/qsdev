package secrets

import "strings"

// StoreMention is one "~/"-prefixed spelling of a credential store in a
// string.
type StoreMention struct {
	// Store is the credential store mentioned.
	Store CredentialStore
	// Whole reports that the mention names the store itself (or a glob over
	// it) rather than a sub-path inside it: "~/.ssh", "~/.ssh/" and
	// "~/.ssh/**" are whole, "~/.ssh/id_ed25519" is not.
	Whole bool
}

// HomeStoreMentions returns every mention of a CredentialStores entry in s
// as "~/"+Rel, in CredentialStores order and left to right within a store.
// A mention ends at the end of s or at a character that cannot continue a
// name (a space, quote, comma, '*', ')', '/', ...), so "~/.sshfoo" and
// "~/.config/ghq" mention no store. It is whole unless the store is followed
// by '/' and a name character, which makes it a sub-path.
//
// It is the one scanner for private copies of a store path: the archtest
// credential-store-literal rule and the module tests that keep the canon the
// single owner of the home credential stores both use it.
func HomeStoreMentions(s string) []StoreMention {
	if !strings.Contains(s, "~/") {
		return nil
	}
	var out []StoreMention
	for _, store := range CredentialStores() {
		needle := "~/" + store.Rel
		for rest := s; ; {
			i := strings.Index(rest, needle)
			if i < 0 {
				break
			}
			rest = rest[i+len(needle):]
			if rest != "" && continuesName(rest[0]) {
				continue
			}
			subPath := len(rest) > 1 && rest[0] == '/' && continuesName(rest[1])
			out = append(out, StoreMention{Store: store, Whole: !subPath})
		}
	}
	return out
}

// continuesName reports whether c can continue a path segment's name.
func continuesName(c byte) bool {
	return c == '.' || c == '-' || c == '_' ||
		('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9')
}
