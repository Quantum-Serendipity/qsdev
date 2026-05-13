package devenv

import "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"

// ServiceSecretDeclarations returns the secret declarations for a given devenv
// service. Services that do not require secrets return nil.
func ServiceSecretDeclarations(serviceName string) []ecosystem.SecretDecl {
	switch serviceName {
	case "postgres":
		return []ecosystem.SecretDecl{
			{
				Name:        "DATABASE_URL",
				Description: "PostgreSQL connection string",
				Required:    true,
				Source:      "postgres",
			},
			{
				Name:         "POSTGRES_PASSWORD",
				Description:  "PostgreSQL superuser password",
				Required:     true,
				AutoGenerate: true,
				GenerateSpec: &ecosystem.GenerateSpec{
					Length:  32,
					Charset: "alphanumeric",
				},
				Source: "postgres",
			},
		}
	case "redis":
		return []ecosystem.SecretDecl{
			{
				Name:        "REDIS_URL",
				Description: "Redis connection string",
				Required:    true,
				Source:      "redis",
			},
		}
	case "mysql":
		return []ecosystem.SecretDecl{
			{
				Name:         "MYSQL_ROOT_PASSWORD",
				Description:  "MySQL root password",
				Required:     true,
				AutoGenerate: true,
				GenerateSpec: &ecosystem.GenerateSpec{
					Length:  32,
					Charset: "alphanumeric",
				},
				Source: "mysql",
			},
			{
				Name:        "MYSQL_URL",
				Description: "MySQL connection string",
				Required:    true,
				Source:      "mysql",
			},
		}
	case "rabbitmq":
		return []ecosystem.SecretDecl{
			{
				Name:         "RABBITMQ_DEFAULT_PASS",
				Description:  "RabbitMQ default user password",
				Required:     true,
				AutoGenerate: true,
				GenerateSpec: &ecosystem.GenerateSpec{
					Length:  32,
					Charset: "alphanumeric",
				},
				Source: "rabbitmq",
			},
		}
	default:
		return nil
	}
}
