package projectctx

// Tier is the priority band a generic tool occupies. It maps directly onto
// spi.ToolRegistration.Tier, the ordering/grouping hint where lower is more
// core.
type Tier int

const (
	// TierCritical tools are fundamental project awareness.
	TierCritical Tier = 0
	// TierStandard tools are common introspection.
	TierStandard Tier = 1
	// TierExtended tools are specialized.
	TierExtended Tier = 2
)
