package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

var (
	bypassNow    = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	scopeA       = BypassScope{ProjectRoot: "/work/project-a", SessionID: "session-1"}
	scopeB       = BypassScope{ProjectRoot: "/work/project-b", SessionID: "session-1"}
	scopeASess2  = BypassScope{ProjectRoot: "/work/project-a", SessionID: "session-2"}
	scopeNoSess  = BypassScope{ProjectRoot: "/work/project-a"}
	scopeNoProjt = BypassScope{SessionID: "session-1"}
)

func mustGrant(t *testing.T, id string, tier BypassTier, scope BypassScope, ttl time.Duration, now time.Time) BypassGrant {
	t.Helper()
	g, err := NewBypassGrant(id, tier, scope, ttl, now)
	if err != nil {
		t.Fatalf("NewBypassGrant(%s): %v", id, err)
	}
	return g
}

func newTestStore(t *testing.T, grants ...BypassGrant) *FileSessionStateStore {
	t.Helper()
	store := NewFileSessionStateStore(filepath.Join(t.TempDir(), ".qsdev", "session-state.json"))
	if len(grants) > 0 {
		if err := store.AddGrants(grants, bypassNow); err != nil {
			t.Fatalf("AddGrants: %v", err)
		}
	}
	return store
}

func TestNewBypassGrant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tier       BypassTier
		scope      BypassScope
		ttl        time.Duration
		wantErr    bool
		wantExpiry time.Duration
	}{
		{name: "session tier default lifetime", tier: Session, scope: scopeA, wantExpiry: DefaultSessionGrantTTL},
		{name: "command tier default lifetime", tier: Command, scope: scopeA, wantExpiry: DefaultCommandTokenTTL},
		{name: "explicit lifetime", tier: Session, scope: scopeA, ttl: 30 * time.Minute, wantExpiry: 30 * time.Minute},
		{name: "maximum lifetime", tier: Session, scope: scopeA, ttl: MaxBypassTTL, wantExpiry: MaxBypassTTL},
		{name: "lifetime over maximum", tier: Session, scope: scopeA, ttl: MaxBypassTTL + time.Second, wantErr: true},
		{name: "negative lifetime", tier: Command, scope: scopeA, ttl: -time.Minute, wantErr: true},
		{name: "enforce_always tier", tier: EnforceAlways, scope: scopeA, wantErr: true},
		{name: "missing session", tier: Session, scope: scopeNoSess, wantErr: true},
		{name: "missing project", tier: Session, scope: scopeNoProjt, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g, err := NewBypassGrant("R-1", tt.tier, tt.scope, tt.ttl, bypassNow)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got grant %+v", g)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewBypassGrant: %v", err)
			}
			if got := g.ExpiresAt.Sub(g.GrantedAt); got != tt.wantExpiry {
				t.Errorf("lifetime = %s, want %s", got, tt.wantExpiry)
			}
			if g.Tier != tt.tier.String() {
				t.Errorf("tier = %q, want %q", g.Tier, tt.tier)
			}
		})
	}
}

// TestFileSessionStateStore_ActiveOverridesScope is the F193 regression: a
// grant lifts its rule only for the project and Claude Code session it was
// issued for, and only until it expires.
func TestFileSessionStateStore_ActiveOverridesScope(t *testing.T) {
	t.Parallel()

	store := newTestStore(t,
		mustGrant(t, "SESS-1", Session, scopeA, time.Hour, bypassNow),
		mustGrant(t, "CMD-1", Command, scopeA, time.Hour, bypassNow),
	)

	tests := []struct {
		name        string
		scope       BypassScope
		now         time.Time
		wantSession []string
		wantCommand []string
	}{
		{name: "same project and session", scope: scopeA, now: bypassNow, wantSession: []string{"SESS-1"}, wantCommand: []string{"CMD-1"}},
		{name: "other project", scope: scopeB, now: bypassNow},
		{name: "other session", scope: scopeASess2, now: bypassNow},
		{name: "no session ID", scope: scopeNoSess, now: bypassNow},
		{name: "no project root", scope: scopeNoProjt, now: bypassNow},
		{name: "just before expiry", scope: scopeA, now: bypassNow.Add(time.Hour - time.Second), wantSession: []string{"SESS-1"}, wantCommand: []string{"CMD-1"}},
		{name: "at expiry", scope: scopeA, now: bypassNow.Add(time.Hour)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := store.ActiveOverrides(tt.scope, tt.now)
			if !slices.Equal(got.Session, tt.wantSession) || !slices.Equal(got.Command, tt.wantCommand) {
				t.Errorf("ActiveOverrides = %+v, want session %v command %v", got, tt.wantSession, tt.wantCommand)
			}
		})
	}
}

