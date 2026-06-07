package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	case "kafka":
		return buildKafka(svc), nil
	case "minio":
		return buildMinIO(svc), nil
	case "mailpit":
		return buildMailpit(svc), nil
	case "keycloak":
		return buildKeycloak(svc), nil
	case "nats":
		return buildNATS(svc), nil
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
			fmt.Sprintf("package = pkgs.postgresql_%s;", ecosystem.NixEscapeString(v)))
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
			fmt.Sprintf("port = %s;", ecosystem.NixEscapeString(port)))
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

func buildKafka(svc types.ServiceChoice) ServiceTemplateData {
	port := settingOr(svc.Settings, "port", "9092")
	mode := settingOr(svc.Settings, "mode", "kraft")
	autoCreate := settingOr(svc.Settings, "auto_create_topics", "true")
	numPartitions := settingOr(svc.Settings, "num_partitions", "1")

	data := ServiceTemplateData{
		DisplayName: "Kafka",
		NixName:     "kafka",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.listeners = %q;", "PLAINTEXT://127.0.0.1:"+port))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.defaultMode = %q;", ecosystem.NixEscapeString(mode)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.%q = %s;", "auto.create.topics.enable", autoCreate))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.%q = %s;", "num.partitions", numPartitions))

	return data
}

func buildMinIO(svc types.ServiceChoice) ServiceTemplateData {
	apiPort := settingOr(svc.Settings, "api_port", "9000")
	consolePort := settingOr(svc.Settings, "console_port", "9001")
	rootUser := settingOr(svc.Settings, "root_user", "minioadmin")
	rootPassword := settingOr(svc.Settings, "root_password", "minioadmin")

	data := ServiceTemplateData{
		DisplayName: "MinIO",
		NixName:     "minio",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("listenAddress = %q;", "127.0.0.1:"+apiPort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("consoleAddress = %q;", "127.0.0.1:"+consolePort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("accessKey = %q;", rootUser))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("secretKey = %q;", rootPassword))

	data.EnvVars = map[string]string{
		"AWS_ENDPOINT_URL":      "http://127.0.0.1:" + apiPort,
		"AWS_ACCESS_KEY_ID":     rootUser,
		"AWS_SECRET_ACCESS_KEY": rootPassword,
		"MINIO_ROOT_USER":       rootUser,
		"MINIO_ROOT_PASSWORD":   rootPassword,
	}

	return data
}

func buildMailpit(svc types.ServiceChoice) ServiceTemplateData {
	smtpPort := settingOr(svc.Settings, "smtp_port", "1025")
	uiPort := settingOr(svc.Settings, "ui_port", "8025")
	maxMessages := settingOr(svc.Settings, "max_messages", "500")

	data := ServiceTemplateData{
		DisplayName: "Mailpit",
		NixName:     "mailpit",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("smtpListenAddress = %q;", "127.0.0.1:"+smtpPort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("uiListenAddress = %q;", "127.0.0.1:"+uiPort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("additionalArgs = [ \"--max\" %q ];", maxMessages))

	data.EnvVars = map[string]string{
		"SMTP_HOST":   "127.0.0.1",
		"SMTP_PORT":   smtpPort,
		"MAIL_FROM":   "dev@localhost",
		"MAILPIT_URL": "http://127.0.0.1:" + uiPort,
	}

	data.Scripts = []ServiceScript{
		{Name: "open-mailpit", Exec: "xdg-open http://127.0.0.1:" + uiPort + " 2>/dev/null || open http://127.0.0.1:" + uiPort},
	}

	return data
}

func buildKeycloak(svc types.ServiceChoice) ServiceTemplateData {
	httpPort := settingOr(svc.Settings, "http_port", "8080")
	adminUser := settingOr(svc.Settings, "admin_user", "admin")
	adminPassword := settingOr(svc.Settings, "admin_password", "admin")
	realm := settingOr(svc.Settings, "realm", "development")

	data := ServiceTemplateData{
		DisplayName: "Keycloak",
		NixName:     "keycloak",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.http-port = %s;", httpPort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.hostname = %q;", "127.0.0.1"))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("initialAdminPassword = %q;", ecosystem.NixEscapeString(adminPassword)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("database.type = %q;", "dev-file"))

	data.EnvVars = map[string]string{
		"KEYCLOAK_URL":    "http://127.0.0.1:" + httpPort,
		"OIDC_ISSUER_URL": "http://127.0.0.1:" + httpPort + "/realms/" + realm,
		"OIDC_CLIENT_ID":  realm + "-client",
		"KEYCLOAK_ADMIN":  adminUser,
	}

	data.Scripts = []ServiceScript{
		{Name: "open-keycloak", Exec: "xdg-open http://127.0.0.1:" + httpPort + "/admin 2>/dev/null || open http://127.0.0.1:" + httpPort + "/admin"},
	}

	return data
}

func buildNATS(svc types.ServiceChoice) ServiceTemplateData {
	port := settingOr(svc.Settings, "port", "4222")
	httpPort := settingOr(svc.Settings, "http_port", "8222")
	jetstream := settingOr(svc.Settings, "jetstream", "true")
	maxPayload := settingOr(svc.Settings, "max_payload", "1048576")

	data := ServiceTemplateData{
		DisplayName: "NATS",
		NixName:     "nats",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("port = %s;", port))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.max_payload = %s;", maxPayload))
	data.ConfigLines = append(data.ConfigLines,
		"monitoring.enable = true;")
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("monitoring.port = %s;", httpPort))

	if jetstream == "true" {
		data.ConfigLines = append(data.ConfigLines,
			"jetstream.enable = true;")
		data.ConfigLines = append(data.ConfigLines,
			`jetstream.storeDir = "$DEVENV_STATE/nats";`)
	}

	data.EnvVars = map[string]string{
		"NATS_URL": "nats://127.0.0.1:" + port,
	}
	if jetstream == "true" {
		data.EnvVars["NATS_JETSTREAM"] = "true"
	}

	data.Scripts = []ServiceScript{
		{Name: "nats-monitor", Exec: "xdg-open http://127.0.0.1:" + httpPort + " 2>/dev/null || open http://127.0.0.1:" + httpPort},
	}

	return data
}

func settingOr(settings map[string]string, key, fallback string) string {
	if v, ok := settings[key]; ok && v != "" {
		return v
	}
	return fallback
}
