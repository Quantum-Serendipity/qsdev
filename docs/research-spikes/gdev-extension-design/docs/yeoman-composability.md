<!-- Source: https://yeoman.io/authoring/composability.html -->
<!-- Retrieved: 2026-05-12 -->

# Yeoman's Composability System

## Core Concept

Yeoman's composability allows generators to work together by combining smaller, reusable parts into larger solutions. As the documentation explains, it's "a way to combine smaller parts to make one large thing."

## Two Composition Approaches

1. **Generator-initiated**: A generator can compose itself with another generator (example: `generator-backbone` uses `generator-mocha`)
2. **End-user initiated**: Currently a planned feature and not yet available

## The `composeWith()` Method

This is the primary API for composition. It accepts two parameters:

- **generatorPath**: Full path to the target generator (typically using `require.resolve()`)
- **options**: An object containing configuration passed to the composed generator

### Usage Examples

For peer dependency generators:
```javascript
this.composeWith(require.resolve('generator-bootstrap/generators/app'), 
  {preprocessor: 'sass'});
```

Using generator namespaces:
```javascript
this.composeWith('backbone:route', {rjs: true});
```

With generator classes:
```javascript
this.composeWith({
  Generator: NodeGenerator,
  path: require.resolve('generator-node/generators/app')
});
```

## Execution Order & Run Loop

Composed generators execute in priority groups. All generators execute functions within a priority group sequentially (in the order `composeWith()` was called), then the process repeats for subsequent groups.

### Execution Example Output

When composing `turbo` and `electric` generators:
```
prompting - turbo
prompting - zap
writing - turbo
writing - zap
```

## Dependency Management

**Recommended approach**: Use `dependencies` for version control. For `peerDependencies`, employ version ranges like `">=1.0.0"` or `"*"` to avoid conflicts.

Note: As of npm@3, peer dependencies require manual installation.
