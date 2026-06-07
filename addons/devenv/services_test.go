package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestServiceToTemplateData_Postgres(t *testing.T) {
	svc := types.ServiceChoice{
		Name:    "postgres",
		Version: "16",
		Settings: map[string]string{
			"initial_db": "myapp_dev",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "PostgreSQL" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "PostgreSQL")
	}
	if got.NixName != "postgres" {
		t.Errorf("NixName = %q, want %q", got.NixName, "postgres")
	}

	assertContains(t, got.ConfigLines, `package = pkgs.postgresql_16;`)
	assertContains(t, got.ConfigLines, `initialDatabases = [{ name = "myapp_dev"; }];`)
}

func TestServiceToTemplateData_PostgresNoSettings(t *testing.T) {
	svc := types.ServiceChoice{Name: "postgres"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_Redis(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "redis",
		Settings: map[string]string{
			"port": "6380",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "Redis" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Redis")
	}
	if got.NixName != "redis" {
		t.Errorf("NixName = %q, want %q", got.NixName, "redis")
	}

	assertContains(t, got.ConfigLines, `port = 6380;`)
}

func TestServiceToTemplateData_RedisNoPort(t *testing.T) {
	svc := types.ServiceChoice{Name: "redis"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_MySQLMariaDB(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "mysql",
		Settings: map[string]string{
			"package":    "mariadb",
			"initial_db": "testdb",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "MySQL" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "MySQL")
	}
	if got.NixName != "mysql" {
		t.Errorf("NixName = %q, want %q", got.NixName, "mysql")
	}

	assertContains(t, got.ConfigLines, `package = pkgs.mariadb;`)
	assertContains(t, got.ConfigLines, `initialDatabases = [{ name = "testdb"; }];`)
}

func TestServiceToTemplateData_MySQLDefault(t *testing.T) {
	svc := types.ServiceChoice{Name: "mysql"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No mariadb package, no initial_db.
	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_MongoDB(t *testing.T) {
	svc := types.ServiceChoice{Name: "mongodb"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "MongoDB" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "MongoDB")
	}
	if got.NixName != "mongodb" {
		t.Errorf("NixName = %q, want %q", got.NixName, "mongodb")
	}
	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_Elasticsearch(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "elasticsearch",
		Settings: map[string]string{
			"cluster_name": "dev-cluster",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "Elasticsearch" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Elasticsearch")
	}
	if got.NixName != "elasticsearch" {
		t.Errorf("NixName = %q, want %q", got.NixName, "elasticsearch")
	}

	assertContains(t, got.ConfigLines, `cluster_name = "dev-cluster";`)
}

func TestServiceToTemplateData_ElasticsearchNoCluster(t *testing.T) {
	svc := types.ServiceChoice{Name: "elasticsearch"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_RabbitMQ(t *testing.T) {
	svc := types.ServiceChoice{Name: "rabbitmq"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "RabbitMQ" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "RabbitMQ")
	}
	if got.NixName != "rabbitmq" {
		t.Errorf("NixName = %q, want %q", got.NixName, "rabbitmq")
	}
	if len(got.ConfigLines) != 0 {
		t.Errorf("ConfigLines = %v, want empty", got.ConfigLines)
	}
}

func TestServiceToTemplateData_UnknownService(t *testing.T) {
	svc := types.ServiceChoice{Name: "cassandra"}

	_, err := devenv.ExportServiceToTemplateData(svc)
	if err == nil {
		t.Fatal("expected error for unknown service, got nil")
	}
}

// --- Kafka ---

func TestServiceToTemplateData_KafkaDefaults(t *testing.T) {
	svc := types.ServiceChoice{Name: "kafka"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "Kafka" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Kafka")
	}
	if got.NixName != "kafka" {
		t.Errorf("NixName = %q, want %q", got.NixName, "kafka")
	}

	assertContains(t, got.ConfigLines, `settings.listeners = "PLAINTEXT://127.0.0.1:9092";`)
	assertContains(t, got.ConfigLines, `settings.defaultMode = "kraft";`)
	assertContains(t, got.ConfigLines, `settings."auto.create.topics.enable" = true;`)
	assertContains(t, got.ConfigLines, `settings."num.partitions" = 1;`)
}

func TestServiceToTemplateData_KafkaKRaftMode(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "kafka",
		Settings: map[string]string{"mode": "kraft"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `settings.defaultMode = "kraft";`)
}

func TestServiceToTemplateData_KafkaZooKeeperMode(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "kafka",
		Settings: map[string]string{"mode": "zookeeper"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `settings.defaultMode = "zookeeper";`)
}

func TestServiceToTemplateData_KafkaCustomPort(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "kafka",
		Settings: map[string]string{"port": "9093"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `settings.listeners = "PLAINTEXT://127.0.0.1:9093";`)
}

// --- MinIO ---

func TestServiceToTemplateData_MinIODefaults(t *testing.T) {
	svc := types.ServiceChoice{Name: "minio"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "MinIO" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "MinIO")
	}
	if got.NixName != "minio" {
		t.Errorf("NixName = %q, want %q", got.NixName, "minio")
	}

	assertContains(t, got.ConfigLines, `listenAddress = "127.0.0.1:9000";`)
	assertContains(t, got.ConfigLines, `consoleAddress = "127.0.0.1:9001";`)
	assertContains(t, got.ConfigLines, `accessKey = "minioadmin";`)
	assertContains(t, got.ConfigLines, `secretKey = "minioadmin";`)
}

func TestServiceToTemplateData_MinIOCustomPorts(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "minio",
		Settings: map[string]string{
			"api_port":     "9100",
			"console_port": "9101",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `listenAddress = "127.0.0.1:9100";`)
	assertContains(t, got.ConfigLines, `consoleAddress = "127.0.0.1:9101";`)
}

func TestServiceToTemplateData_MinIOEnvVars(t *testing.T) {
	svc := types.ServiceChoice{Name: "minio"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEnvVar(t, got.EnvVars, "AWS_ENDPOINT_URL", "http://127.0.0.1:9000")
	assertEnvVar(t, got.EnvVars, "AWS_ACCESS_KEY_ID", "minioadmin")
	assertEnvVar(t, got.EnvVars, "AWS_SECRET_ACCESS_KEY", "minioadmin")
	assertEnvVar(t, got.EnvVars, "MINIO_ROOT_USER", "minioadmin")
	assertEnvVar(t, got.EnvVars, "MINIO_ROOT_PASSWORD", "minioadmin")
}

func TestServiceToTemplateData_MinIOLocalhostBinding(t *testing.T) {
	svc := types.ServiceChoice{Name: "minio"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, line := range got.ConfigLines {
		if contains127(line) {
			return
		}
	}
	t.Error("expected at least one ConfigLine to bind to 127.0.0.1")
}

// --- Mailpit ---

func TestServiceToTemplateData_MailpitDefaults(t *testing.T) {
	svc := types.ServiceChoice{Name: "mailpit"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "Mailpit" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Mailpit")
	}
	if got.NixName != "mailpit" {
		t.Errorf("NixName = %q, want %q", got.NixName, "mailpit")
	}

	assertContains(t, got.ConfigLines, `smtpListenAddress = "127.0.0.1:1025";`)
	assertContains(t, got.ConfigLines, `uiListenAddress = "127.0.0.1:8025";`)
}

func TestServiceToTemplateData_MailpitCustomPorts(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "mailpit",
		Settings: map[string]string{
			"smtp_port": "2025",
			"ui_port":   "9025",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `smtpListenAddress = "127.0.0.1:2025";`)
	assertContains(t, got.ConfigLines, `uiListenAddress = "127.0.0.1:9025";`)
}

func TestServiceToTemplateData_MailpitEnvVars(t *testing.T) {
	svc := types.ServiceChoice{Name: "mailpit"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEnvVar(t, got.EnvVars, "SMTP_HOST", "127.0.0.1")
	assertEnvVar(t, got.EnvVars, "SMTP_PORT", "1025")
	assertEnvVar(t, got.EnvVars, "MAIL_FROM", "dev@localhost")
	assertEnvVar(t, got.EnvVars, "MAILPIT_URL", "http://127.0.0.1:8025")
}

func TestServiceToTemplateData_MailpitMaxMessages(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "mailpit",
		Settings: map[string]string{"max_messages": "1000"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `additionalArgs = [ "--max" "1000" ];`)
}

func TestServiceToTemplateData_MailpitScript(t *testing.T) {
	svc := types.ServiceChoice{Name: "mailpit"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(got.Scripts))
	}
	if got.Scripts[0].Name != "open-mailpit" {
		t.Errorf("script name = %q, want %q", got.Scripts[0].Name, "open-mailpit")
	}
}

// --- Keycloak ---

func TestServiceToTemplateData_KeycloakDefaults(t *testing.T) {
	svc := types.ServiceChoice{Name: "keycloak"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "Keycloak" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Keycloak")
	}
	if got.NixName != "keycloak" {
		t.Errorf("NixName = %q, want %q", got.NixName, "keycloak")
	}

	assertContains(t, got.ConfigLines, `settings.http-port = 8080;`)
	assertContains(t, got.ConfigLines, `settings.hostname = "127.0.0.1";`)
	assertContains(t, got.ConfigLines, `initialAdminPassword = "admin";`)
	assertContains(t, got.ConfigLines, `database.type = "dev-file";`)
}

func TestServiceToTemplateData_KeycloakCustomPort(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "keycloak",
		Settings: map[string]string{"http_port": "8180"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `settings.http-port = 8180;`)
	assertEnvVar(t, got.EnvVars, "KEYCLOAK_URL", "http://127.0.0.1:8180")
}

func TestServiceToTemplateData_KeycloakCustomRealm(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "keycloak",
		Settings: map[string]string{"realm": "staging"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEnvVar(t, got.EnvVars, "OIDC_ISSUER_URL", "http://127.0.0.1:8080/realms/staging")
	assertEnvVar(t, got.EnvVars, "OIDC_CLIENT_ID", "staging-client")
}

func TestServiceToTemplateData_KeycloakEnvVars(t *testing.T) {
	svc := types.ServiceChoice{Name: "keycloak"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEnvVar(t, got.EnvVars, "KEYCLOAK_URL", "http://127.0.0.1:8080")
	assertEnvVar(t, got.EnvVars, "OIDC_ISSUER_URL", "http://127.0.0.1:8080/realms/development")
	assertEnvVar(t, got.EnvVars, "OIDC_CLIENT_ID", "development-client")
	assertEnvVar(t, got.EnvVars, "KEYCLOAK_ADMIN", "admin")
}

func TestServiceToTemplateData_KeycloakScript(t *testing.T) {
	svc := types.ServiceChoice{Name: "keycloak"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(got.Scripts))
	}
	if got.Scripts[0].Name != "open-keycloak" {
		t.Errorf("script name = %q, want %q", got.Scripts[0].Name, "open-keycloak")
	}
}

// --- NATS ---

func TestServiceToTemplateData_NATSDefaults(t *testing.T) {
	svc := types.ServiceChoice{Name: "nats"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.DisplayName != "NATS" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "NATS")
	}
	if got.NixName != "nats" {
		t.Errorf("NixName = %q, want %q", got.NixName, "nats")
	}

	assertContains(t, got.ConfigLines, `port = 4222;`)
	assertContains(t, got.ConfigLines, `settings.max_payload = 1048576;`)
	assertContains(t, got.ConfigLines, `monitoring.enable = true;`)
	assertContains(t, got.ConfigLines, `monitoring.port = 8222;`)
	assertContains(t, got.ConfigLines, `jetstream.enable = true;`)
	assertContains(t, got.ConfigLines, `jetstream.storeDir = "$DEVENV_STATE/nats";`)
}

func TestServiceToTemplateData_NATSJetStreamDisabled(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "nats",
		Settings: map[string]string{"jetstream": "false"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, line := range got.ConfigLines {
		if line == "jetstream.enable = true;" {
			t.Error("expected jetstream.enable to not be present when disabled")
		}
	}

	if _, ok := got.EnvVars["NATS_JETSTREAM"]; ok {
		t.Error("expected NATS_JETSTREAM to not be set when JetStream is disabled")
	}
}

func TestServiceToTemplateData_NATSCustomPorts(t *testing.T) {
	svc := types.ServiceChoice{
		Name: "nats",
		Settings: map[string]string{
			"port":      "4223",
			"http_port": "8223",
		},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `port = 4223;`)
	assertContains(t, got.ConfigLines, `monitoring.port = 8223;`)
	assertEnvVar(t, got.EnvVars, "NATS_URL", "nats://127.0.0.1:4223")
}

func TestServiceToTemplateData_NATSEnvVars(t *testing.T) {
	svc := types.ServiceChoice{Name: "nats"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertEnvVar(t, got.EnvVars, "NATS_URL", "nats://127.0.0.1:4222")
	assertEnvVar(t, got.EnvVars, "NATS_JETSTREAM", "true")
}

func TestServiceToTemplateData_NATSMaxPayload(t *testing.T) {
	svc := types.ServiceChoice{
		Name:     "nats",
		Settings: map[string]string{"max_payload": "2097152"},
	}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContains(t, got.ConfigLines, `settings.max_payload = 2097152;`)
}

func TestServiceToTemplateData_NATSScript(t *testing.T) {
	svc := types.ServiceChoice{Name: "nats"}

	got, err := devenv.ExportServiceToTemplateData(svc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(got.Scripts))
	}
	if got.Scripts[0].Name != "nats-monitor" {
		t.Errorf("script name = %q, want %q", got.Scripts[0].Name, "nats-monitor")
	}
}

// --- Test helpers ---

// assertContains checks that needle appears in the haystack slice.
func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, item := range haystack {
		if item == needle {
			return
		}
	}
	t.Errorf("expected %v to contain %q", haystack, needle)
}

func assertEnvVar(t *testing.T, envVars map[string]string, key, want string) {
	t.Helper()
	got, ok := envVars[key]
	if !ok {
		t.Errorf("env var %q not set", key)
		return
	}
	if got != want {
		t.Errorf("env var %q = %q, want %q", key, got, want)
	}
}

func contains127(s string) bool {
	return len(s) > 0 && (s[0] != 0) && containsSubstring(s, "127.0.0.1")
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
