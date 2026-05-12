<!-- Source: https://cookiecutter.readthedocs.io/en/stable/advanced/hooks.html -->
<!-- Retrieved: 2026-05-12 -->

# Cookiecutter Hooks Documentation

## Overview

Cookiecutter hooks are automated scripts that execute at specific points during project generation. They support both Python and shell scripts to handle validation, preprocessing, and post-processing tasks.

## Hook Types and Execution Timing

The framework provides three hook categories:

1. **pre_prompt** - Executes before user prompts appear. Runs from a repository copy without access to template variables. Useful for prerequisite checks like verifying Docker installation.

2. **pre_gen_project** - Executes after user input but before template processing begins. Runs in the generated project root with full template variable access. Ideal for validating user inputs.

3. **post_gen_project** - Executes after project generation completes. Operates in the project root with template variable support. Suitable for conditional cleanup or setup tasks.

## Implementation Structure

Hooks are placed in a `hooks/` directory within your template. The framework supports parallel file naming conventions using `.py` or `.sh` extensions. "Python scripts are recommended for cross-platform compatibility."

## Error Handling

"If a hook exits with a nonzero status, the project generation halts, and the generated directory is cleaned." This ensures failed generations don't leave incomplete projects.

## Key Examples

- **Validation**: Checking module names against regex patterns to ensure valid Python naming conventions
- **Prerequisites**: Verifying required tools are installed before proceeding
- **Conditional Logic**: Using Jinja templating syntax to selectively remove files based on user choices

The "pre_gen_project and post_gen_project hooks support Jinja template rendering, similar to project templates," enabling dynamic behavior based on user selections.
