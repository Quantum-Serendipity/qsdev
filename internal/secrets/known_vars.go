package secrets

// KnownCredentialVars is the canonical list of environment variable names
// that carry credentials or secrets. Used by the log redaction handler and
// the devenv addon's environment stripping.
var KnownCredentialVars = []string{
	// AWS
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",
	"AWS_DEFAULT_REGION",
	// GitHub
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GITHUB_PAT",
	// GitLab
	"GITLAB_TOKEN",
	"GL_TOKEN",
	// GCP
	"GOOGLE_APPLICATION_CREDENTIALS",
	"GCLOUD_PROJECT",
	"CLOUDSDK_CORE_PROJECT",
	// Azure
	"AZURE_CLIENT_ID",
	"AZURE_CLIENT_SECRET",
	"AZURE_TENANT_ID",
	"AZURE_SUBSCRIPTION_ID",
	// Package registries
	"NPM_TOKEN",
	"PYPI_TOKEN",
	// Docker
	"DOCKER_PASSWORD",
	"DOCKER_AUTH_CONFIG",
	// Nix
	"CACHIX_AUTH_TOKEN",
	// Databases
	"DATABASE_URL",
	"DATABASE_PASSWORD",
	"PGPASSWORD",
	"MYSQL_PWD",
	"REDIS_PASSWORD",
	// Secrets management
	"VAULT_TOKEN",
	// Third-party services
	"SENTRY_DSN",
	"STRIPE_SECRET_KEY",
	"SENDGRID_API_KEY",
	// Communication
	"SLACK_TOKEN",
	"SLACK_WEBHOOK_URL",
	// Generic
	"API_KEY",
	"API_SECRET",
	"SECRET_KEY",
	"PRIVATE_KEY",
	"ENCRYPTION_KEY",
}

// SensitiveKeyPatterns are substrings that, when found in a log attribute
// key name (case-insensitive, word-boundary matched), indicate the value
// should be redacted.
var SensitiveKeyPatterns = []string{
	"password",
	"secret",
	"token",
	"credential",
	"auth",
	"api_key",
	"apikey",
	"private_key",
	"bearer",
}
