<!-- Source: https://liambx.com/blog/claude-code-log-analysis-with-duckdb -->
<!-- Retrieved: 2026-03-26 -->

# Analyzing Claude Code Interaction Logs with DuckDB

## Overview

Blog post by Hirotaka Miyagi demonstrating how to leverage DuckDB for analyzing Claude Code's interaction logs in JSONL format to understand AI behavior and optimize development workflows.

## The Challenge

Claude Code Action runs non-interactively on GitHub Actions, requiring careful `allowed_tools` configuration. When permissions are misconfigured, Claude Code attempts workarounds that increase interaction counts and costs. Manual log analysis is inefficient for large datasets.

## Why DuckDB?

"DuckDB can directly query local files (CSV, Parquet, JSON, etc.) using SQL, allowing for interactive analysis without prior data loading." This in-process OLAP database enables interactive exploration without preprocessing, and Claude Code itself can execute DuckDB queries directly.

## Analysis Approach

Two main objectives:
1. **Investigate failures** -- Understand why specific commands executed and failed
2. **Optimize permissions** -- Identify tools Claude Code uses to grant appropriate access proactively

## Key Findings

Through DuckDB analysis of a 837-line conversation log:
- 532 assistant messages and 299 user messages
- 286 messages containing tool usage
- 3 Bash permission errors (primary issue)
- 8 test failures and various technical errors

Analysis revealed specific commands requiring additional permissions: `node:*`, `npx:*`, `cd:*`, `find:*`, and `grep:*`.

## Conclusion

DuckDB enables efficient, autonomous analysis of Claude Code logs, transforming what would be manual, tedious investigation into structured SQL queries that identify permission gaps and optimization opportunities.
