<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/package.json -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — package.json

```json
{
  "name": "@lateos/npm-scan",
  "version": "0.15.1",
  "description": "Modern npm supply chain security scanner — detects obfuscated payloads, credential stealers, conditional triggers, sandbox evasion, and worm-like propagation. 11 attack types, SBOM, NIST/EU CRA compliance reporting.",
  "main": "backend/index.js",
  "bin": {
    "npm-scan": "cli/cli.js"
  },
  "type": "module",
  "license": "Apache-2.0",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/lateos-ai/npm-scan.git"
  },
  "readme": "README.md",
  "keywords": [
    "npm",
    "security",
    "supply-chain",
    "scanner",
    "malware",
    "shai-hulud",
    "sbom",
    "compliance"
  ],
  "scripts": {
    "dev": "node cli/cli.js",
    "lint": "echo 'Lint stub'",
    "test": "node --test",
    "test:coverage": "node --experimental-test-coverage --test",
    "test:verbose": "node --test --test-reporter spec",
    "prepare": "husky",
    "build": "echo 'Build stub'",
    "corpus": "node tests/corpus/run.js"
  },
  "lint-staged": {
    "**/package{,-lock}.json": "sh -c 'node cli/cli.js scan-lockfile --fail-on high'",
    "**/yarn.lock": "sh -c 'node cli/cli.js scan-lockfile --fail-on high --yarn'",
    "**/pnpm-lock.yaml": "sh -c 'node cli/cli.js scan-lockfile --fail-on high --pnpm'"
  },
  "publishConfig": {
    "access": "public"
  },
  "dependencies": {
    "acorn": "^8.16.0",
    "commander": "^14.0.3",
    "glob": "^13.0.6",
    "js-yaml": "^4.1.1",
    "pdf-lib": "^1.17.1",
    "sql.js": "^1.11.0",
    "tar": "^7.5.15"
  },
  "devDependencies": {
    "husky": "^9.1.7",
    "lint-staged": "^16.4.0"
  }
}
```

## Analysis Notes

- **Entry point**: `backend/index.js` (library), `cli/cli.js` (CLI binary)
- **ESM modules** (`"type": "module"`)
- **Dependencies are minimal**: acorn (AST parsing), commander (CLI), glob (file matching), js-yaml (policy parsing), pdf-lib (PDF generation), sql.js (SQLite persistence), tar (tarball extraction)
- **No runtime dependencies on cloud services** — everything local
- **Build and lint scripts are stubs** — suggests the project doesn't use transpilation or a linter
- **Version 0.15.1** (package.json) vs **v1.0.0** (GitHub release tag) — version mismatch between npm and GitHub
