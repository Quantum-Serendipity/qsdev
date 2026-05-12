<!-- Source: https://nextjs.org/docs/app/api-reference/cli/create-next-app -->
<!-- Retrieved: 2026-05-12 -->

# create-next-app CLI Documentation

The `create-next-app` CLI allows you to create a new Next.js application using the default template or an example from a public GitHub repository.

## Basic Usage

```bash
npx create-next-app@latest [project-name] [options]
pnpm create next-app [project-name] [options]
yarn create next-app [project-name] [options]
bun create next-app [project-name] [options]
```

## CLI Options

| Options | Description |
|---|---|
| `--no-*` | Negate default options. E.g. `--no-ts` |
| `--ts` / `--typescript` | Initialize as TypeScript project (default) |
| `--js` / `--javascript` | Initialize as JavaScript project |
| `--tailwind` | Initialize with Tailwind CSS config (default) |
| `--react-compiler` | Initialize with React Compiler enabled |
| `--eslint` | Initialize with ESLint config |
| `--biome` | Initialize with Biome config |
| `--no-linter` | Skip linter configuration |
| `--app` | Initialize as App Router project |
| `--api` | Initialize with only route handlers |
| `--src-dir` | Initialize inside a `src/` directory |
| `--turbopack` | Force enable Turbopack (enabled by default) |
| `--webpack` | Force enable Webpack |
| `--import-alias` | Specify import alias (default "@/*") |
| `--empty` | Initialize an empty project |
| `--use-npm/pnpm/yarn/bun` | Specify package manager |
| `--example [name] [url]` | Bootstrap from an example |
| `--reset-preferences` | Reset stored preferences |
| `--skip-install` | Skip installing packages |
| `--disable-git` | Disable git initialization |
| `--agents-md` | Include AGENTS.md and CLAUDE.md (default) |
| `--yes` | Use previous preferences or defaults |

## Interactive Prompts

On installation, you see:

```
What is your project named? my-app
Would you like to use the recommended Next.js defaults?
    Yes, use recommended defaults - TypeScript, ESLint, Tailwind CSS, App Router, AGENTS.md
    No, reuse previous settings
    No, customize settings - Choose your own preferences
```

If you choose to customize settings:

```
Would you like to use TypeScript? No / Yes
Which linter would you like to use? ESLint / Biome / None
Would you like to use React Compiler? No / Yes
Would you like to use Tailwind CSS? No / Yes
Would you like your code inside a `src/` directory? No / Yes
Would you like to use App Router? (recommended) No / Yes
Would you like to customize the import alias (`@/*` by default)? No / Yes
What import alias would you like configured? @/*
Would you like to include AGENTS.md to guide coding agents? No / Yes
```

## Key UX Patterns

1. **Recommended defaults as first option** — "Yes, use recommended defaults" is the first choice
2. **Preference persistence** — "No, reuse previous settings" remembers past choices
3. **Progressive disclosure** — Only shows detailed questions if you choose "customize settings"
4. **CLI flags for non-interactive use** — Every prompt has a corresponding flag
5. **Negation pattern** — `--no-*` prefix to negate any default
