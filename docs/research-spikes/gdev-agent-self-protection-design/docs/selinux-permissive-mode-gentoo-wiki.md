<!-- Source: https://wiki.gentoo.org/wiki/SELinux/Tutorials/Permissive_versus_enforcing -->
<!-- Retrieved: 2026-05-15 -->

# SELinux Permissive Mode: Technical Details (Gentoo Wiki)

## Global Permissive Mode

**Definition**: "SELinux prints warnings instead of enforcing" when operating in permissive mode. The system logs potential policy violations without blocking them.

**Setting Methods**:

1. **setenforce command** (temporary):
```
root # setenforce 0  # Switch to permissive
root # setenforce 1  # Switch to enforcing
```

2. **Configuration file** (/etc/selinux/config):
```
SELINUX=permissive
```

3. **Kernel boot parameter** (overrides config file):
```
enforcing=0
```

## Per-Domain Permissive Mode

**Purpose**: Mark specific SELinux domains as permissive while keeping the rest of the system in enforcing mode.

**Setting via semanage**:
```
root # semanage permissive -a xbmc_t
```

**Viewing permissive domains**:
```
root # semanage permissive -l
```

## Key Technical Differences

**Permissive vs. Disabled**:
- Permissive mode: "still logs what it would have denied"
- Disabled: stops generating SELinux contexts for new/modified files, requiring full filesystem relabeling upon re-enabling

**SELinux-Aware Applications**:
Applications linked with libselinux.so may behave differently in permissive mode versus completely disabled mode, potentially still encountering configuration-related failures.

## Transition Workflow

When switching from permissive to enforcing after system readiness:
```
root # setenforce 1
```

**Critical consequence of disabling SELinux**: Files created while disabled lose context assignments, necessitating system relabeling in permissive mode before returning to enforcing operation.
