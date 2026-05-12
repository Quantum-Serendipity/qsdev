# Hardening NixOS: A Comprehensive Guide
- **Source**: https://saylesss88.github.io/nix/hardening_NixOS.html
- **Retrieved**: 2026-05-12

## Core Philosophy

The guide emphasizes that "securing your NixOS system begins with a philosophy of minimalism, explicit configuration, and proactive control." It stresses that there are "no plug and play one size fits all security solutions" and users must evaluate each hardening measure against their specific needs.

## Key Attack Vectors & Protections

**Privilege Escalation** remains the primary concern. The guide recommends adopting least-privilege principles, removing unnecessary SUID binaries, and monitoring vulnerability databases.

**Memory Corruption** (Use-After-Free, Double Free) is mitigated through hardened allocators like GrapheneOS's `hardened_malloc`.

## Modern Privilege Escalation: run0 vs. sudo

Rather than traditional SUID binaries, the guide advocates for `run0`, which "asks the service manager to invoke a command or shell under the target user's UID" without inheriting client context.

## Kernel Hardening

### Critical sysctl Settings

```nix
boot.kernel.sysctl = {
  "kernel.kptr_restrict" = 2;
  "kernel.dmesg_restrict" = 1;
  "kernel.unprivileged_bpf_disabled" = 1;
  "kernel.kexec_load_disabled" = 1;
  "vm.unprivileged_userfaultfd" = 0;
};
```

### Boot Parameters

```nix
boot.kernelParams = [
  "slab_nomerge"
  "init_on_alloc=1"
  "init_on_free=1"
  "pti=on"
  "vsyscall=none"
  "module.sig_enforce=1"
  "lockdown=confidentiality"
];
```

## Secrets Management

Use `sops-nix` or `agenix` to keep encrypted secrets under version control rather than plaintext.

## Impermanence Strategy

Root-as-tmpfs setups defeat attacker persistence -- systems get a fresh, secure state on each boot.

## Software Selection Strategy

When choosing packages, verify:
- Maintainer activity (check GitHub last commit dates)
- CVE history and patch status
- Whether to use `nixpkgs-unstable` (faster security patches, less stability) vs. stable channels

## Important Limitations

Neither SELinux nor AppArmor are fully supported in NixOS due to its immutable store structure. For maximum isolation, it recommends dedicated security-focused distributions for critical operations.
