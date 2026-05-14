<!-- Source: https://raw.githubusercontent.com/freeCodeCamp/devdocs/main/Dockerfile -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Dockerfile Analysis

## Base Image & Environment
- Base: **Ruby 3.4.7** official image
- Character encoding: `LANG=C.UTF-8`
- Service worker enabled: `ENABLE_SERVICE_WORKER=true`

## Dependencies
System packages: `git`, `nodejs`, `libcurl4`
Ruby bundler for gem management. Caches aggressively cleared post-install.

## Build Process
1. Gemfile/Rakefile copied first (layer caching optimization)
2. Bundle dependencies installed with `path.system true`
3. Application code copied
4. `thor docs:download --all` retrieves ALL documentation
5. `thor assets:compile` processes into static assets
6. /tmp purged

## Runtime Configuration
- **Exposed Port:** 9292
- **Server:** Rackup (Ruby Rack application server)
- **Binding:** 0.0.0.0

## Key Observations
- The official Docker image downloads ALL docs (`--all`), making it large
- For selective docs, you'd need a custom build or post-start download
- The `--default` flag downloads only popular docs (much smaller)
- Alpine variant available: `ghcr.io/freecodecamp/devdocs:latest-alpine`
- Images update monthly automatically
