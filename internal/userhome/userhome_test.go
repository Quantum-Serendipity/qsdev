package userhome

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestAccountIgnoresHomeEnvironment pins that the account's home directory
// does not follow HOME or USERPROFILE: a line such as `HOME=/tmp/e qsdev
// claude update` must not move what Account returns.
func TestAccountIgnoresHomeEnvironment(t *testing.T) {
	want, err := Account()
	if err != nil {
		t.Skipf("the current account has no user database entry here: %v", err)
	}
	for _, name := range []string{"HOME", "USERPROFILE"} {
		t.Run(name, func(t *testing.T) {
			fake := filepath.Join(t.TempDir(), "elsewhere")
			t.Setenv(name, fake)
			got, err := Account()
			if err != nil || got != want {
				t.Errorf("Account() with %s=%s = %q, %v; want %q", name, fake, got, err, want)
			}
		})
	}
}

// TestAccountFailsWithoutEntry pins that Account fails, rather than reading
// the environment, when the account has no usable user database entry.
func TestAccountFailsWithoutEntry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	tests := []struct {
		name   string
		lookup func() (*user.User, error)
	}{
		{"no entry", func() (*user.User, error) { return nil, user.UnknownUserIdError(4242) }},
		{"entry without home", func() (*user.User, error) { return &user.User{Uid: "4242"}, nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orig := lookup
			lookup = tt.lookup
			t.Cleanup(func() { lookup = orig })
			if home, err := Account(); err == nil {
				t.Errorf("Account() = %q without a usable entry, want an error", home)
			}
		})
	}
}

// TestFindPasswdEntry pins how a passwd line is read: the home directory is
// the sixth field, and the entry must be the one asked for.
func TestFindPasswdEntry(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		out      string
		key      string
		field    int
		wantHome string
	}{
		{"by uid", "ldapuser:*:54321:100:LDAP User:/home/ldapuser:/bin/bash\n", "54321", uidField, "/home/ldapuser"},
		{"by name", "ldapuser:*:54321:100::/home/ldapuser:/bin/sh", "ldapuser", nameField, "/home/ldapuser"},
		{"later line", "a:x:1:1::/home/a:/bin/sh\nb:x:2:2::/home/b:/bin/sh\n", "2", uidField, "/home/b"},
		{"first match", "a:x:1:1::/home/a:/bin/sh\nc:x:1:1::/home/c:/bin/sh\n", "1", uidField, "/home/a"},
		{"skips comments and compat lines", "# x\n+b:x:2:2::/nis:/bin/sh\nb:x:2:2::/home/b:/bin/sh\n", "2", uidField, "/home/b"},
		{"other account", "root:x:0:0::/root:/bin/sh", "54321", uidField, ""},
		{"name is not a uid", "root:x:0:0::/root:/bin/sh", "0", nameField, ""},
		{"empty", "", "54321", uidField, ""},
		{"short line", "ldapuser:*:54321", "54321", uidField, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := findPasswdEntry(strings.NewReader(tt.out), tt.key, tt.field)
			got := ""
			if u != nil {
				got = u.HomeDir
			}
			if got != tt.wantHome {
				t.Errorf("findPasswdEntry() home = %q, want %q", got, tt.wantHome)
			}
		})
	}
}

// fakePasswd points passwdFile at a file holding content.
func fakePasswd(t *testing.T, content string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	orig := passwdFile
	passwdFile = p
	resetPasswdCache(t)
	t.Cleanup(func() { passwdFile = orig })
}

// resetPasswdCache empties the lookup cache now and when t ends, so a test
// that replaces the passwd database sees its own answers.
func resetPasswdCache(t *testing.T) {
	t.Helper()
	passwdCache.Clear()
	t.Cleanup(passwdCache.Clear)
}

// fakeGetent installs a getent script as the only one getentPaths finds (or
// none, when nssHome is ""). It prints a passwd entry whose home directory is
// $HOME when getent is given an environment, and nssHome otherwise, so a test
// sees whether the caller's environment reached it.
func fakeGetent(t *testing.T, nssHome string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("getent is not used on Windows")
	}
	missing := filepath.Join(t.TempDir(), "missing", "getent")
	orig := getentPaths
	resetPasswdCache(t)
	t.Cleanup(func() { getentPaths = orig })
	if nssHome == "" {
		getentPaths = []string{missing}
		return
	}
	script := filepath.Join(t.TempDir(), "getent")
	body := "#!/bin/sh\n" +
		"h=\"${HOME:-" + nssHome + "}\"\n" +
		"case \"$2\" in 54321|nssuser) echo \"nssuser:*:54321:100::$h:/bin/sh\" ;; *) exit 2 ;; esac\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	getentPaths = []string{missing, script}
}

