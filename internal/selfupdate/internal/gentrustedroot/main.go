// Command gentrustedroot refreshes the Sigstore public-good trusted root that
// qsdev embeds for in-process self-update signature verification.
//
// It fetches trusted_root.json through a full TUF update rooted in the TUF
// trust anchor shipped with the vendored sigstore-go module, so the file it
// writes is authenticated by the Sigstore root-signing ceremony rather than by
// the transport. Run it via `go generate ./internal/selfupdate` when Sigstore
// rotates Fulcio, Rekor, CT log or TSA keys, then review and commit the diff.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

const trustedRootTarget = "trusted_root.json"

func main() {
	out := flag.String("o", trustedRootTarget, "output path for the trusted root JSON")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintf(os.Stderr, "gentrustedroot: %v\n", err)
		os.Exit(1)
	}
}

func run(out string) error {
	// An in-memory TUF cache keeps the run hermetic: nothing is read from or
	// written to ~/.sigstore, so a stale or tampered local cache cannot leak
	// into the embedded root.
	client, err := tuf.New(tuf.DefaultOptions().WithDisableLocalCache())
	if err != nil {
		return fmt.Errorf("creating TUF client: %w", err)
	}
	data, err := client.GetTarget(trustedRootTarget)
	if err != nil {
		return fmt.Errorf("fetching %s via TUF: %w", trustedRootTarget, err)
	}
	// Refuse to write anything sigstore-go itself cannot load.
	if _, err := root.NewTrustedRootFromJSON(data); err != nil {
		return fmt.Errorf("parsing fetched trusted root: %w", err)
	}
	if err := fileutil.WriteFileAtomic(out, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", out, err)
	}
	return nil
}
