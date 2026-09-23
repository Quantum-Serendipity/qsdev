/* ll-restrict.c -- Landlock restriction helper for hook sandboxing
 *
 * Usage: ll-restrict [--verbose] [--ro PATH...] [--rw PATH...] [--deny-net]
 *                    -- CMD [ARGS...]
 *        ll-restrict --version
 *
 * Applies Landlock filesystem (and optionally network) restrictions,
 * then execs CMD. Designed to run INSIDE a bubblewrap namespace as
 * the inner wrapper stage.
 *
 * --version prints "landlock-abi:N", the Landlock ABI the running kernel
 * enforces (0 when Landlock is unsupported or disabled, e.g. missing from
 * the boot lsm= list), and exits 0. qsdev's capability probe relies on it.
 *
 * Nothing is written to stderr on success unless --verbose is given, so
 * the wrapped hook's stderr is not polluted. Every diagnostic starts with
 * "ll-restrict: ".
 *
 * Exit codes are in a range hooks do not normally use, so the caller can
 * tell a helper failure (CMD never ran) from CMD's own exit status. Keep
 * them in sync with internal/sandbox/bwrap/llrestrict.go.
 *   0   - --version only (otherwise exec replaces the process)
 *   120 - Usage error
 *   121 - Landlock unsupported
 *   122 - Landlock setup failed
 *   123 - exec of CMD failed
 */

#define _GNU_SOURCE
#include <errno.h>
#include <fcntl.h>
#include <linux/landlock.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/prctl.h>
#include <sys/syscall.h>
#include <unistd.h>

#ifndef landlock_create_ruleset
static inline int landlock_create_ruleset(
    const struct landlock_ruleset_attr *attr, size_t size, __u32 flags) {
    return (int)syscall(__NR_landlock_create_ruleset, attr, size, flags);
}
#endif

#ifndef landlock_add_rule
static inline int landlock_add_rule(int ruleset_fd,
    enum landlock_rule_type rule_type, const void *rule_attr, __u32 flags) {
    return (int)syscall(__NR_landlock_add_rule, ruleset_fd,
                        rule_type, rule_attr, flags);
}
#endif

#ifndef landlock_restrict_self
static inline int landlock_restrict_self(int ruleset_fd, __u32 flags) {
    return (int)syscall(__NR_landlock_restrict_self, ruleset_fd, flags);
}
#endif

#define ACCESS_FS_V1 ( \
    LANDLOCK_ACCESS_FS_EXECUTE | \
    LANDLOCK_ACCESS_FS_WRITE_FILE | \
    LANDLOCK_ACCESS_FS_READ_FILE | \
    LANDLOCK_ACCESS_FS_READ_DIR | \
    LANDLOCK_ACCESS_FS_REMOVE_DIR | \
    LANDLOCK_ACCESS_FS_REMOVE_FILE | \
    LANDLOCK_ACCESS_FS_MAKE_CHAR | \
    LANDLOCK_ACCESS_FS_MAKE_DIR | \
    LANDLOCK_ACCESS_FS_MAKE_REG | \
    LANDLOCK_ACCESS_FS_MAKE_SOCK | \
    LANDLOCK_ACCESS_FS_MAKE_FIFO | \
    LANDLOCK_ACCESS_FS_MAKE_BLOCK | \
    LANDLOCK_ACCESS_FS_MAKE_SYM)

#define ACCESS_FS_V2 (ACCESS_FS_V1 | LANDLOCK_ACCESS_FS_REFER)
#define ACCESS_FS_V3 (ACCESS_FS_V2 | LANDLOCK_ACCESS_FS_TRUNCATE)

#define ACCESS_READ ( \
    LANDLOCK_ACCESS_FS_EXECUTE | \
    LANDLOCK_ACCESS_FS_READ_FILE | \
    LANDLOCK_ACCESS_FS_READ_DIR)

#define MAX_PATHS 64

#define EXIT_USAGE       120
#define EXIT_UNSUPPORTED 121
#define EXIT_SETUP       122
#define EXIT_EXEC        123

struct path_entry {
    const char *path;
    int writable;
};

