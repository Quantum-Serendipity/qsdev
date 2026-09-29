package contentsign

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// copyPub copies a generated public key file into dir under name.
func copyPub(t *testing.T, kp keyPair, dir, name string) {
	t.Helper()
	data, err := os.ReadFile(kp.pubPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// setHome points the user home directory at home for the test.
func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
}

// userKeysDir is the default trusted-keys dir under home.
func userKeysDir(home string) string {
	return filepath.Join(home, "."+branding.Get().AppName, "keys")
}

func TestParsePublicKey(t *testing.T) {
	t.Parallel()
	kp := newKeyPair(t, t.TempDir(), "k", "")
	pubFile, err := os.ReadFile(kp.pubPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "bare key", input: kp.pub.String()},
		{name: "full .pub file with untrusted comment", input: string(pubFile)},
		{name: "empty", input: "  ", wantErr: true},
		{name: "garbage", input: "RWnot-a-key", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pk, err := ParsePublicKey(tt.input)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidPublicKey) {
					t.Errorf("error = %v, want ErrInvalidPublicKey", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePublicKey: %v", err)
			}
			if pk.ID() != kp.pub.ID() {
				t.Errorf("ID = %s, want %s", pk.ID(), kp.pub.ID())
			}
		})
	}
}

func TestLoadTrustedKeysExplicitDir(t *testing.T) {
	t.Parallel()
	a := newKeyPair(t, t.TempDir(), "a", "")

	t.Run("loads only .pub files", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		copyPub(t, a, dir, "a.pub")
		// Not keys: a non-.pub file holding garbage and a directory named *.pub.
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("garbage"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "sub.pub"), 0o755); err != nil {
			t.Fatal(err)
		}
		keys, err := LoadTrustedKeys(dir)
		if err != nil {
			t.Fatalf("LoadTrustedKeys: %v", err)
		}
		if len(keys) != 1 || keys[0].ID() != a.pub.ID() {
			t.Errorf("keys = %v, want exactly %s", keys, a.pub.ID())
		}
	})

	t.Run("malformed .pub is an error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		copyPub(t, a, dir, "a.pub")
		if err := os.WriteFile(filepath.Join(dir, "bad.pub"), []byte("garbage"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadTrustedKeys(dir); err == nil {
			t.Error("expected an error for a malformed .pub file")
		}
	})

	t.Run("missing explicit dir is an error", func(t *testing.T) {
		t.Parallel()
		if _, err := LoadTrustedKeys(filepath.Join(t.TempDir(), "typo")); err == nil {
			t.Error("expected an error for a missing explicitly named keys dir")
		}
	})

	t.Run("empty explicit dir is an explicit empty set", func(t *testing.T) {
		t.Parallel()
		keys, err := LoadTrustedKeys(t.TempDir())
		if err != nil {
			t.Fatalf("LoadTrustedKeys: %v", err)
		}
		if keys == nil || len(keys) != 0 {
			t.Errorf("keys = %#v, want a non-nil empty set", keys)
		}
	})
}

func TestLoadTrustedKeysDefaultDir(t *testing.T) {
	kp := newKeyPair(t, t.TempDir(), "user", "")

	t.Run("missing default dir yields no keys", func(t *testing.T) {
		setHome(t, t.TempDir())
		keys, err := LoadTrustedKeys("")
		if err != nil || len(keys) != 0 {
			t.Errorf("LoadTrustedKeys(\"\") = %v, %v; want no keys, no error", keys, err)
		}
	})

	t.Run("loads keys from the home keys dir", func(t *testing.T) {
		home := t.TempDir()
		setHome(t, home)
		copyPub(t, kp, userKeysDir(home), "user.pub")
		dir, err := DefaultTrustedKeysDir()
		if err != nil || dir != userKeysDir(home) {
			t.Fatalf("DefaultTrustedKeysDir() = %q, %v; want %q", dir, err, userKeysDir(home))
		}
		keys, err := LoadTrustedKeys("")
		if err != nil || len(keys) != 1 {
			t.Fatalf("LoadTrustedKeys(\"\") = %v, %v; want the user key", keys, err)
		}
	})
}

// TestLoadTrustedKeysNoHomeIgnoresWorkingDir proves that with no home
// directory the default key set does NOT fall back to the CWD-relative
// .qsdev/keys, which a cloned repository controls.
func TestLoadTrustedKeysNoHomeIgnoresWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "plan9" {
		t.Skip("home directory is not derived from $HOME on this platform")
	}
	kp := newKeyPair(t, t.TempDir(), "attacker", "")
	repo := t.TempDir()
	copyPub(t, kp, filepath.Join(repo, "."+branding.Get().AppName, "keys"), "attacker.pub")
	t.Chdir(repo)
	t.Setenv("HOME", "")

	if dir, err := DefaultTrustedKeysDir(); err == nil {
		t.Errorf("DefaultTrustedKeysDir() = %q, want an error without a home directory", dir)
	}
	keys, err := LoadTrustedKeys("")
	if err != nil {
		t.Fatalf("LoadTrustedKeys: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("loaded %d key(s) from the working directory, want none", len(keys))
	}
}

// TestVerifyExplicitEmptyKeySetDoesNotFallBack proves an explicitly supplied
// (possibly empty) trust set is honored instead of silently widening to the
// user's default keys.
func TestVerifyExplicitEmptyKeySetDoesNotFallBack(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	setHome(t, home)
	personal := newKeyPair(t, t.TempDir(), "personal", "")
	copyPub(t, personal, userKeysDir(home), "personal.pub")

	content := writeContent(t, t.TempDir(), "artifact", "payload")
	if _, err := Sign(ctx, content, SignOptions{KeyPath: personal.secPath}); err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// nil means "use the defaults": the personal key verifies.
	res, err := Verify(ctx, content, VerifyOptions{RequireTrusted: true})
	if err != nil || !res.Verified {
		t.Fatalf("default keys: Verify = %+v, %v; want verified", res, err)
	}

	emptyDir, err := LoadTrustedKeys(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for name, keys := range map[string][]PublicKey{
		"literal empty set":  {},
		"empty explicit dir": emptyDir,
	} {
		t.Run(name, func(t *testing.T) {
			res, err := Verify(ctx, content, VerifyOptions{TrustedKeys: keys, RequireTrusted: true})
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
			if res.Verified || res.Status != StatusFailed {
				t.Errorf("result = %+v, want failed (no fallback to default keys)", res)
			}
			if !strings.Contains(res.Reason, "no trusted keys") {
				t.Errorf("reason = %q, want it to say no trusted keys are configured", res.Reason)
			}
		})
	}
}
