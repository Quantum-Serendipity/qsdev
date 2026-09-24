package projectctx

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ledgerCache serves the project's generated-file state ledger, re-reading it
// only when the state file's presence or modification time changes. The server
// is long-lived while `qsdev enable`/`qsdev disable` rewrite the ledger, so a
// snapshot taken once at startup would go stale.
type ledgerCache struct {
	path string

	mu      sync.Mutex
	loaded  bool
	present bool
	mtime   time.Time
	st      types.GeneratedState
}

func newLedgerCache(path string) *ledgerCache {
	return &ledgerCache{path: path, st: emptyLedger()}
}

func emptyLedger() types.GeneratedState {
	return types.GeneratedState{Files: map[string]types.FileState{}}
}

// current returns the ledger as it is now. An absent state file yields an empty
// ledger. When the file changed but cannot be read or parsed, it keeps serving
// the last good snapshot and returns a non-empty warning describing why.
func (c *ledgerCache) current() (types.GeneratedState, string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var mtime time.Time
	info, statErr := os.Stat(c.path)
	present := statErr == nil
	if present {
		mtime = info.ModTime()
	}
	if c.loaded && present == c.present && mtime.Equal(c.mtime) {
		return c.st, ""
	}

	st, err := state.LoadStateFromFile(c.path)
	if err != nil {
		return c.st, fmt.Sprintf("state file unreadable; serving the last good ledger: %v", err)
	}
	c.loaded, c.present, c.mtime, c.st = true, present, mtime, st
	return st, ""
}