func TestFileSessionStateStore_UnusableStateGrantsNothing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		content    string
		wantLegacy []string
		wantErr    bool
	}{
		{
			name:       "legacy unscoped overrides",
			content:    `{"sessionBypassOverrides":["CG-001"]}`,
			wantLegacy: []string{"CG-001"},
		},
		{name: "malformed file", content: `{"grants": [`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "session-state.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			store := NewFileSessionStateStore(path)
			if got := store.ActiveOverrides(scopeA, bypassNow); len(got.Session)+len(got.Command) != 0 {
				t.Errorf("ActiveOverrides = %+v, want none", got)
			}
			_, legacy, err := store.Grants(bypassNow)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Grants error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.Equal(legacy, tt.wantLegacy) {
				t.Errorf("legacy = %v, want %v", legacy, tt.wantLegacy)
			}
		})
	}
}

func TestFileSessionStateStore_AddGrantsReplacesAndPrunes(t *testing.T) {
	t.Parallel()

	store := newTestStore(t,
		mustGrant(t, "OLD", Session, scopeA, time.Minute, bypassNow),
		mustGrant(t, "KEEP", Session, scopeB, time.Hour, bypassNow),
		mustGrant(t, "R-1", Session, scopeA, time.Hour, bypassNow),
	)

	later := bypassNow.Add(30 * time.Minute)
	refreshed := mustGrant(t, "R-1", Session, scopeA, 2*time.Hour, later)
	if err := store.AddGrants([]BypassGrant{refreshed}, later); err != nil {
		t.Fatalf("AddGrants: %v", err)
	}

	grants, legacy, err := store.Grants(later)
	if err != nil {
		t.Fatalf("Grants: %v", err)
	}
	if len(legacy) != 0 {
		t.Errorf("legacy = %v, want none", legacy)
	}
	var ids []string
	for _, g := range grants {
		ids = append(ids, g.RuleID)
	}
	if !slices.Equal(ids, []string{"KEEP", "R-1"}) {
		t.Fatalf("grants = %v, want [KEEP R-1] (expired OLD pruned, R-1 replaced)", ids)
	}
	if !grants[1].ExpiresAt.Equal(refreshed.ExpiresAt) {
		t.Errorf("R-1 expires %s, want the refreshed %s", grants[1].ExpiresAt, refreshed.ExpiresAt)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(store.path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("state file mode = %o, want 600", perm)
		}
	}
}

func TestFileSessionStateStore_ConsumeCommandTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scope   BypassScope
		ids     []string
		now     time.Time
		wantErr bool
		// wantLeft are the rule IDs still active for scopeA afterwards.
		wantLeftCommand []string
	}{
		{name: "consumes one token", scope: scopeA, ids: []string{"CMD-1"}, now: bypassNow, wantLeftCommand: []string{"CMD-2"}},
		{name: "consumes several tokens", scope: scopeA, ids: []string{"CMD-1", "CMD-2"}, now: bypassNow},
		{name: "nothing to consume", scope: scopeA, now: bypassNow, wantLeftCommand: []string{"CMD-1", "CMD-2"}},
		{name: "other session cannot spend the token", scope: scopeASess2, ids: []string{"CMD-1"}, now: bypassNow, wantErr: true, wantLeftCommand: []string{"CMD-1", "CMD-2"}},
		{name: "session grant is not a token", scope: scopeA, ids: []string{"SESS-1"}, now: bypassNow, wantErr: true, wantLeftCommand: []string{"CMD-1", "CMD-2"}},
		{name: "all or nothing", scope: scopeA, ids: []string{"CMD-1", "MISSING"}, now: bypassNow, wantErr: true, wantLeftCommand: []string{"CMD-1", "CMD-2"}},
		{name: "expired token", scope: scopeA, ids: []string{"CMD-1"}, now: bypassNow.Add(2 * time.Hour), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := newTestStore(t,
				mustGrant(t, "CMD-1", Command, scopeA, time.Hour, bypassNow),
				mustGrant(t, "CMD-2", Command, scopeA, time.Hour, bypassNow),
				mustGrant(t, "SESS-1", Session, scopeA, time.Hour, bypassNow),
			)
			err := store.ConsumeCommandTokens(tt.scope, tt.ids, tt.now)
			if tt.wantErr {
				if !errors.Is(err, ErrBypassTokenUnavailable) {
					t.Fatalf("err = %v, want ErrBypassTokenUnavailable", err)
				}
			} else if err != nil {
				t.Fatalf("ConsumeCommandTokens: %v", err)
			}
			got := store.ActiveOverrides(scopeA, tt.now)
			if !slices.Equal(got.Command, tt.wantLeftCommand) {
				t.Errorf("remaining tokens = %v, want %v", got.Command, tt.wantLeftCommand)
			}
		})
	}
}

