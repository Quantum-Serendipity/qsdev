# @yawlabs/aws-mcp README
- **Source**: https://raw.githubusercontent.com/YawLabs/aws-mcp/main/README.md
- **Retrieved**: 2026-05-14

## What It Does

MCP server providing AI assistants with comprehensive AWS API access. Positioned as an alternative to AWS's official MCP server, offering a single configuration entry that covers most AWS functionality without requiring Python.

## AWS Services Covered

Generic access to "hundreds of resource types" through Cloud Control API, including Lambda functions, S3 buckets, IAM roles, SSM parameters, and RDS instances. Also covers data-plane operations through direct AWS CLI proxying, CloudWatch Logs access, and IAM permission simulation.

## Authentication Requirements

- Node.js 22+
- AWS CLI v2 installed and accessible via PATH
- AWS profile configured for SSO/IAM Identity Center in `~/.aws/config`
- Environment variables: `AWS_PROFILE` (defaults to "default") and `AWS_REGION`/`AWS_DEFAULT_REGION` (defaults to "us-east-1")

## Key Tools Exposed (24 tools across 5 categories)

**Authentication:** aws_whoami, aws_login_start, aws_login_complete, aws_refresh_if_expiring_soon

**Session Management:** aws_session_set, aws_session_get, aws_session_clear, aws_list_profiles, aws_assume_role

**API Access:** aws_call, aws_paginate, aws_multi_region, aws_logs_tail

**Resource Management:** aws_resource_get, aws_resource_list, aws_resource_create, aws_resource_update, aws_resource_delete, aws_resource_status, aws_resource_diff

**Advanced:** aws_script (JavaScript sandbox), aws_iam_simulate, aws_docs_search, aws_docs_read

## Distinctive Features

- SSO re-authentication via device-code flow (avoids browser handoff failures)
- Generic CRUD operations across hundreds of resources via Cloud Control API
- Dry-run diffing before updates
- Multi-region parallel execution
- JavaScript-based workflow scripting
- Integrated AWS documentation search and retrieval
