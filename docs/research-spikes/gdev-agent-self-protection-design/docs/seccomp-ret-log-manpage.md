<!-- Source: https://man7.org/linux/man-pages/man2/seccomp.2.html -->
<!-- Retrieved: 2026-05-15 -->

# SECCOMP_RET_LOG Technical Details

## Overview
SECCOMP_RET_LOG is a seccomp filter return action introduced in Linux 4.14 that allows system calls to execute while logging the filter's decision.

## Core Functionality
This action results in "the system call being executed after the filter return action is logged." It occupies a middle position in the precedence hierarchy — higher than SECCOMP_RET_ALLOW but lower than enforcement actions like SECCOMP_RET_TRAP and SECCOMP_RET_KILL_THREAD.

## Logging Mechanism

**SECCOMP_FILTER_FLAG_LOG Flag:**
When this flag is set during filter installation, "all filter return actions except SECCOMP_RET_ALLOW should be logged." An administrator can override this behavior through the sysctl interface.

**actions_logged Sysctl:**
The /proc/sys/kernel/seccomp/actions_logged file provides administrator control over which actions are logged.

## Logging Decision Rules

The kernel applies this hierarchy for determining whether to log actions:

1. SECCOMP_RET_ALLOW actions are never logged
2. Kill actions (SECCOMP_RET_KILL_PROCESS/SECCOMP_RET_KILL_THREAD) are logged if present in actions_logged
3. Filtered actions are logged if the SECCOMP_FILTER_FLAG_LOG flag was set AND the action appears in actions_logged
4. If kernel auditing is enabled and the process is audited, the action is logged
5. Otherwise, no logging occurs

## Key Distinction
Unlike SECCOMP_RET_ALLOW, which silently permits system calls without any logging capability, SECCOMP_RET_LOG explicitly records the filter decision while still allowing execution to proceed.