int main(int argc, char *argv[]) {
    struct path_entry paths[MAX_PATHS];
    int path_count = 0;
    int deny_net = 0;
    int verbose = 0;
    int cmd_start = -1;

    if (argc == 2 && strcmp(argv[1], "--version") == 0) {
        int v = landlock_create_ruleset(NULL, 0,
                                        LANDLOCK_CREATE_RULESET_VERSION);
        printf("landlock-abi:%d\n", v < 0 ? 0 : v);
        return 0;
    }

    for (int i = 1; i < argc; i++) {
        if (strcmp(argv[i], "--") == 0) {
            cmd_start = i + 1;
            break;
        } else if (strcmp(argv[i], "--ro") == 0 && i + 1 < argc) {
            if (path_count >= MAX_PATHS) {
                fprintf(stderr, "ll-restrict: too many paths\n");
                return EXIT_USAGE;
            }
            paths[path_count++] = (struct path_entry){argv[++i], 0};
        } else if (strcmp(argv[i], "--rw") == 0 && i + 1 < argc) {
            if (path_count >= MAX_PATHS) {
                fprintf(stderr, "ll-restrict: too many paths\n");
                return EXIT_USAGE;
            }
            paths[path_count++] = (struct path_entry){argv[++i], 1};
        } else if (strcmp(argv[i], "--deny-net") == 0) {
            deny_net = 1;
        } else if (strcmp(argv[i], "--verbose") == 0) {
            verbose = 1;
        } else {
            fprintf(stderr, "ll-restrict: unknown option: %s\n", argv[i]);
            return EXIT_USAGE;
        }
    }

    if (cmd_start < 0 || cmd_start >= argc) {
        fprintf(stderr, "ll-restrict: usage: ll-restrict [--verbose] "
                        "[--ro PATH] [--rw PATH] [--deny-net] "
                        "-- CMD [ARGS...] | --version\n");
        return EXIT_USAGE;
    }

    int abi = landlock_create_ruleset(NULL, 0,
                                     LANDLOCK_CREATE_RULESET_VERSION);
    if (abi < 0) {
        if (errno == ENOSYS || errno == EOPNOTSUPP) {
            fprintf(stderr, "ll-restrict: Landlock unavailable "
                            "(kernel too old or disabled)\n");
            return EXIT_UNSUPPORTED;
        }
        perror("ll-restrict: landlock_create_ruleset version check");
        return EXIT_SETUP;
    }
    if (verbose) {
        fprintf(stderr, "ll-restrict: Landlock ABI v%d\n", abi);
    }

    __u64 handled_fs;
    if (abi >= 3)      handled_fs = ACCESS_FS_V3;
    else if (abi >= 2) handled_fs = ACCESS_FS_V2;
    else               handled_fs = ACCESS_FS_V1;

    struct landlock_ruleset_attr ruleset_attr = {
        .handled_access_fs = handled_fs,
        .handled_access_net = 0,
    };

    if (deny_net && abi >= 4) {
        ruleset_attr.handled_access_net =
            LANDLOCK_ACCESS_NET_BIND_TCP |
            LANDLOCK_ACCESS_NET_CONNECT_TCP;
    }

    int ruleset_fd = landlock_create_ruleset(&ruleset_attr,
                                            sizeof(ruleset_attr), 0);
    if (ruleset_fd < 0) {
        perror("ll-restrict: landlock_create_ruleset");
        return EXIT_SETUP;
    }

    for (int i = 0; i < path_count; i++) {
        int parent_fd = open(paths[i].path, O_PATH | O_CLOEXEC);
        if (parent_fd < 0) {
            /* A missing path only narrows access, so it is not an error. */
            if (verbose) {
                fprintf(stderr, "ll-restrict: skipping %s: %s\n",
                        paths[i].path, strerror(errno));
            }
            continue;
        }
        struct landlock_path_beneath_attr path_beneath = {
            .allowed_access = paths[i].writable ? handled_fs : ACCESS_READ,
            .parent_fd = parent_fd,
        };
        int err = landlock_add_rule(ruleset_fd,
                                    LANDLOCK_RULE_PATH_BENEATH,
                                    &path_beneath, 0);
        close(parent_fd);
        if (err) {
            fprintf(stderr, "ll-restrict: rule for %s failed: %s\n",
                    paths[i].path, strerror(errno));
        }
    }

    if (verbose && deny_net && abi < 4) {
        fprintf(stderr, "ll-restrict: ABI < 4, network deny via "
                        "bwrap --unshare-net only\n");
    }

    if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)) {
        perror("ll-restrict: prctl(PR_SET_NO_NEW_PRIVS)");
        close(ruleset_fd);
        return EXIT_SETUP;
    }

    if (landlock_restrict_self(ruleset_fd, 0)) {
        perror("ll-restrict: landlock_restrict_self");
        close(ruleset_fd);
        return EXIT_SETUP;
    }

    close(ruleset_fd);
    execvp(argv[cmd_start], &argv[cmd_start]);
    perror("ll-restrict: execvp");
    return EXIT_EXEC;
}
