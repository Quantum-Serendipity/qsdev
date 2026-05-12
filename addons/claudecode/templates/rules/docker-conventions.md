# Docker Conventions

- Use multi-stage builds to keep final images small.
- Pin base image versions — never use `latest` in production Dockerfiles.
- Run containers as non-root. Add a `USER` instruction after installing packages.
- Order instructions from least to most frequently changed for layer caching.
- Use `.dockerignore` to exclude `.git/`, `node_modules/`, build artifacts, and secrets.
- Prefer `COPY` over `ADD` unless you need tar extraction or URL fetching.
- Combine `RUN` commands with `&&` to reduce layers. Clean caches in the same layer.
- Never embed secrets or credentials in Dockerfiles. Use build secrets or runtime env vars.
- Validate with `hadolint` before committing. Fix all warnings.
- Use `HEALTHCHECK` instructions for monitored services.
