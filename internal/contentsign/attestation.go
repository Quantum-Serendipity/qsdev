package contentsign

import (
	"context"
	"os/exec"
	"path/filepath"
)

// AttestationStore answers whether a binary has a verified external attestation:
// a detached Minisign signature (<path>.minisig) from one of the trusted keys.
type AttestationStore struct {
	TrustedKeys []PublicKey // when empty, keys are loaded from DefaultTrustedKeysDir
}

// IsAttested reports whether the binary identified by command has a valid
// signature from a trusted key. A relative command is resolved via PATH. Any
// failure (unresolvable command, missing signature, bad signature, no trusted
// keys) yields false — attestation is opt-in and absence is simply "not attested".
func (s AttestationStore) IsAttested(ctx context.Context, command string) bool {
	path := command
	if !filepath.IsAbs(path) {
		resolved, err := exec.LookPath(command)
		if err != nil {
			return false
		}
		path = resolved
	}

	res, err := Verify(ctx, path, VerifyOptions{TrustedKeys: s.TrustedKeys, RequireTrusted: true})
	if err != nil {
		return false
	}
	return res.Verified && res.Status == StatusSignedVerified
}
