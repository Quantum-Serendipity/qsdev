# devenv.sh JavaScript/Node.js Configuration
- **Source**: https://devenv.sh/supported-languages/javascript/
- **Retrieved**: 2026-05-12

## Core Enable Option

**languages.javascript.enable**
- Type: boolean
- Default: `false`
- Purpose: Activates JavaScript development tools within the environment

## Node.js Runtime Configuration

**languages.javascript.package**
- Type: package
- Default: `pkgs.nodejs-slim`

**languages.javascript.nodejs.enable**
- Type: boolean
- Default: `true`
- Purpose: Controls whether the Node.js runtime is available

## Package Manager Options

### npm
- **languages.javascript.npm.enable**: Installs npm (boolean, default: `false`)
- **languages.javascript.npm.package**: Defaults to Node.js npm output
- **languages.javascript.npm.install.enable**: Runs npm install automatically (boolean, default: `false`)

### pnpm
- **languages.javascript.pnpm.enable**: Installs pnpm (boolean, default: `false`)
- **languages.javascript.pnpm.package**: Default package: `pkgs.pnpm`
- **languages.javascript.pnpm.install.enable**: Auto-install capability (boolean, default: `false`)

### Yarn
- **languages.javascript.yarn.enable**: Installs yarn (boolean, default: `false`)
- **languages.javascript.yarn.package**: Default package: `pkgs.yarn`
- **languages.javascript.yarn.install.enable**: Automatic installation option (boolean, default: `false`)

### Bun
- **languages.javascript.bun.enable**: Installs bun runtime (boolean, default: `false`)
- **languages.javascript.bun.package**: Default: `pkgs.bun`
- **languages.javascript.bun.install.enable**: Auto-install during initialization (boolean, default: `false`)

## Additional Features

**languages.javascript.corepack.enable**
- Type: boolean
- Default: `false`
- Purpose: "Wrappers for npm, pnpm and Yarn via Node.js Corepack"

**languages.javascript.directory**
- Type: string
- Default: Project root
- Purpose: Specifies JavaScript project location, accepting absolute or relative paths

**languages.javascript.lsp.enable**
- Type: boolean
- Default: `true`
- Purpose: Enables TypeScript Language Server for JavaScript support

**languages.javascript.lsp.package**
- Type: package
- Default: `pkgs.typescript-language-server`
