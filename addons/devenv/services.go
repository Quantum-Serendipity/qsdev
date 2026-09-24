package devenv

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// serviceToTemplateData converts a ServiceChoice from wizard answers into a
// ServiceTemplateData suitable for rendering in the devenv.nix template.
// It maps service names to their Nix attribute names and translates settings
// into Nix configuration lines.
//
// Settings come from answers files and .qsdev.yaml, so every value is either
// validated (ports, integers, booleans, enums) or emitted as an escaped Nix
// string literal; nothing is spliced into devenv.nix verbatim.
func serviceToTemplateData(svc types.ServiceChoice) (ServiceTemplateData, error) {
	var build func(types.ServiceChoice) (ServiceTemplateData, error)
	switch svc.Name {
	case "postgres":
		build = buildPostgres
	case "redis":
		build = buildRedis
	case "mysql":
		build = buildMySQL
	case "mongodb":
		build = buildMongoDB
	case "elasticsearch":
		build = buildElasticsearch
	case "rabbitmq":
		build = buildRabbitMQ
	case "kafka":
		build = buildKafka
	case "minio":
		build = buildMinIO
	case "mailpit":
		build = buildMailpit
	case "keycloak":
		build = buildKeycloak
	case "nats":
		build = buildNATS
	default:
		return ServiceTemplateData{}, fmt.Errorf("unknown service: %q", svc.Name)
	}

	return build(svc)
}

// postgresVersionRe matches the major-version suffix of pkgs.postgresql_<N>.
var postgresVersionRe = regexp.MustCompile(`^[0-9]+$`)

func buildPostgres(svc types.ServiceChoice) (ServiceTemplateData, error) {
	data := ServiceTemplateData{
		DisplayName: "PostgreSQL",
		NixName:     "postgres",
	}

	if v := svc.Version; v != "" {
		// The version becomes part of a Nix identifier, where string escaping
		// does not apply, so only a bare major version is accepted.
		if !postgresVersionRe.MatchString(v) {
			return ServiceTemplateData{}, fmt.Errorf("version %q must be a major version number (e.g. 16)", v)
		}
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("package = pkgs.postgresql_%s;", v))
	}

	if db := svc.Settings["initial_db"]; db != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("initialDatabases = [{ name = %s; }];", nixStr(db)))
	}

	return data, nil
}

func buildRedis(svc types.ServiceChoice) (ServiceTemplateData, error) {
	data := ServiceTemplateData{
		DisplayName: "Redis",
		NixName:     "redis",
	}

	if svc.Settings["port"] != "" {
		port, err := portSetting(svc.Settings, "port", "")
		if err != nil {
			return ServiceTemplateData{}, err
		}
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("port = %d;", port))
	}

	return data, nil
}

func buildMySQL(svc types.ServiceChoice) (ServiceTemplateData, error) {
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
			fmt.Sprintf("initialDatabases = [{ name = %s; }];", nixStr(db)))
	}

	return data, nil
}

func buildMongoDB(_ types.ServiceChoice) (ServiceTemplateData, error) {
	return ServiceTemplateData{
		DisplayName: "MongoDB",
		NixName:     "mongodb",
	}, nil
}

func buildElasticsearch(svc types.ServiceChoice) (ServiceTemplateData, error) {
	data := ServiceTemplateData{
		DisplayName: "Elasticsearch",
		NixName:     "elasticsearch",
	}

	if cn := svc.Settings["cluster_name"]; cn != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("cluster_name = %s;", nixStr(cn)))
	}

	return data, nil
}

func buildRabbitMQ(_ types.ServiceChoice) (ServiceTemplateData, error) {
	return ServiceTemplateData{
		DisplayName: "RabbitMQ",
		NixName:     "rabbitmq",
	}, nil
}

// kafkaModes are the values devenv accepts for services.kafka.defaultMode.
var kafkaModes = map[string]bool{"kraft": true, "zookeeper": true}

