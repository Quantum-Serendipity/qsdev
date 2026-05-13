package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// serviceToTemplateData converts a ServiceChoice from wizard answers into a
// ServiceTemplateData suitable for rendering in the devenv.nix template.
// It maps service names to their Nix attribute names and translates settings
// into Nix configuration lines.
func serviceToTemplateData(svc types.ServiceChoice) (ServiceTemplateData, error) {
	switch svc.Name {
	case "postgres":
		return buildPostgres(svc), nil
	case "redis":
		return buildRedis(svc), nil
	case "mysql":
		return buildMySQL(svc), nil
	case "mongodb":
		return buildMongoDB(svc), nil
	case "elasticsearch":
		return buildElasticsearch(svc), nil
	case "rabbitmq":
		return buildRabbitMQ(svc), nil
	default:
		return ServiceTemplateData{}, fmt.Errorf("unknown service: %q", svc.Name)
	}
}

func buildPostgres(svc types.ServiceChoice) ServiceTemplateData {
	data := ServiceTemplateData{
		DisplayName: "PostgreSQL",
		NixName:     "postgres",
	}

	if v := svc.Version; v != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("package = pkgs.postgresql_%s;", v))
	}

	if db := svc.Settings["initial_db"]; db != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("initialDatabases = [{ name = %q; }];", db))
	}

	return data
}

func buildRedis(svc types.ServiceChoice) ServiceTemplateData {
	data := ServiceTemplateData{
		DisplayName: "Redis",
		NixName:     "redis",
	}

	if port := svc.Settings["port"]; port != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("port = %s;", port))
	}

	return data
}

func buildMySQL(svc types.ServiceChoice) ServiceTemplateData {
	data := ServiceTemplateData{
		DisplayName: "MySQL",
		NixName:     "mysql",
	}

	if pkg := svc.Settings["package"]; pkg == "mariadb" {
		data.ConfigLines = append(data.ConfigLines,
			"package = pkgs.mariadb;")
	}

	if db := svc.Settings["initial_db"]; db != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("initialDatabases = [{ name = %q; }];", db))
	}

	return data
}

func buildMongoDB(_ types.ServiceChoice) ServiceTemplateData {
	return ServiceTemplateData{
		DisplayName: "MongoDB",
		NixName:     "mongodb",
	}
}

func buildElasticsearch(svc types.ServiceChoice) ServiceTemplateData {
	data := ServiceTemplateData{
		DisplayName: "Elasticsearch",
		NixName:     "elasticsearch",
	}

	if cn := svc.Settings["cluster_name"]; cn != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("cluster_name = %q;", cn))
	}

	return data
}

func buildRabbitMQ(_ types.ServiceChoice) ServiceTemplateData {
	return ServiceTemplateData{
		DisplayName: "RabbitMQ",
		NixName:     "rabbitmq",
	}
}
