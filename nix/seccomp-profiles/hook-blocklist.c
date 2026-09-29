/* seccomp-filter-gen.c -- generates BPF filter for bwrap --seccomp
 *
 * Outputs a binary BPF program to stdout. Redirect to a file at
 * Nix build time, then pass to bwrap via: --seccomp FD  FD< filter.bpf
 *
 * Blocklist approach: allow everything except 43 dangerous syscalls.
 * All blocked syscalls return EPERM (not KILL) for debuggability.
 *
 * Namespace creation is also blocked through clone(2): unshare and setns
 * alone leave clone(CLONE_NEWUSER|...) open. clone3(2) passes its flags in a
 * struct that seccomp cannot inspect, so it returns ENOSYS and libc falls back
 * to clone(2), where the flag filter applies. (bwrap --disable-userns closes
 * nested user namespaces at the kernel level as well.)
 */

#define _GNU_SOURCE
#include <errno.h>
#include <sched.h>
#include <seccomp.h>
#include <stdio.h>
#include <unistd.h>

static const int blocked[] = {
    /* Category 1: Kernel module loading */
    SCMP_SYS(init_module), SCMP_SYS(finit_module), SCMP_SYS(delete_module),
    /* Category 2: Kernel/system control */
    SCMP_SYS(kexec_load), SCMP_SYS(kexec_file_load),
    SCMP_SYS(reboot), SCMP_SYS(acct),
    /* Category 3: Mount manipulation (includes new mount API) */
    SCMP_SYS(mount), SCMP_SYS(umount2), SCMP_SYS(pivot_root),
    SCMP_SYS(swapon), SCMP_SYS(swapoff),
    SCMP_SYS(open_tree), SCMP_SYS(move_mount),
    SCMP_SYS(fsopen), SCMP_SYS(fsconfig), SCMP_SYS(fsmount),
    SCMP_SYS(fspick), SCMP_SYS(mount_setattr),
    /* Category 4: Process introspection */
    SCMP_SYS(ptrace), SCMP_SYS(process_vm_readv),
    SCMP_SYS(process_vm_writev), SCMP_SYS(kcmp),
    /* Category 5: Kernel exploit primitives */
    SCMP_SYS(userfaultfd), SCMP_SYS(bpf), SCMP_SYS(perf_event_open),
    /* Category 6: io_uring (bypasses seccomp on I/O operations) */
    SCMP_SYS(io_uring_setup), SCMP_SYS(io_uring_enter),
    SCMP_SYS(io_uring_register),
    /* Category 7: Namespace escape */
    SCMP_SYS(open_by_handle_at), SCMP_SYS(unshare), SCMP_SYS(setns),
    /* Category 8: Time manipulation */
    SCMP_SYS(settimeofday), SCMP_SYS(clock_settime), SCMP_SYS(adjtimex),
    /* Category 9: Kernel keyring */
    SCMP_SYS(add_key), SCMP_SYS(keyctl), SCMP_SYS(request_key),
    /* Category 10: Information leak */
    SCMP_SYS(quotactl), SCMP_SYS(lookup_dcookie),
};

#ifdef __x86_64__
static const int blocked_arch[] = {
    SCMP_SYS(ioperm), SCMP_SYS(iopl), SCMP_SYS(modify_ldt),
};
#else
static const int blocked_arch[] = {};
#endif

/* clone(2) flags that create a new namespace. Each gets its own rule because a
 * masked-equality compare matches one bit; rules for the same syscall and
 * action are OR'ed, so any namespace bit denies the call. */
static const unsigned long clone_ns_flags[] = {
    CLONE_NEWNS, CLONE_NEWCGROUP, CLONE_NEWUTS, CLONE_NEWIPC,
    CLONE_NEWUSER, CLONE_NEWPID, CLONE_NEWNET,
};

/* On s390/s390x the clone flags are the SECOND argument. */
#if defined(__s390__) || defined(__s390x__)
#define CLONE_FLAGS_ARG(op, mask, datum) SCMP_A1(op, mask, datum)
#else
#define CLONE_FLAGS_ARG(op, mask, datum) SCMP_A0(op, mask, datum)
#endif

int main(void) {
    scmp_filter_ctx ctx = seccomp_init(SCMP_ACT_ALLOW);
    if (!ctx) return 1;

    for (size_t i = 0; i < sizeof(blocked)/sizeof(blocked[0]); i++) {
        if (seccomp_rule_add(ctx, SCMP_ACT_ERRNO(1), blocked[i], 0) < 0)
            return 1;
    }
    for (size_t i = 0; i < sizeof(blocked_arch)/sizeof(blocked_arch[0]); i++) {
        if (seccomp_rule_add(ctx, SCMP_ACT_ERRNO(1), blocked_arch[i], 0) < 0)
            return 1;
    }
    for (size_t i = 0; i < sizeof(clone_ns_flags)/sizeof(clone_ns_flags[0]); i++) {
        if (seccomp_rule_add(ctx, SCMP_ACT_ERRNO(EPERM), SCMP_SYS(clone), 1,
                             CLONE_FLAGS_ARG(SCMP_CMP_MASKED_EQ,
                                             clone_ns_flags[i], clone_ns_flags[i])) < 0)
            return 1;
    }
    if (seccomp_rule_add(ctx, SCMP_ACT_ERRNO(ENOSYS), SCMP_SYS(clone3), 0) < 0)
        return 1;

    if (seccomp_export_bpf(ctx, STDOUT_FILENO) < 0)
        return 1;

    seccomp_release(ctx);
    return 0;
}