func buildKafka(svc types.ServiceChoice) (ServiceTemplateData, error) {
	port, err := portSetting(svc.Settings, "port", "9092")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	mode := settingOr(svc.Settings, "mode", "kraft")
	if !kafkaModes[mode] {
		return ServiceTemplateData{}, fmt.Errorf("setting \"mode\": %q must be \"kraft\" or \"zookeeper\"", mode)
	}
	autoCreate, err := boolSetting(svc.Settings, "auto_create_topics", "true")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	numPartitions, err := positiveIntSetting(svc.Settings, "num_partitions", "1")
	if err != nil {
		return ServiceTemplateData{}, err
	}

	data := ServiceTemplateData{
		DisplayName: "Kafka",
		NixName:     "kafka",
	}

	plaintext := fmt.Sprintf("PLAINTEXT://127.0.0.1:%d", port)
	listeners := []string{plaintext}
	var quorumVoters string
	if mode == "kraft" {
		// Overriding listeners replaces devenv's kraft default, which carries
		// the CONTROLLER listener KRaft needs, so it must be re-declared. The
		// quorum voter points at the same address the controller binds to
		// (devenv's default names localhost, which may resolve to ::1).
		controllerPort := kafkaControllerPort(port)
		listeners = append(listeners, fmt.Sprintf("CONTROLLER://127.0.0.1:%d", controllerPort))
		quorumVoters = fmt.Sprintf("1@127.0.0.1:%d", controllerPort)
	}

	// defaultMode is a top-level option; under settings it would only become
	// a stray server.properties key and the mode would be silently ignored.
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("defaultMode = %s;", nixStr(mode)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.listeners = %s;", nixStrList(listeners)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.%s = %s;", nixStr("advertised.listeners"), nixStrList([]string{plaintext})))
	if quorumVoters != "" {
		data.ConfigLines = append(data.ConfigLines,
			fmt.Sprintf("settings.%s = %s;", nixStr("controller.quorum.voters"), nixStr(quorumVoters)))
	}
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.%s = %t;", nixStr("auto.create.topics.enable"), autoCreate))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.%s = %d;", nixStr("num.partitions"), numPartitions))

	return data, nil
}

// kafkaControllerPort returns the KRaft controller port paired with the
// broker port: the next port up (9092 -> 9093, devenv's own default pairing),
// or the one below when the broker already uses the highest port.
func kafkaControllerPort(port int) int {
	if port == maxPort {
		return port - 1
	}
	return port + 1
}

func buildMinIO(svc types.ServiceChoice) (ServiceTemplateData, error) {
	apiPort, err := portSetting(svc.Settings, "api_port", "9000")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	consolePort, err := portSetting(svc.Settings, "console_port", "9001")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	rootUser := settingOr(svc.Settings, "root_user", "minioadmin")
	rootPassword := settingOr(svc.Settings, "root_password", "minioadmin")

	data := ServiceTemplateData{
		DisplayName: "MinIO",
		NixName:     "minio",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("listenAddress = %s;", nixStr(fmt.Sprintf("127.0.0.1:%d", apiPort))))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("consoleAddress = %s;", nixStr(fmt.Sprintf("127.0.0.1:%d", consolePort))))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("accessKey = %s;", nixStr(rootUser)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("secretKey = %s;", nixStr(rootPassword)))

	data.EnvVars = map[string]string{
		"AWS_ENDPOINT_URL":      fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		"AWS_ACCESS_KEY_ID":     rootUser,
		"AWS_SECRET_ACCESS_KEY": rootPassword,
		"MINIO_ROOT_USER":       rootUser,
		"MINIO_ROOT_PASSWORD":   rootPassword,
	}

	return data, nil
}

func buildMailpit(svc types.ServiceChoice) (ServiceTemplateData, error) {
	smtpPort, err := portSetting(svc.Settings, "smtp_port", "1025")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	uiPort, err := portSetting(svc.Settings, "ui_port", "8025")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	maxMessages, err := positiveIntSetting(svc.Settings, "max_messages", "500")
	if err != nil {
		return ServiceTemplateData{}, err
	}

	data := ServiceTemplateData{
		DisplayName: "Mailpit",
		NixName:     "mailpit",
	}

	uiURL := fmt.Sprintf("http://127.0.0.1:%d", uiPort)
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("smtpListenAddress = %s;", nixStr(fmt.Sprintf("127.0.0.1:%d", smtpPort))))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("uiListenAddress = %s;", nixStr(fmt.Sprintf("127.0.0.1:%d", uiPort))))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("additionalArgs = %s;", nixStrList([]string{"--max", strconv.Itoa(maxMessages)})))

	data.EnvVars = map[string]string{
		"SMTP_HOST":   "127.0.0.1",
		"SMTP_PORT":   strconv.Itoa(smtpPort),
		"MAIL_FROM":   "dev@localhost",
		"MAILPIT_URL": uiURL,
	}

	data.Scripts = []ServiceScript{
		{Name: "open-mailpit", Exec: openURLScript(uiURL)},
	}

	return data, nil
}

