<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/user-guides/workspace-lifecycle.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Workspace Lifecycle: Comprehensive Overview

## Core Definition
Workspaces are described as "flexible, reproducible, and isolated units of compute" that progress through creation, management, operation, and deletion phases.

## Workspace States

**Primary States:**
- **Running**: Workspace has started and is ready to accept connections
- **Stopped**: Ephemeral resources are destroyed; persistent resources remain idle
- **Deleted**: All resources destroyed; workspace records removed from database

**Broken States:**
- **Failed**: Provisioning errors occur; no resources consume compute
- **Unhealthy**: Resources provisioned but agent cannot facilitate connections

## Resource Persistence Model

Workspaces contain two resource types:
- **Ephemeral resources**: Destroyed and recreated when workspace restarts
- **Persistent resources**: Remain provisioned during stopped state; destroyed only upon deletion

## Creation and Provisioning Process

When users create a workspace, they initiate a build request to the control plane. Coder then:

1. Uses Terraform to provision resources defined by a template
2. Distinguishes between computational resources (running the agent) and peripheral resources
3. Starts the agent process on computational resources
4. Agent opens connections via SSH, terminal, and IDEs (JetBrains, VSCode)
5. Agent executes startup scripts for configuration and personalization
6. Workspace transitions to Running state

## State Transitions

**Starting a Workspace:**
- Manual activation via dashboard, CLI, or API
- Automatic start enabled by user preference

**Stopping a Workspace:**
- Manual cessation by users/admins
- Automatic stopping via template updates or inactivity scheduling
- Can resume by manual start or automatic activation on connection

**Deleting a Workspace:**
- Manual deletion by users/admins
- Automatic deletion via dormancy policies (enterprise)
- Executes `terraform destroy` irreversibly
- Can use `--orphan` flag to delete workspace without removing resources (Template Admin role only)

## Quota and Cost Control

"By default, there is no limit on the number of workspaces a user may create," but enterprise administrators can implement quotas per template, group, or organization to prevent over-provisioning.

Automatic scheduling controls costs through inactivity-based stopping and dormancy-triggered deletion.

## Error Diagnosis

**Failed workspaces** typically result from misalignment between template definitions and actual infrastructure resources. **Unhealthy workspaces** usually stem from agent misconfiguration or startup script errors.

## Build Timing Visibility

Starting in v2.17, the dashboard displays workspace build timing breakdowns, including provisioning duration and agent startup steps (encompassing dotfiles and application startups).
