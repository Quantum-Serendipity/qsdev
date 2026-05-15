# Making a PyPI-Friendly README
- **Source**: https://packaging.python.org/guides/making-a-pypi-friendly-readme/
- **Retrieved**: 2026-05-15

## Supported Formats

PyPI's README renderer accepts three markup languages:

- Plain text
- "reStructuredText (without Sphinx extensions)"
- Markdown (GitHub Flavored Markdown or CommonMark)

## Rendering Limitations

**reStructuredText constraints:** Sphinx extensions like directives and roles (e.g., `:py:func:` or `:ref:`) are prohibited. Invalid markup prevents rendering, forcing PyPI to display raw source instead.

**Markdown note:** Users employing GitHub-flavored Markdown must upgrade their tools to minimum versions: setuptools 38.6.0+, wheel 0.31.0+, and twine 1.11.0+.

## Required Metadata Configuration

In `setup.py`, configure two fields:

1. **`long_description`**: "the contents (not the path) of the README file itself"
2. **`long_description_content_type`**: A Content-Type value matching your markup format (e.g., `text/markdown`, `text/x-rst`, `text/plain`)

## Best Practices for Display Success

The documentation recommends:

- Store README files in your project root alongside `setup.py`
- Use standard naming conventions (`README`, `README.txt`, `README.rst`, or `README.md`)
- Validate reStructuredText with `twine check dist/*` before uploading
- Upload distributions via twine rather than direct PyPI uploads