// TestFileSessionStateStore_TokenIsOneShotUnderConcurrency checks that when
// parallel tool calls race to redeem the same one-shot token, exactly one
// succeeds.
func TestFileSessionStateStore_TokenIsOneShotUnderConcurrency(t *testing.T) {
	t.Parallel()

	store := newTestStore(t, mustGrant(t, "CMD-1", Command, scopeA, time.Hour, bypassNow))

	const callers = 12
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		successes int
	)
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each caller uses its own store, like separate hook processes.
			err := NewFileSessionStateStore(store.path).ConsumeCommandTokens(scopeA, []string{"CMD-1"}, bypassNow)
			if err != nil && !errors.Is(err, ErrBypassTokenUnavailable) {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("%d callers redeemed the one-shot token, want exactly 1", successes)
	}
}

// TestFileSessionStateStore_StuckLockFailsClosed checks that redeeming a
// token while another holder sits on the state lock fails after a bounded
// wait (so the hook blocks the call inside its own timeout) and leaves the
// token unspent.
func TestFileSessionStateStore_StuckLockFailsClosed(t *testing.T) {
	t.Parallel()

	store := newTestStore(t, mustGrant(t, "CMD-1", Command, scopeA, time.Hour, bypassNow))
	store.lockTimeout = 50 * time.Millisecond

	held, err := fileutil.LockExclusive(context.Background(), store.path+".lock")
	if err != nil {
		t.Fatalf("LockExclusive: %v", err)
	}
	defer func() { _ = held.Unlock() }()

	err = store.ConsumeCommandTokens(scopeA, []string{"CMD-1"}, bypassNow)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ConsumeCommandTokens with the lock held: err = %v, want context.DeadlineExceeded", err)
	}
	if got := store.ActiveOverrides(scopeA, bypassNow); !slices.Equal(got.Command, []string{"CMD-1"}) {
		t.Errorf("tokens after the failed redemption = %v, want [CMD-1]", got.Command)
	}
}

func TestFileSessionStateStore_Clear(t *testing.T) {
	t.Parallel()

	store := newTestStore(t, mustGrant(t, "SESS-1", Session, scopeA, time.Hour, bypassNow))
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if got := store.ActiveOverrides(scopeA, bypassNow); len(got.Session) != 0 {
		t.Errorf("overrides after Clear = %+v, want none", got)
	}
	if err := store.Clear(); err != nil {
		t.Errorf("Clear of an absent state: %v", err)
	}
}

