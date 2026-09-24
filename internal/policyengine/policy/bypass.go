package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

const (
	// DefaultSessionGrantTTL is how long a session-tier bypass lasts when the
	// grant does not name a lifetime.
	DefaultSessionGrantTTL = 8 * time.Hour
	// DefaultCommandTokenTTL is how long an unused command-tier token stays
	// redeemable when the grant does not name a lifetime.
	DefaultCommandTokenTTL = time.Hour
	// MaxBypassTTL caps the lifetime of any bypass grant.
	MaxBypassTTL = 24 * time.Hour

	// sessionStateVersion is the schema version of the session state file.
	sessionStateVersion = 2
)

// sessionStateLockTimeout bounds the wait for the session state lock. The
// PreToolUse hook redeems tokens under this lock, and Claude Code lets a call
// run when its hook times out, so the wait must end, and fail closed, well
// inside the hook's timeout instead of letting a stuck holder turn a one-shot
// token into a reusable one.
const sessionStateLockTimeout = 2 * time.Second

// ErrBypassTokenUnavailable is returned by ConsumeCommandTokens when a
// command-tier token an evaluation relied on is no longer redeemable: another
// tool call consumed it first, it expired, or the grants were cleared.
var ErrBypassTokenUnavailable = errors.New("command bypass token is no longer available")

// BypassScope identifies the project and Claude Code session a bypass grant
// applies to. A scope missing either part matches no grant.
type BypassScope struct {
	ProjectRoot string
	SessionID   string
}

// Valid reports whether the scope names both a project and a session.
func (s BypassScope) Valid() bool {
	return s.ProjectRoot != "" && s.SessionID != ""
}