// TestLookupPasswd pins the passwd lookup used where os/user would consult
// the environment: /etc/passwd first, then getent run from a fixed path with
// an empty environment, so an NSS account (absent from /etc/passwd in a
// build without cgo) resolves to its real home and HOME set for the process
// does not reach the answer. Not parallel: it replaces package variables.
func TestLookupPasswd(t *testing.T) {
	const nssHome = "/home/nssuser"
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USER", "nssuser")

	t.Run("local entry", func(t *testing.T) {
		fakePasswd(t, "local:x:54321:100::/home/local:/bin/sh\n")
		fakeGetent(t, nssHome)
		if u, err := lookupPasswd("54321", uidField); err != nil || u.HomeDir != "/home/local" {
			t.Errorf("lookupPasswd(54321) = %+v, %v; want home /home/local", u, err)
		}
	})
	t.Run("nss entry", func(t *testing.T) {
		fakePasswd(t, "root:x:0:0::/root:/bin/sh\n")
		fakeGetent(t, nssHome)
		if u, err := lookupPasswd("54321", uidField); err != nil || u.HomeDir != nssHome {
			t.Errorf("lookupPasswd(54321) = %+v, %v; want home %q", u, err, nssHome)
		}
		if u, err := lookupPasswd("nssuser", nameField); err != nil || u.HomeDir != nssHome {
			t.Errorf("lookupPasswd(nssuser) = %+v, %v; want home %q", u, err, nssHome)
		}
		if u, err := lookupPasswd("4242", uidField); err == nil {
			t.Errorf("lookupPasswd(4242) = %+v, want an error", u)
		}
		if u, err := lookupPasswd("-x", nameField); err == nil {
			t.Errorf("lookupPasswd(-x) = %+v, want an option-like key refused", u)
		}
	})
	t.Run("no entry and no getent", func(t *testing.T) {
		fakePasswd(t, "root:x:0:0::/root:/bin/sh\n")
		fakeGetent(t, "")
		if u, err := lookupPasswd("54321", uidField); err == nil {
			t.Errorf("lookupPasswd(54321) = %+v, want an error, not an answer from HOME or USER", u)
		}
		// Regression (U18-WS1 round 2): os/user in a build without cgo
		// answered for the running account from HOME and USER.
		if readsPasswd {
			if home, err := Account(); err == nil {
				t.Errorf("Account() = %q without a passwd entry, want an error", home)
			}
		}
	})
}

// TestNamed pins that Named reads an account's home directory from the user
// database and fails for an account it has no entry for.
func TestNamed(t *testing.T) {
	orig := lookupName
	t.Cleanup(func() { lookupName = orig })
	lookupName = func(name string) (*user.User, error) {
		if name == "alice" {
			return &user.User{Username: name, HomeDir: "/home/alice"}, nil
		}
		return nil, user.UnknownUserError(name)
	}
	if got, err := Named("alice"); err != nil || got != "/home/alice" {
		t.Errorf("Named(alice) = %q, %v; want /home/alice", got, err)
	}
	if got, err := Named("nobody-" + strconv.Itoa(os.Getpid())); err == nil {
		t.Errorf("Named(unknown) = %q, want an error", got)
	}
}

// TestLookupPasswdMemoized pins that a passwd lookup runs getent at most once
// per key for the life of the process, failures included: a guard hook
// expands every ~name word of a command several times, and an account the
// database does not know would otherwise cost a getent run (up to
// getentTimeout when NSS hangs) on each expansion (U18-WS1). Not parallel: it
// replaces package variables.
func TestLookupPasswdMemoized(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("getent is not used on Windows")
	}
	fakePasswd(t, "root:x:0:0::/root:/bin/sh\n")
	dir := t.TempDir()
	runs := filepath.Join(dir, "runs")
	script := filepath.Join(dir, "getent")
	body := "#!/bin/sh\n" +
		"echo run >> '" + runs + "'\n" +
		"case \"$2\" in nssuser) echo 'nssuser:*:54321:100::/home/nssuser:/bin/sh' ;; *) exit 2 ;; esac\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil { // #nosec G306 -- test executable
		t.Fatal(err)
	}
	orig := getentPaths
	getentPaths = []string{script}
	resetPasswdCache(t)
	t.Cleanup(func() { getentPaths = orig })

	for range 3 {
		if _, err := lookupPasswd("nouser", nameField); err == nil {
			t.Fatal("lookupPasswd(nouser) succeeded, want an error")
		}
		u, err := lookupPasswd("nssuser", nameField)
		if err != nil || u.HomeDir != "/home/nssuser" {
			t.Fatalf("lookupPasswd(nssuser) = %+v, %v; want home /home/nssuser", u, err)
		}
		u.HomeDir = "/tmp/changed" // the caller's copy, not the cached entry
	}
	data, err := os.ReadFile(runs)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "run"); n != 2 {
		t.Errorf("getent ran %d times for 3 lookups each of two keys, want 2 (one per key)", n)
	}
}
