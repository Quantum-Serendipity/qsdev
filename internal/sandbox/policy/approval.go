package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// ErrPolicyNotApproved reports a policy file whose current content has not
// been approved with `sandbox approve`. The policy comes from the repository
// the sandbox exists to contain, so it is privileged configuration: it is
// never evaluated until the user has approved exactly this content.
var ErrPolicyNotApproved = errors.New("sandbox policy is not approved")

// approvalDigestFormat versions the snapshot digest, so a change to what it
// covers invalidates every recorded approval.
const approvalDigestFormat = "qsdev-sandbox-policy-approval-v1"

// maxSnapshotFileSize bounds each file read into a policy snapshot. A policy
// is a small attribute set; anything larger is not one.
const maxSnapshotFileSize = 1 << 20

// Snapshot is the exact content a policy evaluation may read: the policy file
// and the regular *.nix files beside it, read once. The policy is evaluated
// from a private copy of the snapshot with Nix's restricted evaluation
// confined to that copy, so the Digest covers everything the evaluation can
// see and an approval cannot be raced by editing the files afterwards.
type Snapshot struct {
	// Path is the policy file's absolute, symlink-resolved path. Approvals
	// are recorded against it.
	Path string
	// Entry is the policy file's base name within Files.
	Entry string
	// Files maps each base name to its content; it includes Entry.
	Files map[string][]byte
	// Digest is the hex SHA-256 over Entry and every file's name and content.
	Digest string
}

// ReadSnapshot reads the policy at policyPath and its sibling *.nix files.
// Siblings that are not regular files (symlinks, directories, sockets) are
// left out, so an import through one fails instead of reading a file outside
// the policy directory. The policy file itself must be a regular file.
func ReadSnapshot(policyPath string) (*Snapshot, error) {
	abs, err := filepath.Abs(policyPath)
	if err != nil {
		return nil, fmt.Errorf("resolving policy path %s: %w", policyPath, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("resolving policy path %s: %w", policyPath, err)
	}
	snap := &Snapshot{Path: resolved, Entry: filepath.Base(resolved), Files: map[string][]byte{}}

	entry, err := readSnapshotFile(resolved)
	if err != nil {
		return nil, err
	}
	snap.Files[snap.Entry] = entry

	dir := filepath.Dir(resolved)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing policy directory %s: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == snap.Entry || filepath.Ext(name) != ".nix" || !e.Type().IsRegular() {
			continue
		}
		data, err := readSnapshotFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		snap.Files[name] = data
	}
	snap.Digest = snap.digest()
	return snap, nil
}

// readSnapshotFile reads one regular policy file, refusing anything else and
// anything larger than maxSnapshotFileSize.
func readSnapshotFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("policy file %s is not a regular file", path)
	}
	if info.Size() > maxSnapshotFileSize {
		return nil, fmt.Errorf("policy file %s is larger than %d bytes", path, maxSnapshotFileSize)
	}
	data, err := os.ReadFile(path) //nolint:gosec // path is the policy file or a sibling in its directory
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}
	if len(data) > maxSnapshotFileSize {
		return nil, fmt.Errorf("policy file %s is larger than %d bytes", path, maxSnapshotFileSize)
	}
	return data, nil
}

// digest hashes the entry name and every file's name and content, each field
// length-prefixed so no two snapshots share an encoding.
func (s *Snapshot) digest() string {
	h := sha256.New()
	writeField := func(b []byte) {
		_, _ = fmt.Fprintf(h, "%d:", len(b))
		_, _ = h.Write(b)
	}
	writeField([]byte(approvalDigestFormat))
	writeField([]byte(s.Entry))
	for _, name := range s.FileNames() {
		writeField([]byte(name))
		writeField(s.Files[name])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// FileNames returns the snapshot's file names, sorted.
func (s *Snapshot) FileNames() []string {
	names := make([]string, 0, len(s.Files))
	for name := range s.Files {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// writeTo materialises the snapshot in dir, which must be private to this
// process.
func (s *Snapshot) writeTo(dir string) (string, error) {
	for name, data := range s.Files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			return "", fmt.Errorf("writing policy snapshot: %w", err)
		}
	}
	return filepath.Join(dir, s.Entry), nil
}

// ApprovalStore records which policy content the user has approved. It lives
// in the user's home directory, outside any repository, so a repository
// cannot approve its own policy.
type ApprovalStore struct {
	path string
}

// approvalFile is the on-disk shape of the approval store.
type approvalFile struct {
	Approvals map[string]approvalRecord `json:"approvals"`
}

// approvalRecord is one approved policy: the digest of its snapshot and when
// it was approved.
type approvalRecord struct {
	Digest     string    `json:"digest"`
	ApprovedAt time.Time `json:"approved_at"`
}

// NewApprovalStore returns a store backed by the file at path.
func NewApprovalStore(path string) *ApprovalStore {
	return &ApprovalStore{path: path}
}

// DefaultApprovalStore returns the user-global store,
// ~/.<app>/sandbox-policy-approvals.json. It fails when the home directory
// cannot be determined or is not absolute: a relative fallback would resolve
// against the working directory, which a cloned repository controls.
func DefaultApprovalStore() (*ApprovalStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locating sandbox policy approvals: %w", err)
	}
	if !filepath.IsAbs(home) {
		return nil, fmt.Errorf("locating sandbox policy approvals: home directory %q is not absolute", home)
	}
	return NewApprovalStore(filepath.Join(home, "."+branding.Get().AppName, "sandbox-policy-approvals.json")), nil
}

// Path returns the store's file path.
func (s *ApprovalStore) Path() string { return s.path }

// Check returns nil when snap's exact content is approved for its path, and
// an error wrapping ErrPolicyNotApproved otherwise.
func (s *ApprovalStore) Check(snap *Snapshot) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	rec, ok := f.Approvals[snap.Path]
	switch {
	case !ok:
		return fmt.Errorf("%w: %s has never been approved; review it and run `%s sandbox approve --policy %s`",
			ErrPolicyNotApproved, snap.Path, branding.Get().AppName, snap.Path)
	case rec.Digest != snap.Digest:
		return fmt.Errorf("%w: %s (or a *.nix file beside it) changed since it was approved; review it and run `%s sandbox approve --policy %s`",
			ErrPolicyNotApproved, snap.Path, branding.Get().AppName, snap.Path)
	}
	return nil
}

// Approve records snap's content as approved for its path, replacing any
// earlier approval of that path.
func (s *ApprovalStore) Approve(snap *Snapshot, now time.Time) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	f.Approvals[snap.Path] = approvalRecord{Digest: snap.Digest, ApprovedAt: now.UTC()}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding sandbox policy approvals: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("creating sandbox policy approvals directory: %w", err)
	}
	if err := fileutil.WriteFileAtomic(s.path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("writing sandbox policy approvals: %w", err)
	}
	return nil
}

// load reads the store; a missing file is an empty store. A corrupt file is
// an error rather than an empty store, so it is noticed instead of silently
// dropping every approval on the next write.
func (s *ApprovalStore) load() (*approvalFile, error) {
	f := &approvalFile{Approvals: map[string]approvalRecord{}}
	data, err := os.ReadFile(s.path) //nolint:gosec // the store path is in the user's home directory
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading sandbox policy approvals: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return f, nil
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, fmt.Errorf("parsing sandbox policy approvals %s: %w", s.path, err)
	}
	if f.Approvals == nil {
		f.Approvals = map[string]approvalRecord{}
	}
	return f, nil
}
