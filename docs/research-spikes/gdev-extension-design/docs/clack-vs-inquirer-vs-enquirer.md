<!-- Source: https://www.pkgpulse.com/guides/ink-vs-clack-vs-enquirer-interactive-cli-nodejs-2026 -->
<!-- Retrieved: 2026-05-12 -->

# Comprehensive Comparison: Ink vs @clack/prompts vs Enquirer

## Overview & Download Statistics

**@clack/prompts** dominates with ~500K weekly downloads and represents the modern standard for CLI prompts. **Enquirer** leads in absolute numbers at ~5M weekly downloads as an established solution. **Ink** occupies ~900K weekly downloads as a specialized tool for complex interactive UIs.

## Core Characteristics

### @clack/prompts
- **Bundle Size**: ~2KB (smallest footprint)
- **API Style**: Modern, minimal with built-in cancellation handling
- **TypeScript**: Native support with proper type definitions
- **Module Format**: ESM-native
- **Prompt Types**: 8 core types covering standard interactions

The library features a `group()` API that "runs prompts sequentially, returns all values" with centralized cancellation handling. Cancellation returns `Symbol('cancel')` rather than undefined, forcing explicit handling.

### Enquirer
- **Bundle Size**: ~100KB
- **Prompt Types**: 15+ specialized types including autocomplete, scale/rating, date picker
- **TypeScript**: Community-maintained @types/enquirer
- **Module Format**: CommonJS with ESM interop
- **Customization**: Extensive styling options through global and per-prompt configuration

Enquirer excels with "autocomplete — searchable list" functionality and "scale/rating prompt" capabilities that @clack/prompts lacks.

### Ink
- **Bundle Size**: ~150KB + React dependency
- **Core Model**: React components rendering to terminal
- **TypeScript**: Native, full React ecosystem integration
- **Module Format**: ESM
- **Unique Feature**: Live-updating stateful UIs

Ink enables "complex stateful UIs" through React's state management, making it ideal for dashboards showing "live-updating status across multiple concurrent operations."

## API Pattern Differences

**@clack/prompts** uses a procedural, promise-based approach:
```javascript
const name = await text({ message: "Name?" })
handleCancel(name)
const selected = await select({ message: "Choose?", options: [...] })
```

**Enquirer** follows similar patterns but with more configuration depth:
```javascript
const { result } = await prompt({ type: "autocomplete", ... })
```

**Ink** inverts this through React component composition, maintaining state internally and re-rendering on updates rather than pausing for user input.

## Cancellation Handling

@clack/prompts provides explicit cancellation patterns. The guide notes that "isCancel() checks" for `Symbol('cancel')` and recommends a "handleCancel()" wrapper function after every prompt. The `group()` API centralizes this with an "onCancel" callback.

Enquirer and Ink handle cancellation through standard error handling and component lifecycle methods respectively.

## TypeScript Integration Quality

The guide emphasizes that "@clack/prompts is written in TypeScript and ships its own definitions" with cancellation "represented as `symbol`... which forces developers to handle the cancellation case explicitly."

Enquirer's types use generics: `prompt<{ name: string }>()` provides typed results. Ink offers "the full TypeScript/React type-checking ecosystem" with typed useState hooks.

## When to Use Each

**@clack/prompts**: "standard prompts: text, confirm, select, multiselect, spinner" with "small bundle and ESM are priorities"

**Enquirer**: Projects needing "autocomplete, scale/rating, date picker, or 15+ other prompt types" with "deep customization of prompt appearance"

**Ink**: CLIs displaying "live-updating output (progress bars, real-time logs, dashboards)" with "multiple concurrent async operations with streaming status"

## Testing & Distribution Considerations

@clack/prompts and Enquirer are testable through stdin mocking. Ink benefits from "ink-testing-library" which renders components to an "in-memory frame buffer" without a real terminal.

For bundled executables (pkg, nexe), @clack/prompts creates the leanest binaries. Ink's React dependency "adds several megabytes" to compiled CLIs. For `npx` distribution, @clack/prompts' 2KB size minimizes "cold start time on first run."

## Terminal Compatibility

All three libraries respect `NO_COLOR` environment variables and check `process.stdout.isTTY`. Ink "degrades gracefully in CI environments by detecting `CI=true`" and falls back to plain text output for GitHub Actions and Docker contexts.
