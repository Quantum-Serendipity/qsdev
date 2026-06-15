package contentsign

import (
	"context"
	"runtime"
	"sync"
)

// VerifyCorpus verifies every manifest entry with bounded parallelism
// (runtime.NumCPU() workers) and returns one VerificationResult per entry, in
// the same order as entries. It never returns an error: each per-entry outcome,
// including I/O failures, is recorded in that entry's result.
func VerifyCorpus(ctx context.Context, entries []ContentManifestEntry, opts VerifyOptions) []VerificationResult {
	results := make([]VerificationResult, len(entries))
	if len(entries) == 0 {
		return results
	}

	workers := min(runtime.NumCPU(), len(entries))

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, entry := range entries {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, entry ContentManifestEntry) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = verifyOne(ctx, entry, opts)
		}(i, entry)
	}
	wg.Wait()
	return results
}

// verifyOne runs VerifyEntry and folds any I/O error into the result so the
// corpus pass never propagates a Go error.
func verifyOne(ctx context.Context, entry ContentManifestEntry, opts VerifyOptions) VerificationResult {
	res, err := VerifyEntry(ctx, entry, opts)
	if err != nil {
		res.Status = StatusFailed
		res.Verified = false
		res.Reason = err.Error()
	}
	return res
}
