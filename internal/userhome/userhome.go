// Package userhome resolves home directories from the operating system's
// user database, independent of the environment.
//
// os.UserHomeDir reads HOME (USERPROFILE on Windows), which any parent
// process, or a shell line an AI agent runs, can set for one invocation. A
// file that steers a guardrail (the org overlay) must not move with it, so its
// location is anchored to the account instead, the way OpenSSH reads
// ~/.ssh from the password database rather than from $HOME.
//
// os/user cannot be used for that on Linux and the other non-macOS Unixes in
// a build without cgo (the release builds): its Current falls back to $HOME
// and $USER when /etc/passwd has no entry for the account, and LookupId and
// Lookup return Current's answer for the running account. An account that
// comes from NSS (SSSD, LDAP or Active Directory, NIS, systemd-homed) has no
// entry there, so the environment would decide. On those systems the passwd
// database is read directly: /etc/passwd, then getent, run from a fixed system
// path with an empty environment, so neither PATH nor a variable such as
// LD_PRELOAD can choose what answers. macOS (Directory Services through libc)
// and Windows (the process token and the account database) do not consult
// the environment, so os/user is used there.
package userhome

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// ErrNoAccount reports that the user database has no entry for an account:
// /etc/passwd lists none and getent, when installed, finds none either (or
// os/user reports the account unknown). It is not reported for a lookup that
// fails some other way (an unreadable database, a getent run that hangs or
// cannot start), so a caller may tell an account the system does not know,
// such as an arbitrary container uid, from a lookup it could not complete.
var ErrNoAccount = errors.New("the user database has no entry for the account")

// errGetentNotFound reports getent's own "key not found" result (exit
// status 2).
var errGetentNotFound = errors.New("getent found no entry")

// errNoGetent reports that getent is not installed at any of getentPaths.
var errNoGetent = errors.New("getent is not installed at a system path")

// getentNotFoundStatus is the exit status getent documents for a key the
// database does not have.
const getentNotFoundStatus = 2

// passwdFile is the local passwd database (a variable for tests).
var passwdFile = "/etc/passwd"

// getentPaths are the locations getent is run from, in order: the usual FHS
// locations and NixOS's system profile. getent is never looked up through
// PATH, which the caller's environment controls. A variable so tests can
// supply a fake.
var getentPaths = []string{"/usr/bin/getent", "/bin/getent", "/run/current-system/sw/bin/getent"}

// getentTimeout bounds one getent run; a hung directory service must not hang
// the CLI or a guard hook.
const getentTimeout = 5 * time.Second

// readsPasswd reports whether accounts are looked up in the passwd database
// directly rather than through os/user (see the package comment).
var readsPasswd = runtime.GOOS != "windows" && runtime.GOOS != "darwin" && runtime.GOOS != "plan9"

// lookup returns the user database entry of the account running the process.
// It is a variable so tests can simulate an account without an entry.
var lookup = func() (*user.User, error) {
	uid := os.Getuid()
	switch {
	case uid < 0:
		// Windows has no numeric uid: user.Current reads the profile
		// directory from the process token, not from USERPROFILE.
		return user.Current()
	case readsPasswd:
		return lookupPasswd(strconv.Itoa(uid), uidField)
	default:
		u, err := user.LookupId(strconv.Itoa(uid))
		return u, unknownAccount(err)
	}
}

