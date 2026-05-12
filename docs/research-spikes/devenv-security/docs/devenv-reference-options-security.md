<!-- Source: https://devenv.sh/reference/options/ -->
<!-- Retrieved: 2026-05-12 -->
<!-- Note: This is a filtered extraction of security-relevant options only -->

# Security-Relevant Configuration Options in devenv.nix

## Shell & Execution
- **enterShell**: Executes code when entering the development environment
- **enterTest**: Runs during test initialization
- **scripts.<name>.exec**: Specifies executable commands within named scripts

## Process Management
- **processes.<name>.exec**: Defines process execution commands
- **processes.<name>.env**: Sets environment variables for processes
- **processes.<name>.linux.capabilities**: Configures Linux capabilities for processes
- **processes.<name>.ready**: Health check configuration with timeout and threshold settings
- **process.manager.implementation**: Selects the process manager (hivemind, honcho, mprocs, overmind, process-compose)

## Authentication & Secrets
- **cachix.enable/push/pull**: Binary cache configuration with authentication
- **secretspec.enable**: Activates secret specification support
- **secretspec.provider**: Configures secret provider backend
- **secretspec.secrets**: Defines managed secrets

## File & Certificate Management
- **files.<name>.text/source**: Creates or references files in the environment
- **certFile**: References SSL certificate file location
- **keyFile**: References SSL private key file location
- **certificates**: Configures certificate settings

## Container Security
- **containers.<name>.layers.<name>.perms**: Manages file permissions (uid, gid, mode)
- **container.isBuilding**: Indicates container build status
