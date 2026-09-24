package sectools

import "testing"

func TestParseGitHubRemote(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		remote string
		want   gitHubRepo
		wantOK bool
	}{
		{"https", "https://github.com/acme/widget.git", gitHubRepo{"acme", "widget"}, true},
		{"https no suffix", "https://github.com/acme/widget", gitHubRepo{"acme", "widget"}, true},
		{"https trailing slash", "https://github.com/acme/widget/", gitHubRepo{"acme", "widget"}, true},
		{"https uppercase host", "https://GitHub.com/Acme/Widget.git", gitHubRepo{"Acme", "Widget"}, true},
		{"scp", "git@github.com:acme/widget.git", gitHubRepo{"acme", "widget"}, true},
		{"scp no user", "github.com:acme/widget", gitHubRepo{"acme", "widget"}, true},
		{"ssh with port", "ssh://git@github.com:22/acme/widget.git", gitHubRepo{"acme", "widget"}, true},
		{"git protocol", "git://github.com/acme/widget.git", gitHubRepo{"acme", "widget"}, true},
		{"dotted repo", "https://github.com/acme/widget.js.git", gitHubRepo{"acme", "widget.js"}, true},
		{"surrounding space", "  https://github.com/acme/widget.git\n", gitHubRepo{"acme", "widget"}, true},

		{"empty", "", gitHubRepo{}, false},
		{"gitlab", "git@gitlab.com:acme/widget.git", gitHubRepo{}, false},
		{"enterprise host", "https://github.acme.com/acme/widget.git", gitHubRepo{}, false},
		{"lookalike host", "https://github.com.evil.example/acme/widget.git", gitHubRepo{}, false},
		{"github in path", "https://evil.example/github.com/acme/widget.git", gitHubRepo{}, false},
		{"owner only", "https://github.com/acme", gitHubRepo{}, false},
		{"nested path", "https://github.com/acme/widget/tree/main", gitHubRepo{}, false},
		{"dot-dot repo", "https://github.com/acme/..", gitHubRepo{}, false},
		{"bad owner", "https://github.com/ac_me/widget.git", gitHubRepo{}, false},
		{"quote in repo", "https://github.com/acme/wid\"get.git", gitHubRepo{}, false},
		{"newline in repo", "git@github.com:acme/wid\nget.git", gitHubRepo{}, false},
		{"local path", "/srv/git/widget.git", gitHubRepo{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseGitHubRemote(tt.remote)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("parseGitHubRemote(%q) = %+v, %v; want %+v, %v", tt.remote, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestCosignPolicyName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		repo gitHubRepo
		want string
	}{
		{gitHubRepo{"acme", "widget"}, "acme-widget-keyless"},
		{gitHubRepo{"Acme", "Widget_API"}, "acme-widget-api-keyless"},
		{gitHubRepo{"acme", "widget.js"}, "acme-widget-js-keyless"},
		{gitHubRepo{"acme", "_widget_"}, "acme-widget-keyless"},
		{gitHubRepo{"acme", "a-.b"}, "acme-a-b-keyless"},
		{gitHubRepo{"acme", "a..b"}, "acme-a-b-keyless"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := cosignPolicyName(tt.repo); got != tt.want {
				t.Errorf("cosignPolicyName(%+v) = %q, want %q", tt.repo, got, tt.want)
			}
		})
	}
}