// TestEvaluate_BypassSemantics pins how grants lift rules in Evaluate: a
// session grant lifts only a session-tier rule; a command token lifts only a
// command-tier rule, only for a call the rule matches, and is reported as
// consumed only when the call is allowed.
func TestEvaluate_BypassSemantics(t *testing.T) {
	t.Parallel()

	matchBash := Condition{Type: ToolMatch, ToolName: "Bash"}
	neverMatch := Condition{Type: All, Conditions: []Condition{
		{Type: ToolMatch, ToolName: "Bash"},
		{Type: Not, Condition: &Condition{Type: ToolMatch, ToolName: "Bash"}},
	}}

	rule := func(id string, tier BypassTier, cond Condition) PolicyRule {
		r := makeRule(id, tier, High, ToolMatch, Block)
		r.Conditions = cond
		return r
	}

	tests := []struct {
		name         string
		rules        []PolicyRule
		overrides    ActiveOverrides
		wantBlock    string
		wantConsumed []string
	}{
		{
			name:      "session grant does not lift a command-tier rule",
			rules:     []PolicyRule{rule("CMD-1", Command, matchBash)},
			overrides: ActiveOverrides{Session: []string{"CMD-1"}},
			wantBlock: "CMD-1",
		},
		{
			name:      "command token does not lift a session-tier rule",
			rules:     []PolicyRule{rule("SESS-1", Session, matchBash)},
			overrides: ActiveOverrides{Command: []string{"SESS-1"}},
			wantBlock: "SESS-1",
		},
		{
			name:         "matching command-tier rule spends its token",
			rules:        []PolicyRule{rule("CMD-1", Command, matchBash)},
			overrides:    ActiveOverrides{Command: []string{"CMD-1"}},
			wantConsumed: []string{"CMD-1"},
		},
		{
			name:      "non-matching command-tier rule keeps its token",
			rules:     []PolicyRule{rule("CMD-1", Command, neverMatch)},
			overrides: ActiveOverrides{Command: []string{"CMD-1"}},
		},
		{
			name: "token is not reported when another rule blocks the call",
			rules: []PolicyRule{
				rule("CMD-1", Command, matchBash),
				rule("CMD-2", Command, matchBash),
			},
			overrides: ActiveOverrides{Command: []string{"CMD-1"}},
			wantBlock: "CMD-2",
		},
		{
			name: "monitor-mode rule does not spend a token",
			rules: func() []PolicyRule {
				r := rule("CMD-1", Command, matchBash)
				r.MonitorMode = true
				return []PolicyRule{r}
			}(),
			overrides: ActiveOverrides{Command: []string{"CMD-1"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			set := compileTestPolicy(t, makePolicy(tt.rules...))
			d := Evaluate(set, &EvalContext{ToolName: "Bash", Command: "ls", Overrides: tt.overrides})
			if tt.wantBlock != "" {
				if d.Action != Block || d.RuleID != tt.wantBlock {
					t.Fatalf("decision = %s by %q, want block by %q", d.Action, d.RuleID, tt.wantBlock)
				}
				if len(d.ConsumedTokens) != 0 {
					t.Errorf("blocked decision reports consumed tokens %v", d.ConsumedTokens)
				}
				return
			}
			if d.Action == Block {
				t.Fatalf("unexpected block by %s", d.RuleID)
			}
			if !slices.Equal(d.ConsumedTokens, tt.wantConsumed) {
				t.Errorf("ConsumedTokens = %v, want %v", d.ConsumedTokens, tt.wantConsumed)
			}
		})
	}
}

func TestPolicyEngine_ConsumeCommandTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		state   SessionStateReader
		ids     []string
		wantErr bool
	}{
		{name: "no tokens needed", state: nil},
		{name: "store without token support fails closed", state: scopeOnlyReader{}, ids: []string{"CMD-1"}, wantErr: true},
		{name: "static reader redeems its token", state: &StaticSessionStateReader{Command: []string{"CMD-1"}}, ids: []string{"CMD-1"}},
		{name: "missing token fails", state: &StaticSessionStateReader{}, ids: []string{"CMD-1"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			engine, err := NewPolicyEngine([]string{"testdata/valid-policy.yaml"}, tt.state, EngineOptions{})
			if err != nil {
				t.Fatalf("NewPolicyEngine: %v", err)
			}
			err = engine.ConsumeCommandTokens(&EvalContext{ProjectRoot: "/p", SessionID: "s"}, tt.ids)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrBypassTokenUnavailable) {
				t.Errorf("err = %v, want ErrBypassTokenUnavailable", err)
			}
		})
	}
}

// scopeOnlyReader resolves overrides but cannot redeem tokens.
type scopeOnlyReader struct{}

func (scopeOnlyReader) ActiveOverrides(BypassScope, time.Time) ActiveOverrides {
	return ActiveOverrides{}
}
