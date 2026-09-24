package ecosystem

import "slices"

// OrderedLockFiles returns an ecosystem's lock file names from
// LockFilesByEcosystem, most faithful first: dedicated lockfiles come before
// files the catalog also lists among that ecosystem's manifests (e.g.
// requirements.txt), which can be hand-written, are often unpinned, and never
// describe the full resolved graph. The relative catalog order is otherwise
// kept. Consumers that pick the first lock file present (scanning, status
// reporting) therefore choose uv.lock over a loose requirements.txt. The
// catalog slices are never mutated.
func OrderedLockFiles(eco string) []string {
	manifests := ManifestsByEcosystem[eco]
	out := slices.Clone(LockFilesByEcosystem[eco])
	slices.SortStableFunc(out, func(a, b string) int {
		aManifest, bManifest := slices.Contains(manifests, a), slices.Contains(manifests, b)
		switch {
		case aManifest == bManifest:
			return 0
		case bManifest:
			return -1
		default:
			return 1
		}
	})
	return out
}
