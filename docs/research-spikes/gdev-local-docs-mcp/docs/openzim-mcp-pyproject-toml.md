<!-- Source: https://raw.githubusercontent.com/cameronrye/openzim-mcp/main/pyproject.toml -->
<!-- Retrieved: 2026-05-14 -->

# OpenZIM-MCP pyproject.toml

## Project Metadata
- **Name:** openzim-mcp
- **Version:** 2.0.0a12
- **Python Requirement:** >=3.12
- **License:** MIT
- **Maintainer:** Cameron Rye

## Core Dependencies

| Package | Version Constraint |
|---------|-------------------|
| beautifulsoup4 | >=4.14.3, <5.0 |
| html2text | >=2025.4.15, <2027.0 |
| libzim | >=3.9.0, <4.0 |
| mcp[cli] | >=1.27.0, <2.0 |
| pydantic | >=2.13.3, <3.0 |
| pydantic-settings | >=2.14.0, <3.0 |
| tiktoken | >=0.7.0, <1.0 |

## Entry Point
`openzim-mcp` command calls `openzim_mcp.__main__:main`

## Key Notes
- Requires Python >=3.12
- libzim dependency bundles native C++ libzim in PyPI wheels
- Uses MCP SDK with CLI extras
- tiktoken for token counting
- beautifulsoup4 + html2text for HTML-to-text conversion
