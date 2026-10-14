// Package secrets owns the credential canon, so it may spell every store.
package secrets

// Homes names whole stores; the canon owner is exempt.
var Homes = []string{"~/.ssh", "~/.docker/config.json", "~/.kube/**"}
