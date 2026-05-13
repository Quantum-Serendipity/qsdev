<!-- Source: https://eslint.org/docs/latest/extend/shareable-configs -->
<!-- Retrieved: 2026-05-12 -->

# ESLint Shareable Configs: Complete Guide

## Package Naming Conventions

ESLint recommends two naming patterns for shareable configs:

1. **Standard packages**: Begin with `eslint-config-`, such as `eslint-config-myconfig`
2. **Scoped packages**: Use `@scope/eslint-config` or `@scope/eslint-config-myconfig` format

These conventions make packages easily discoverable on npm.

## Creating a Shareable Config

A shareable config is an npm package that exports configuration objects. The default export comes from the `main` entry point (typically `index.js`). Here's the basic structure:

```javascript
// index.js
export default [
	{
		languageOptions: {
			globals: {
				MyGlobal: true,
			},
		},
		rules: {
			semi: [2, "always"],
		},
	},
];
```

The documentation emphasizes that "you can read these settings from a file or generate them dynamically" since `index.js` is standard JavaScript.

## Publishing to npm

When publishing, include these in your `package.json`:

- **Keywords**: Use `eslint` and `eslintconfig` for discoverability
- **Peer Dependencies**: Declare ESLint compatibility using `">= 9"` syntax for forward compatibility
- **Regular Dependencies**: List any required plugins or parsers as `dependencies`

## Consuming Shareable Configs

Import and use the config in `eslint.config.js`:

```javascript
import myconfig from "eslint-config-myconfig";

export default defineConfig([
	{
		files: ["**/*.js"],
		extends: [myconfig],
	},
]);
```

Note: The CLI `--config` flag cannot reference shareable configs.

## Overriding Settings

Settings placed after the `extends` property override the shared config:

```javascript
export default defineConfig([
	{
		files: ["**/*.js"],
		extends: [myconfig],
		rules: {
			"no-unused-vars": "warn",
		},
	},
]);
```

## Sharing Multiple Configs

Export additional configs as separate files in your package:

```javascript
// my-special-config.js
export default {
	rules: {
		quotes: [2, "double"],
	},
};
```

Access them via subpath imports:

```javascript
import mySpecialConfig from "eslint-config-myconfig/my-special-config.js";
```

The documentation strongly recommends "always including a default export for your package to avoid confusion."
