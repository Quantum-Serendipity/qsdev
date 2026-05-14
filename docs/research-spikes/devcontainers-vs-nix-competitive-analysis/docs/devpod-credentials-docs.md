# DevPod — Credential Handling (Official Documentation)
- **Source**: https://devpod.sh/docs/developing-in-workspaces/credentials (fetched from raw GitHub)
- **Retrieved**: 2026-03-20

DevPod will automatically make certain local credentials available inside of the development container through a credentials helper. This allows you to reuse existing local credentials in a safe manner within the development container without explicitly configuring them inside each workspace. Currently DevPod supports this feature for git credentials and docker credentials.

## Git Credentials

DevPod will make HTTPS credentials available inside the dev container through a git credentials helper. SSH credentials are available through agent-forwarding that will be configured automatically on the SSH configuration for the workspace.

If you don't want DevPod to inject the credentials, you can disable that via:
```
devpod context set-options default -o SSH_INJECT_GIT_CREDENTIALS=false
```

## Docker Credentials

DevPod will make docker registry credentials available inside the dev container through a docker credentials helper. This allows you to pull and push images from and to private registries from within the dev container.

Disable with:
```
devpod context set-options default -o SSH_INJECT_DOCKER_CREDENTIALS=false
```

## GPG Credentials

DevPod will make gpg keys available inside the dev container through an SSH tunnel. This allows you to sign commits from inside the workspace.

Enable with:
```
devpod context set-options default -o GPG_AGENT_FORWARDING=true
```

Or when creating a workspace:
```
devpod up --gpg-agent-forwarding my-workspace
```

## Key Security Property

Credentials are never stored in workspaces — only forwarded when needed. The forwarding uses the secure tunnel (SSH over vendor-specific API) established between client and agent.
