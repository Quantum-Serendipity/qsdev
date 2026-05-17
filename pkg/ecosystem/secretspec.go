package ecosystem

// SecretDecl describes a single secret that an ecosystem module or service
// requires. It is used by the secret-spec generator to produce secretspec.toml.
type SecretDecl struct {
	Name         string
	Description  string
	Required     bool
	AutoGenerate bool
	GenerateSpec *GenerateSpec
	Source       string
}

// GenerateSpec describes how an auto-generated secret should be produced.
type GenerateSpec struct {
	Length  int
	Charset string // "alphanumeric", "hex", "base64", "uuid"
}

// SecretDeclarer is a supplementary interface that ecosystem modules may
// implement to declare the secrets they require. Modules that do not need
// secrets simply do not implement this interface.
type SecretDeclarer interface {
	SecretDeclarations(config ModuleConfig) []SecretDecl
}
