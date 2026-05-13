package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
