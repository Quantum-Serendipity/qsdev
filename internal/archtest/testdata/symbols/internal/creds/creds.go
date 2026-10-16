package creds

// Planted copies of whole credential stores: a file store, a directory glob,
// a Bash deny rule, a raw literal, a store followed by a space and a store
// followed by a quote. A comment naming ~/.ssh is ignored.
var planted = []string{
	"~/.docker/config.json",
	"~/.ssh/**",
	"Bash(cat ~/.kube/*)",
	`Read(~/.npmrc)`,
	"ls ~/.ssh && true",
	"~/.npmrc\"",
}

// Allowed: narrower sub-paths, project-relative names, a $HOME locator
// script, a longer sibling directory and a longer file name.
var allowed = []string{
	"~/.aws/credentials",
	"~/.config/gcloud/credentials.db",
	"cat ~/.ssh/id_rsa",
	".npmrc",
	"./.npmrc",
	`ln -sf "$HOME/.config/composer/auth.json" auth.json`,
	"~/.sshfoo/x",
	"~/.yarnrc.yml.bak",
}

// All returns every fixture literal.
func All() []string { return append(planted, allowed...) }
