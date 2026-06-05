package selfupdate

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// BackgroundCheck starts a goroutine that checks for updates with a 5-second
// timeout. It returns a channel that will receive at most one message: a
// human-readable update notice string. If no update is available or the
// check is suppressed, the channel receives nothing and is closed.
//
// Returns nil if QSDEV_NO_UPDATE_CHECK=1 is set.
func BackgroundCheck(currentVersion string) <-chan string {
	if os.Getenv(branding.Get().EnvNoUpdate) == "1" {
		return nil
	}

	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cfg := DefaultConfig()

		done := make(chan *Release, 1)
		go func() {
			release, err := CheckForUpdate(ctx, cfg, currentVersion)
			if err != nil || release == nil {
				done <- nil
				return
			}
			done <- release
		}()

		select {
		case release := <-done:
			if release != nil {
				app := branding.Get().AppName
				cur := strings.TrimPrefix(currentVersion, "v")
				notice := fmt.Sprintf(
					"A new version of %s is available: v%s (current: v%s)\nRun '%s self-update' to update.",
					app, release.Version, cur, app,
				)
				ch <- notice
			}
		case <-ctx.Done():
			// Timed out — silently skip.
		}
	}()

	return ch
}

// PrintNotice performs a non-blocking read from the channel returned by
// BackgroundCheck. If a notice is available, it prints it to stderr.
// If the channel is nil or empty, it returns immediately.
func PrintNotice(ch <-chan string) {
	if ch == nil {
		return
	}
	select {
	case notice, ok := <-ch:
		if ok && notice != "" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, notice)
		}
	default:
		// No notice available — don't block.
	}
}
