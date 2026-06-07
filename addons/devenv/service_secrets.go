package devenv

import "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"

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
	case "kafka":
		return []ecosystem.SecretDecl{
			{
				Name:        "KAFKA_BOOTSTRAP_SERVERS",
				Description: "Kafka bootstrap server address",
				Required:    true,
				Source:      "kafka",
			},
		}
	case "minio":
		return []ecosystem.SecretDecl{
			{
				Name:         "MINIO_ROOT_PASSWORD",
				Description:  "MinIO root user password",
				Required:     true,
				AutoGenerate: true,
				GenerateSpec: &ecosystem.GenerateSpec{
					Length:  32,
					Charset: "alphanumeric",
				},
				Source: "minio",
			},
			{
				Name:        "AWS_SECRET_ACCESS_KEY",
				Description: "S3-compatible secret access key (same as MINIO_ROOT_PASSWORD for local dev)",
				Required:    true,
				Source:      "minio",
			},
		}
	case "keycloak":
		return []ecosystem.SecretDecl{
			{
				Name:         "KEYCLOAK_ADMIN_PASSWORD",
				Description:  "Keycloak admin console password",
				Required:     true,
				AutoGenerate: true,
				GenerateSpec: &ecosystem.GenerateSpec{
					Length:  32,
					Charset: "alphanumeric",
				},
				Source: "keycloak",
			},
		}
	default:
		return nil
	}
}
