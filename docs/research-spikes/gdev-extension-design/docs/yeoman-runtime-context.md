<!-- Source: https://yeoman.io/authoring/running-context -->
<!-- Retrieved: 2026-05-12 -->

# Yeoman Generator Runtime Context

## Core Concept
Yeoman organizes generator execution through a **run loop**—a queue system with priority support that ensures methods execute in a predictable sequence. This enables proper composition of multiple generators.

## Method Classification

**Prototype Methods as Tasks**: "Each method directly attached to a Generator prototype is considered to be a task. Each task is run in sequence by the Yeoman environment run loop."

**Private Methods**: Three approaches prevent automatic execution:
- Prefix with underscore (`_methodName`)
- Define as instance methods in the constructor
- Place on parent generator classes

## Priority Queue System

The run loop executes methods matching these priority names in order:

1. **initializing** - State checks and configuration retrieval
2. **prompting** - User input collection via `this.prompt()`
3. **configuring** - Metadata file creation and project setup
4. **default** - Unnamed methods fall here
5. **writing** - Generator-specific file output
6. **conflicts** - Internal conflict handling
7. **install** - Dependency installation (npm, bower)
8. **end** - Cleanup and completion tasks

## Asynchronous Support

Two async patterns pause execution:
- **Promise-based**: Return a promise; the loop continues upon resolution
- **Legacy callback**: Call `this.async()` to obtain a completion function

This architecture enables "generators will play nice with others" through standardized execution ordering.
