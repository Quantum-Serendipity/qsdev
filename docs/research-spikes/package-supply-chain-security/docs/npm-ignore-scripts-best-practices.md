# NPM Ignore-Scripts: Security Best Practices

- **Source**: https://www.nodejs-security.com/blog/npm-ignore-scripts-best-practices-as-security-mitigation-for-malicious-packages
- **Retrieved**: 2026-05-12

## How Ignore-Scripts Works

The `ignore-scripts` flag prevents execution of lifecycle hooks (like `postinstall` scripts) during package installation. As the article explains, "any maintainer and any package in your dependency tree may run any operating system commands during your own `npm install` process" without this protection.

You can implement it three ways:

1. **Per-installation**: `npm install --ignore-scripts <package-name>`
2. **Project-level** (via `.npmrc`): `echo "ignore-scripts=true" >> .npmrc`
3. **System-wide**: `npm config set ignore-scripts true`

## Real Security Incidents

The article cites two notable compromises:
- **eslint-scope**: A maintainer account was breached, leading to a version with malicious postinstall scripts designed to steal npm credentials
- **crossenv**: A typo-squatting package that would exfiltrate environment variables, exploiting user trust

## Functionality Trade-offs

Some legitimate packages require postinstall scripts to function:
- bcrypt
- node-sass
- sharp

Research shows approximately 2.2% of npm packages use install scripts, meaning most packages won't be affected by enabling this flag.

## Managed Allowlisting

For trusted packages requiring scripts, the article recommends using "LavaMoat's allow-script package to manage an allow-list of trusted packages."

## Testing Configuration

Use `@lavamoat/preinstall-always-fail` to verify your setup—successful installation indicates scripts are properly disabled.
