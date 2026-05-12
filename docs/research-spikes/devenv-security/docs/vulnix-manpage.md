<!-- Source: https://github.com/nix-community/vulnix/blob/master/doc/vulnix.1.md -->
<!-- Retrieved: 2026-05-12 -->

# Vulnix Manpage (vulnix.1)

## Invocation Modes
- `vulnix --system` — analyzes the current system
- `vulnix PATH ...` — scans specified derivations or store outputs
- Multiple scan targets can be combined

## Key Command-Line Options

**Scan targets:**
- `-S, --system` — current system configuration
- `-G, --gc-roots` — all garbage collection roots
- `-p, --profile=PATH` — nix profiles (from `nix-env` or `nix profile`)
- `-f, --from-file=PATH` — derivations listed in a file

**Vulnerability management:**
- `-w, --whitelist=PATH|URL` — load exemption rules (repeatable)
- `-W, --write-whitelist=PATH` — generate updated whitelist with new findings
- `-s, --show-whitelisted` — display masked vulnerabilities alongside results

**Database and output:**
- `-m, --mirror=URL` — custom NVD source (defaults to official NIST feed)
- `-c, --cache-dir=PATH` — override cache location (~/.cache/vulnix)
- `-j, --json` — structured JSON output format
- `-v, --verbose` — additional diagnostic information (stackable)

**Requisites control:**
- `-r, --requisites` — include transitive dependencies (default)
- `-R, --no-requisites` — scan only specified paths

## Exit Codes

Compatible with Nagios monitoring standards:
- `0` — no vulnerabilities detected
- `1` — all findings whitelisted (only with `--show-whitelisted`)
- `2` — active vulnerabilities present
- `3` — error condition

## Patch Detection

The scanner automatically excludes CVEs with corresponding patches. Derivations should name patches with CVE identifiers: `CVE-2018-9055.patch` or multi-vulnerability patches like `CVE-2018-9055+CVE-2018-9600.patch`.

## JSON Output Structure

Each vulnerable derivation includes:
- `name`, `pname`, `version`
- `affected_by` — applicable CVE list
- `whitelisted` — masked identifiers
- `derivation` — .drv file path
- `cvssv3_basescore` — severity scores per CVE
