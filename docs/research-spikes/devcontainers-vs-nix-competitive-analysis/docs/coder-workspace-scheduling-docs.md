<!-- Source: https://raw.githubusercontent.com/coder/coder/main/docs/user-guides/workspace-scheduling.md -->
<!-- Retrieved: 2026-03-20 -->

# Coder Workspace Scheduling: Complete Overview

## Core Scheduling Features

**Autostart** enables workspaces to launch automatically at specified times on selected days. Administrators must enable this capability in template settings, and they can restrict which days trigger autostart. Users select their preferred timezone.

**Autostop** halts workspaces after a designated duration in hours. The system respects user activity — it won't shutdown if someone actively uses the workspace and waits for inactivity periods before enforcing termination. "Autostop must be enabled on the template prior to workspace creation, it is not applied to existing running workspaces."

## Activity Detection Mechanisms

Coder identifies workspace activity through specific session types:
- VSCode/code-server sessions
- JetBrains IDE connections
- Web terminal usage
- SSH connections
- AI agent task reporting showing "working" status

Importantly, "Viewing workspace details in the dashboard" and similar administrative actions don't constitute activity. Users should close all connections when finished to prevent unexpected expenses.

## Template Administrator Controls

Template admins configure the **activity bump duration**, determining how long to postpone shutdown after inactivity detection (defaulting to 1 hour). They enable/disable autostart and autostop features at the template level.

## Premium Features

**Autostop Requirement** (Premium) allows admins to enforce mandatory workspace stops for updates, overriding active connections. Frequency is specified in days or weeks, applied in user timezone.

**User Quiet Hours** (Premium) enables users to designate timeframes when forced shutdowns occur during autostop requirement enforcement.

**Dormancy** (Premium) automatically deletes inactive workspaces after configurable thresholds, with separate settings for deletion timing in dormant states.
