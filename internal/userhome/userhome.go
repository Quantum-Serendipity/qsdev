// Package userhome resolves the home directory of the account running the
// process from the operating system's user database, independent of the
// environment.
//
// os.UserHomeDir reads HOME (USERPROFILE on Windows), which any parent
// process, or a shell line an AI agent runs, can set for one invocation. A
// file that steers a guardrail (the org overlay) must not move with it, so its
// location is anchored to the account instead, the way OpenSSH reads
// ~/.ssh from the password database rather than from $HOME.
package userhome

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"
)

// lookup returns the user database entry of the account running the process.
// It is a variable so tests can simulate an account without an entry.
var lookup = func() (*user.User, error) {
	if uid := os.Getuid(); uid >= 0 {
		// user.Current falls back to $HOME when the account has no entry
		// (a build without cgo, an arbitrary container uid); LookupId does
		// not, so the result never comes from the environment.
		return user.LookupId(strconv.Itoa(uid))
	}
	// Windows has no numeric uid: user.Current reads the profile directory
	// from the process token, not from USERPROFILE.
	return user.Current()
}

// Account returns the home directory the user database records for the
// account running the process. It fails when the account has no entry or the
// entry names no home directory.
func Account() (string, error) {
	u, err := lookup()
	if err != nil {
		return "", fmt.Errorf("looking up the current account: %w", err)
	}
	if u.HomeDir == "" {
		return "", errors.New("the user database records no home directory for the current account")
	}
	return u.HomeDir, nil
}