// BypassGrant is one human-granted bypass of a session- or command-tier rule,
// bound to a single project and Claude Code session. A session-tier grant
// lifts its rule until ExpiresAt; a command-tier grant is a one-shot token
// that the next tool call the rule matches consumes.
type BypassGrant struct {
	RuleID      string    `json:"rule_id"`
	Tier        string    `json:"tier"`
	ProjectRoot string    `json:"project_root"`
	SessionID   string    `json:"session_id"`
	GrantedAt   time.Time `json:"granted_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// NewBypassGrant builds the grant for a rule of the given tier. A zero ttl
// selects the tier's default lifetime. It rejects a tier that cannot be
// bypassed, an incomplete scope and a lifetime outside (0, MaxBypassTTL].
func NewBypassGrant(ruleID string, tier BypassTier, scope BypassScope, ttl time.Duration, now time.Time) (BypassGrant, error) {
	if tier != Session && tier != Command {
		return BypassGrant{}, fmt.Errorf("rule %s: bypass_tier %s cannot be bypassed", ruleID, tier)
	}
	if !scope.Valid() {
		return BypassGrant{}, fmt.Errorf("rule %s: a bypass needs both a project root and a session ID", ruleID)
	}
	if ttl == 0 {
		ttl = DefaultSessionGrantTTL
		if tier == Command {
			ttl = DefaultCommandTokenTTL
		}
	}
	if ttl < 0 || ttl > MaxBypassTTL {
		return BypassGrant{}, fmt.Errorf("bypass lifetime %s must be positive and at most %s", ttl, MaxBypassTTL)
	}
	return BypassGrant{
		RuleID:      ruleID,
		Tier:        tier.String(),
		ProjectRoot: scope.ProjectRoot,
		SessionID:   scope.SessionID,
		GrantedAt:   now.UTC(),
		ExpiresAt:   now.Add(ttl).UTC(),
	}, nil
}

// Active reports whether the grant is unexpired at now and bound to scope.
func (g BypassGrant) Active(scope BypassScope, now time.Time) bool {
	return scope.Valid() && g.ProjectRoot == scope.ProjectRoot && g.SessionID == scope.SessionID && now.Before(g.ExpiresAt)
}

// sameSlot reports whether two grants lift the same rule, in the same way, for
// the same scope, so a newer one replaces the older instead of accumulating.
func (g BypassGrant) sameSlot(o BypassGrant) bool {
	return g.RuleID == o.RuleID && g.Tier == o.Tier && g.ProjectRoot == o.ProjectRoot && g.SessionID == o.SessionID
}

// ActiveOverrides are the bypasses in force for one evaluation: the rule IDs
// of active session-tier grants and of unconsumed command-tier tokens.
type ActiveOverrides struct {
	Session []string
	Command []string
}

// activeOverrides resolves the grants in force for scope at now.
func activeOverrides(grants []BypassGrant, scope BypassScope, now time.Time) ActiveOverrides {
	var o ActiveOverrides
	for _, g := range grants {
		if !g.Active(scope, now) {
			continue
		}
		switch g.Tier {
		case Session.String():
			o.Session = append(o.Session, g.RuleID)
		case Command.String():
			o.Command = append(o.Command, g.RuleID)
		}
	}
	return o
}

type sessionState struct {
	Version int           `json:"version"`
	Grants  []BypassGrant `json:"grants"`
	// LegacyOverrides is the unscoped, non-expiring override list written by
	// releases before version 2. It is read only so it can be reported; it
	// never lifts a rule and is dropped on the next write.
	LegacyOverrides []string `json:"sessionBypassOverrides,omitempty"`
}

// FileSessionStateStore keeps bypass grants in a JSON file shared by every
// qsdev process of the user. Reads see a consistent file because writes
// replace it atomically; read-modify-write sequences additionally hold an
// exclusive lock on a sibling lock file.
type FileSessionStateStore struct {
	path        string
	lockTimeout time.Duration
}

func NewFileSessionStateStore(path string) *FileSessionStateStore {
	return &FileSessionStateStore{path: path, lockTimeout: sessionStateLockTimeout}
}

// ActiveOverrides returns the grants in force for scope at now. An unreadable
// or malformed state file yields no overrides, the strictest state.
func (s *FileSessionStateStore) ActiveOverrides(scope BypassScope, now time.Time) ActiveOverrides {
	state, err := s.load()
	if err != nil {
		return ActiveOverrides{}
	}
	return activeOverrides(state.Grants, scope, now)
}

// Grants returns every grant unexpired at now, and the legacy unscoped
// overrides a pre-version-2 state file still lists (which no longer apply).
func (s *FileSessionStateStore) Grants(now time.Time) (grants []BypassGrant, legacy []string, err error) {
	state, err := s.load()
	if err != nil {
		return nil, nil, err
	}
	return unexpired(state.Grants, now), state.LegacyOverrides, nil
}

// AddGrants records grants, replacing any existing grant for the same rule,
// tier and scope, and prunes expired grants.
func (s *FileSessionStateStore) AddGrants(grants []BypassGrant, now time.Time) error {
	return s.update(func(state *sessionState) error {
		kept := unexpired(state.Grants, now)
		for _, g := range grants {
			kept = slices.DeleteFunc(kept, g.sameSlot)
			kept = append(kept, g)
		}
		state.Grants = kept
		return nil
	})
}

// ConsumeCommandTokens redeems one command-tier token per rule ID for scope.
// It is all-or-nothing: when any token is missing or expired nothing is
// consumed and the error wraps ErrBypassTokenUnavailable, so a concurrent
// call cannot reuse a token another call already spent.
func (s *FileSessionStateStore) ConsumeCommandTokens(scope BypassScope, ruleIDs []string, now time.Time) error {
	if len(ruleIDs) == 0 {
		return nil
	}
	return s.update(func(state *sessionState) error {
		kept := unexpired(state.Grants, now)
		for _, id := range ruleIDs {
			i := slices.IndexFunc(kept, func(g BypassGrant) bool {
				return g.RuleID == id && g.Tier == Command.String() && g.Active(scope, now)
			})
			if i < 0 {
				return fmt.Errorf("rule %s: %w", id, ErrBypassTokenUnavailable)
			}
			kept = slices.Delete(kept, i, i+1)
		}
		state.Grants = kept
		return nil
	})
}

// Clear removes every grant.
func (s *FileSessionStateStore) Clear() error {
	lock, err := s.lock()
	if err != nil {
		return fmt.Errorf("clearing session state: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	if err := os.Remove(s.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clearing session state: %w", err)
	}
	return nil
}

// lock takes the state lock, waiting at most the store's lock timeout.
func (s *FileSessionStateStore) lock() (*fileutil.FileLock, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.lockTimeout)
	defer cancel()
	return fileutil.LockExclusive(ctx, s.path+".lock")
}

// load reads the state file. A missing file is an empty state.
func (s *FileSessionStateStore) load() (sessionState, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return sessionState{Version: sessionStateVersion}, nil
	}
	if err != nil {
		return sessionState{}, fmt.Errorf("reading session state: %w", err)
	}
	var state sessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return sessionState{}, fmt.Errorf("parsing session state %s: %w", s.path, err)
	}
	return state, nil
}

// update applies mutate to the state under the state lock and writes the
// result atomically. Nothing is written when mutate fails.
func (s *FileSessionStateStore) update(mutate func(*sessionState) error) error {
	lock, err := s.lock()
	if err != nil {
		return fmt.Errorf("updating session state: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	state, err := s.load()
	if err != nil {
		return err
	}
	if err := mutate(&state); err != nil {
		return err
	}
	state.Version = sessionStateVersion
	state.LegacyOverrides = nil

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session state: %w", err)
	}
	if err := fileutil.WriteFileAtomic(s.path, data, fileutil.ModePrivate); err != nil {
		return fmt.Errorf("writing session state: %w", err)
	}
	return nil
}

func unexpired(grants []BypassGrant, now time.Time) []BypassGrant {
	return slices.DeleteFunc(slices.Clone(grants), func(g BypassGrant) bool {
		return !now.Before(g.ExpiresAt)
	})
}

// StaticSessionStateReader serves fixed overrides regardless of scope, for
// callers (and tests) that resolve bypasses themselves. Consuming a command
// token removes it.
type StaticSessionStateReader struct {
	mu      sync.Mutex
	Session []string
	Command []string
}

func (r *StaticSessionStateReader) ActiveOverrides(BypassScope, time.Time) ActiveOverrides {
	r.mu.Lock()
	defer r.mu.Unlock()
	return ActiveOverrides{Session: slices.Clone(r.Session), Command: slices.Clone(r.Command)}
}

func (r *StaticSessionStateReader) ConsumeCommandTokens(_ BypassScope, ruleIDs []string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	remaining := slices.Clone(r.Command)
	for _, id := range ruleIDs {
		i := slices.Index(remaining, id)
		if i < 0 {
			return fmt.Errorf("rule %s: %w", id, ErrBypassTokenUnavailable)
		}
		remaining = slices.Delete(remaining, i, i+1)
	}
	r.Command = remaining
	return nil
}
