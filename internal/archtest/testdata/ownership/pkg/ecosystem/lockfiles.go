package ecosystem

const NameGo = "go"

// LockFilesByEcosystem is the owned lockfile list. The string key below is
// an ecosystem id, not a lockfile name.
var LockFilesByEcosystem = map[string][]string{
	NameGo:    {"go.sum"},
	"fixture": {"fixture.lock"},
}

var own = []string{"fixture.lock", "go.sum"}
