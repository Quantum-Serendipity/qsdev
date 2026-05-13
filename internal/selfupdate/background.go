package selfupdate

import (
	"fmt"
	"os"
	"time"
)

// BackgroundCheck starts a goroutine that checks for updates with a 5-second
// timeout. It returns a channel that will receive at most one message: a
// human-readable update notice string. If no update is available or the
// check is suppressed, the channel receives nothing and is closed.
//
// Returns nil if QSDEV_NO_UPDATE_CHECK=1 is set.
func BackgroundCheck(currentVersion string) <-chan string {
	if os.Getenv("QSDEV_NO_UPDATE_CHECK") == "1" {
		return nil
	}

	ch := make(chan string, 1)
	go func() {
		defer close(ch)

		cfg := DefaultConfig()

		done := make(chan *Release, 1)
		go func() {
			release, err := CheckForUpdate(cfg, currentVersion)
			if err != nil || release == nil {
				done <- nil
				return
			}
			done <- release
		}()

		// Wait up to 5 seconds for the check to complete.
		select {
		case release := <-done:
			if release != nil {
				notice := fmt.Sprintf(
					"A new version of qsdev is available: %s (current: %s)\nRun 'qsdev self-update' to update.",
					release.Version, currentVersion,
				)
				ch <- notice
			}
		case <-time.After(5 * time.Second):
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
