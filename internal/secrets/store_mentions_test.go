package secrets

import (
	"slices"
	"testing"
)

func TestHomeStoreMentions(t *testing.T) {
	t.Parallel()
	type m struct {
		rel   string
		whole bool
	}
	tests := []struct {
		in   string
		want []m
	}{
		{"~/.ssh", []m{{".ssh", true}}},
		{"~/.ssh/", []m{{".ssh", true}}},
		{"~/.ssh/**", []m{{".ssh", true}}},
		{"~/.ssh*", []m{{".ssh", true}}},
		{"Bash(cat ~/.kube/*)", []m{{".kube", true}}},
		{"Read(~/.npmrc)", []m{{".npmrc", true}}},
		{"~/.docker/config.json", []m{{".docker/config.json", true}}},
		{"ls ~/.ssh && true", []m{{".ssh", true}}},
		{`~/.npmrc"`, []m{{".npmrc", true}}},
		{"~/.netrc, ~/.pgpass", []m{{".netrc", true}, {".pgpass", true}}},
		{"cat ~/.ssh/id_rsa", []m{{".ssh", false}}},
		{"Bash(cat ~/.aws/credentials*)", []m{{".aws", false}}},
		{"~/.aws/sso/cache/*", []m{{".aws", false}}},
		{"~/.aws/credentials ~/.kube/*", []m{{".aws", false}, {".kube", true}}},
		{"~/.ssh/id_rsa ~/.ssh", []m{{".ssh", false}, {".ssh", true}}},
		{"~/.sshfoo", nil},
		{"~/.sshfoo/x", nil},
		{"~/.config/ghq", nil},
		{"~/.npmrc.bak", nil},
		{"~/.yarnrc.yml.bak", nil},
		{"~/.cargo/credentials.toml", []m{{".cargo/credentials.toml", true}}},
		{".npmrc", nil},
		{"./.npmrc", nil},
		{"$HOME/.npmrc", nil},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			var got []m
			for _, x := range HomeStoreMentions(tt.in) {
				got = append(got, m{x.Store.Rel, x.Whole})
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("HomeStoreMentions(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestHomeStoreMentions_CarriesStore pins that a mention carries the canon
// entry itself, Dir flag included.
func TestHomeStoreMentions_CarriesStore(t *testing.T) {
	t.Parallel()
	got := HomeStoreMentions("~/.config/gh/")
	want := []StoreMention{{Store: CredentialStore{Rel: ".config/gh", Dir: true}, Whole: true}}
	if !slices.Equal(got, want) {
		t.Errorf("HomeStoreMentions = %v, want %v", got, want)
	}
}
