# Deploy

Deploy the current branch to staging or production via the CI/CD pipeline.

## Pre-flight checks

1. Confirm you are on the correct branch. Run `git branch --show-current` and
   verify it matches the expected release branch or `main`.
2. Ensure the working tree is clean: `git status --porcelain` must produce no
   output. If there are uncommitted changes, stop and ask the user whether to
   commit or stash them.
3. Verify the branch is up to date with the remote:
   `git fetch origin && git diff HEAD..origin/$(git branch --show-current) --stat`.
   If behind, prompt the user to pull first.

## Determine target environment

- Ask the user for the target: **staging** or **production**.
- For production deployments, require explicit confirmation before proceeding.
- Check for an existing deployment configuration file (e.g., `.github/workflows/deploy.yml`,
  `Makefile` deploy target, `fly.toml`, `render.yaml`, `vercel.json`, `Procfile`).

## Run tests and checks

1. Execute the project's full test suite. Use the build system detected in the
   project (e.g., `go test ./...`, `npm test`, `cargo test`, `pytest`).
2. Run the linter if configured (e.g., `golangci-lint run`, `npm run lint`).
3. If any test or lint step fails, stop immediately and report the failures.
   Do not proceed with deployment when tests are red.

## Trigger deployment

- If a CI pipeline is configured, trigger it via the appropriate mechanism:
  - GitHub Actions: `gh workflow run <workflow> --ref <branch>`
  - Makefile: `make deploy-<env>`
  - Platform CLI: `fly deploy`, `vercel --prod`, `render deploy`
- If no CI pipeline exists, report that no deployment mechanism was found and
  suggest the user configure one.

## Post-deployment verification

1. Wait for the deployment to complete. Poll the CI status or platform dashboard.
2. Run a basic health check against the deployed environment:
   - HTTP endpoint: `curl -sf <health-url>` should return 200.
   - CLI check: any project-specific smoke test command.
3. Report the deployment status: success with URL, or failure with logs.

## Rollback procedure

If the health check fails after deployment:

1. Identify the previous stable deployment (last successful commit/tag).
2. Trigger a rollback to that version using the same deployment mechanism.
3. Verify the rollback succeeded with another health check.
4. Report the rollback outcome and recommend the user investigate the failure.
