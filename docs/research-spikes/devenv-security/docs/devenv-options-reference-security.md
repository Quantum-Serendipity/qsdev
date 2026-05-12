# devenv.nix Security-Related Configuration Options
- **Source**: https://devenv.sh/reference/options/
- **Retrieved**: 2026-05-12

## Certificate Management
- **certFile**: Configure certificate file path for SSL/TLS connections
- **keyFile**: Set private key file for SSL/TLS authentication
- **certificates**: Define trusted certificates for the environment

## Container Security
- **containers.<name>.enableLayerDeduplication**: Optimize container layers while maintaining security integrity
- **container.isBuilding**: Flag indicating container build status
- **containers.<name>.isBuilding**: Track individual container build state

## Process Isolation
- **processes.<name>.linux.capabilities**: "Configure Linux capabilities for process sandboxing"
- **processes.<name>.ready**: Health check mechanisms for process validation
- **processes.<name>.restart**: Control restart behavior with max attempts and windows

## Service-Level Security
- **services.keycloak.sslCertificate**: SSL certificate configuration for authentication service
- **services.keycloak.sslCertificateKey**: Private key for Keycloak SSL/TLS
- **services.elasticsearch.single_node**: Restrict Elasticsearch to single node for security

## Cache Security
- **cachix.enable**: Control access to binary cache
- **cachix.pull**: Configure cache retrieval permissions
- **cachix.push**: Manage cache publishing access

## Environment Isolation
- **devcontainer.enable**: Enable containerized development environments
- **secretspec.enable**: Manage secret specifications and providers
- **dotenv.enable**: Control environment variable loading from files

These options support sandboxing, certificate management, process isolation, and access controls within devenv configurations.
