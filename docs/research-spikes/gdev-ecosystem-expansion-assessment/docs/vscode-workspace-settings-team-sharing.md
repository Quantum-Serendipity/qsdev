# VS Code Workspace Settings: Team Sharing Guide
- **Source**: https://spin.atomicobject.com/vscode-workspace-settings/
- **Retrieved**: 2026-05-14

## Why Share Settings

"Setting norms for developer tasks on a team project is important for maintaining your code's consistency and readability." Visual Studio Code supports both individual user settings and shared workspace configurations, allowing teams to enforce standards without overriding personal preferences.

## Files to Share

Teams should create a `.vscode` folder in their project root containing two files:

1. **settings.json** — Contains workspace-specific configurations like linting and formatting rules
2. **extensions.json** — Lists recommended plugins for the project

Example: enforcing ESLint and Prettier for formatting consistency across a Laravel/React project.

## Benefits

Primary advantage is efficiency: "Adding a workspace folder to your Visual Studio Code project can ensure that your team wastes less time resolving linting and formatting issues due to mismatched settings." This automation prevents repeated pull request feedback about formatting inconsistencies.

## Implementation Concerns

The article recommends ensuring the `.vscode` folder is tracked in version control by checking that your `.gitignore` file doesn't exclude it. This step is crucial for the shared settings to reach all team members.

## Key Takeaway

By automating formatting and linting rules through shared workspace configuration, development teams can focus on substantive code review rather than style corrections.
