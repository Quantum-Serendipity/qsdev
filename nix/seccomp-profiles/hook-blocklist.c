/* seccomp-filter-gen.c -- generates BPF filter for bwrap --seccomp
 *
 * Outputs a binary BPF program to stdout. Redirect to a file at
 * Nix build time, then pass to bwrap via: --seccomp FD  FD< filter.bpf
 *
 * Blocklist approach: allow everything except 43 dangerous syscalls.
 * All blocked syscalls return EPERM (not KILL) for debuggability.
 */

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

    if (seccomp_export_bpf(ctx, STDOUT_FILENO) < 0)
        return 1;

    seccomp_release(ctx);
    return 0;
}
