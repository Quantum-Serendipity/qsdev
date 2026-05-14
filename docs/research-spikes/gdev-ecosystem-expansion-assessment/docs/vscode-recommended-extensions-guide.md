# Using Recommended Extensions and Settings in VS Code
- **Source**: https://leonardofaria.net/2023/02/10/using-recommended-extensions-and-settings-in-vs-code
- **Retrieved**: 2026-05-14

## Setting Up extensions.json

Create `.vscode/extensions.json` in project root:

```json
{
  "recommendations": [
    "esbenp.prettier-vscode",
    "dbaeumer.vscode-eslint",
    "bradlc.vscode-tailwindcss"
  ]
}
```

When teammates open the workspace, VS Code displays a banner suggesting these extensions. You can add comments to JSON files used by VS Code to document why each extension matters.

## Configuring workspace settings.json

Ties extensions to their configurations:

```json
{
  "editor.defaultFormatter": "esbenp.prettier-vscode",
  "[javascript]": {
    "editor.defaultFormatter": "esbenp.prettier-vscode"
  },
  "editor.formatOnSave": true
}
```

Ensures consistent formatting behavior without requiring individual setup.

## Team Best Practices

Settings follow a precedence hierarchy where "later scopes override earlier scopes." Workspace-level settings sit midway — ideal for team standardization while allowing individual user preferences at higher levels.

By combining extensions.json and settings.json in version control, teams establish consistent development environments across different codebases and technologies.

## Two Distribution Patterns

1. **extensions.json** — Non-mandatory prompts stored in a repository
2. **Extension packs** — Marketplace-published distributions that automatically install dependencies when the pack is installed