func buildKeycloak(svc types.ServiceChoice) (ServiceTemplateData, error) {
	httpPort, err := portSetting(svc.Settings, "http_port", "8080")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	adminUser := settingOr(svc.Settings, "admin_user", "admin")
	adminPassword := settingOr(svc.Settings, "admin_password", "admin")
	realm := settingOr(svc.Settings, "realm", "development")

	data := ServiceTemplateData{
		DisplayName: "Keycloak",
		NixName:     "keycloak",
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.http-port = %d;", httpPort))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.hostname = %s;", nixStr("127.0.0.1")))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("initialAdminPassword = %s;", nixStr(adminPassword)))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("database.type = %s;", nixStr("dev-file")))

	data.EnvVars = map[string]string{
		"KEYCLOAK_URL":    baseURL,
		"OIDC_ISSUER_URL": baseURL + "/realms/" + realm,
		"OIDC_CLIENT_ID":  realm + "-client",
		"KEYCLOAK_ADMIN":  adminUser,
	}

	data.Scripts = []ServiceScript{
		{Name: "open-keycloak", Exec: openURLScript(baseURL + "/admin")},
	}

	return data, nil
}

func buildNATS(svc types.ServiceChoice) (ServiceTemplateData, error) {
	port, err := portSetting(svc.Settings, "port", "4222")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	httpPort, err := portSetting(svc.Settings, "http_port", "8222")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	jetstream, err := boolSetting(svc.Settings, "jetstream", "true")
	if err != nil {
		return ServiceTemplateData{}, err
	}
	maxPayload, err := positiveIntSetting(svc.Settings, "max_payload", "1048576")
	if err != nil {
		return ServiceTemplateData{}, err
	}

	data := ServiceTemplateData{
		DisplayName: "NATS",
		NixName:     "nats",
	}

	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("port = %d;", port))
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("settings.max_payload = %d;", maxPayload))
	data.ConfigLines = append(data.ConfigLines,
		"monitoring.enable = true;")
	data.ConfigLines = append(data.ConfigLines,
		fmt.Sprintf("monitoring.port = %d;", httpPort))

	// devenv's nats module derives the JetStream store_dir from DEVENV_STATE
	// itself; only the enable switch is an option.
	if jetstream {
		data.ConfigLines = append(data.ConfigLines,
			"jetstream.enable = true;")
	}

	data.EnvVars = map[string]string{
		"NATS_URL": fmt.Sprintf("nats://127.0.0.1:%d", port),
	}
	if jetstream {
		data.EnvVars["NATS_JETSTREAM"] = "true"
	}

	data.Scripts = []ServiceScript{
		{Name: "nats-monitor", Exec: openURLScript(fmt.Sprintf("http://127.0.0.1:%d", httpPort))},
	}

	return data, nil
}

// openURLScript returns a shell command that opens url in the desktop browser
// on Linux (xdg-open) or macOS (open). url must be built from validated parts.
func openURLScript(url string) string {
	return "xdg-open " + url + " 2>/dev/null || open " + url
}

func settingOr(settings map[string]string, key, fallback string) string {
	if v, ok := settings[key]; ok && v != "" {
		return v
	}
	return fallback
}

// maxPort is the highest valid TCP port.
const maxPort = 65535

// portSetting returns the named setting (or fallback) as a TCP port number.
func portSetting(settings map[string]string, key, fallback string) (int, error) {
	v := settingOr(settings, key, fallback)
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > maxPort {
		return 0, fmt.Errorf("setting %q: %q is not a valid port (1-%d)", key, v, maxPort)
	}
	return n, nil
}

// positiveIntSetting returns the named setting (or fallback) as an integer >= 1.
func positiveIntSetting(settings map[string]string, key, fallback string) (int, error) {
	v := settingOr(settings, key, fallback)
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("setting %q: %q is not a positive integer", key, v)
	}
	return n, nil
}

// boolSetting returns the named setting (or fallback), which must be exactly
// "true" or "false".
func boolSetting(settings map[string]string, key, fallback string) (bool, error) {
	switch v := settingOr(settings, key, fallback); v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("setting %q: %q must be \"true\" or \"false\"", key, v)
	}
}

// nixStr renders s as an escaped Nix double-quoted string literal. Go's %q is
// not a substitute: it leaves Nix's "${" antiquotation live.
func nixStr(s string) string {
	return `"` + ecosystem.NixEscapeString(s) + `"`
}

// nixStrList renders items as a Nix list of escaped string literals.
func nixStrList(items []string) string {
	out := "["
	for _, item := range items {
		out += " " + nixStr(item)
	}
	return out + " ]"
}