// unknownAccount marks an os/user error that reports the account unknown as
// ErrNoAccount; other errors are returned as they are.
func unknownAccount(err error) error {
	var unknownID user.UnknownUserIdError
	var unknownName user.UnknownUserError
	if errors.As(err, &unknownID) || errors.As(err, &unknownName) {
		return fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	return err
}

// lookupName returns the user database entry of the account named name. It
// is a variable so tests can simulate the database.
var lookupName = func(name string) (*user.User, error) {
	if readsPasswd {
		return lookupPasswd(name, nameField)
	}
	u, err := user.Lookup(name)
	return u, unknownAccount(err)
}

// The passwd entry fields a lookup key is matched against.
const (
	nameField = 0
	uidField  = 2
)

// passwdKey identifies one passwd lookup: the key and the field it is
// matched against.
type passwdKey struct {
	key   string
	field int
}

// passwdResult is the outcome of one passwd lookup, kept by passwdCache.
type passwdResult struct {
	user *user.User
	err  error
}

// passwdCache memoizes lookupPasswd for the life of the process, failures
// included: a guard hook expands every ~name word of a command, several
// times over, and an account missing from /etc/passwd costs a getent run
// (up to getentTimeout when NSS hangs) on each lookup. The user database
// does not change in a way that matters within one short-lived run.
var passwdCache sync.Map // passwdKey -> passwdResult

// lookupPasswd returns the passwd entry whose field number field is key: from
// passwdFile, or, when that has none (a directory-service account), from
// getent (see getentPasswd). Results, failures included, are memoized (see
// passwdCache); the returned entry is a copy the caller may change.
func lookupPasswd(key string, field int) (*user.User, error) {
	k := passwdKey{key, field}
	if v, ok := passwdCache.Load(k); ok {
		return copyUser(v.(passwdResult))
	}
	u, err := lookupPasswdUncached(key, field)
	passwdCache.Store(k, passwdResult{u, err})
	return copyUser(passwdResult{u, err})
}

// copyUser returns r's entry as a fresh copy, or r's error.
func copyUser(r passwdResult) (*user.User, error) {
	if r.err != nil {
		return nil, r.err
	}
	u := *r.user
	return &u, nil
}

// lookupPasswdUncached is lookupPasswd without the memoization.
func lookupPasswdUncached(key string, field int) (*user.User, error) {
	if key == "" {
		return nil, errors.New("looking up an empty account key")
	}
	u, err := scanPasswdFile(key, field)
	if err == nil {
		return u, nil
	}
	nss, nssErr := getentPasswd(key, field)
	if nssErr == nil {
		return nss, nil
	}
	err = errors.Join(err, nssErr)
	// Only a database that answered "no such entry" means the account is
	// unknown: getent's not-found result, or, without getent, a passwd file
	// that was read and has no entry.
	if errors.Is(nssErr, errGetentNotFound) || (errors.Is(nssErr, errNoGetent) && errors.Is(err, errNotInPasswdFile)) {
		return nil, fmt.Errorf("no passwd entry for %q: %w: %w", key, ErrNoAccount, err)
	}
	return nil, fmt.Errorf("no passwd entry for %q: %w", key, err)
}

// errNotInPasswdFile reports that passwdFile was read and has no entry for
// the key.
var errNotInPasswdFile = errors.New("no entry in the passwd file")

// scanPasswdFile returns the entry of passwdFile whose field number field is
// key.
func scanPasswdFile(key string, field int) (*user.User, error) {
	f, err := os.Open(passwdFile)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", passwdFile, err)
	}
	defer f.Close()
	if u := findPasswdEntry(f, key, field); u != nil {
		return u, nil
	}
	return nil, fmt.Errorf("%s has no entry for %q: %w", passwdFile, key, errNotInPasswdFile)
}

// findPasswdEntry returns the first passwd line of r
// (name:password:uid:gid:gecos:home:shell) whose field number field is key,
// or nil. NIS compat lines (+, -) are skipped, as os/user does.
func findPasswdEntry(r io.Reader, key string, field int) *user.User {
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' || line[0] == '+' || line[0] == '-' {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) != 7 || fields[field] != key {
			continue
		}
		return &user.User{Username: fields[0], Uid: fields[2], Gid: fields[3], Name: fields[4], HomeDir: fields[5]}
	}
	return nil
}

// getentPasswd runs `getent passwd <key>` from the first of getentPaths that
// exists, with an empty environment, and returns the entry it prints whose
// field number field is key.
func getentPasswd(key string, field int) (*user.User, error) {
	if strings.HasPrefix(key, "-") {
		return nil, fmt.Errorf("refusing the option-like account key %q", key)
	}
	for _, p := range getentPaths {
		out, err := runGetent(p, key)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if u := findPasswdEntry(bytes.NewReader(out), key, field); u != nil {
			return u, nil
		}
		return nil, fmt.Errorf("getent printed no passwd entry for %q: %w", key, errGetentNotFound)
	}
	return nil, errNoGetent
}

// runGetent runs the getent binary at path for the passwd entry of key, with
// an empty environment, and returns what it prints. A run that outlasts
// getentTimeout is killed. The kill timer stands in for a context deadline:
// Account and Named have no caller context to derive one from
// (catalog.OrgConfigPath, canon.ExpandTilde), and only the cmd and instance
// layers may start a root context.
func runGetent(path, key string) ([]byte, error) {
	cmd := procexec.Command(path, "passwd", key)
	cmd.Env = []string{}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %s passwd %s: %w", path, key, err)
	}
	timer := time.AfterFunc(getentTimeout, func() { _ = cmd.Process.Kill() })
	defer timer.Stop()
	if err := cmd.Wait(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == getentNotFoundStatus {
			err = fmt.Errorf("%w (%w)", errGetentNotFound, err)
		}
		return nil, fmt.Errorf("running %s passwd %s: %w", path, key, err)
	}
	return stdout.Bytes(), nil
}

// Account returns the home directory the user database records for the
// account running the process. It fails when the account has no entry or the
// entry names no home directory.
func Account() (string, error) {
	u, err := lookup()
	if err != nil {
		return "", fmt.Errorf("looking up the current account: %w", err)
	}
	return homeOf(u, "the current account")
}

// Named returns the home directory the user database records for the account
// called name, the directory the shell's ~name expands to. It fails when the
// account has no entry or the entry names no home directory.
func Named(name string) (string, error) {
	u, err := lookupName(name)
	if err != nil {
		return "", fmt.Errorf("looking up account %q: %w", name, err)
	}
	return homeOf(u, "account "+strconv.Quote(name))
}

// homeOf returns u's home directory, or an error naming who when it has none.
func homeOf(u *user.User, who string) (string, error) {
	if u.HomeDir == "" {
		return "", fmt.Errorf("the user database records no home directory for %s", who)
	}
	return u.HomeDir, nil
}
