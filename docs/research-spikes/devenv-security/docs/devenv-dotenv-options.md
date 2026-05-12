# devenv.sh Dotenv Configuration
- **Source**: https://devenv.sh/reference/options/#dotenv
- **Retrieved**: 2026-05-12

## Available Options

1. **`dotenv.enable`** - Controls whether dotenv functionality is active
2. **`dotenv.disableHint`** - Suppresses informational messages about dotenv
3. **`dotenv.filename`** - Specifies which file to load (customizable beyond the default `.env`)

## How Devenv Handles .env Files

Devenv integrates environment variable management through dotenv support. The configuration appears under the "Integrations" section alongside direnv and other tools, suggesting it's an optional feature that can be enabled or disabled per project.

## Exposure and Security Considerations

**Where it's exposed:** The integration is documented alongside secretspec and direnv, indicating environment variables become available within the development shell environment.

**Security implications worth noting:**

- The `disableHint` option exists, suggesting users may want to suppress warnings -- potentially indicating sensitivity around file handling
- Placement alongside a "secretspec" integration (for secret management) suggests developers should be cautious about what goes in .env files
- The configurable filename option allows flexibility but also requires deliberate configuration to match your setup

**Best practice:** Since .env files typically contain credentials, they should be excluded from version control and protected appropriately on shared systems.
