# Apache License 2.0 Compatibility for Addon/Plugin Authors

## Context

This report analyzes license compatibility for Go addon packages that import an Apache-2.0 licensed framework ("gdev") as a library dependency. It covers what licenses addon authors can use, how Go's compilation model affects the analysis, and practical compliance requirements.

---

## 1. Apache-2.0 Compatibility Matrix

### Understanding the Permissive Nature

Apache 2.0 is a **permissive** license. Unlike copyleft licenses (GPL family), it does not require derivative works to use the same license. The Apache License 2.0, Section 1, explicitly defines "Derivative Works" to **exclude** "works that remain separable from, or merely link (or bind by name) to the interfaces of, the Work." This is a critical clause for addon authors: code that *uses* an Apache-2.0 library through its public API is generally not considered a derivative work of that library under the Apache License itself.

This means **the Apache License imposes almost no constraints on what license addon code can use.** The constraints flow from the *addon's* chosen license, not from Apache 2.0.

### Compatibility by License Family

#### Permissive Licenses: All Compatible

| License | Compatible? | Direction | Notes |
|---------|------------|-----------|-------|
| **Apache-2.0** | Yes | Bidirectional | Simplest choice; same patent protections |
| **MIT** | Yes | Bidirectional | Simpler terms, no patent grant |
| **BSD-2-Clause** | Yes | Bidirectional | Minimal requirements |
| **BSD-3-Clause** | Yes | Bidirectional | No endorsement clause |
| **ISC** | Yes | Bidirectional | Functionally equivalent to MIT |

All permissive licenses combine freely with Apache 2.0. The ASF classifies these as "Category A" -- permitted without restriction in Apache projects. Google classifies them all as "Notice" licenses with identical compliance requirements (include copyright notices).

**Sources**: [ASF Third-Party License Policy](https://www.apache.org/legal/resolved.html), [Google License Classification](https://opensource.google/documentation/reference/thirdparty/licenses), [LicenseCheck.io Guide](https://licensecheck.io/guides/apache-compatible)

#### GPL Family: Complex and Mostly Problematic

| License | Compatible? | Direction | Notes |
|---------|------------|-----------|-------|
| **GPL-2.0-only** | **No** | Incompatible both ways | Patent termination clause conflict |
| **GPL-3.0** | **One-way** | Apache-2.0 code CAN enter GPL-3.0 works; GPL-3.0 code CANNOT enter Apache works | The combined work must be GPL-3.0 |
| **LGPL-2.1 / LGPL-3.0** | **Conditional** | Dynamic linking only | Static linking (Go's model) creates incompatibility |
| **AGPL-3.0** | **No** | Effectively prohibited | Network copyleft; Google bans it entirely |

**The GPL-2.0 incompatibility** is the most commonly encountered issue. The FSF considers Apache 2.0 incompatible with GPLv2 because Apache 2.0's patent termination and indemnification provisions constitute "additional restrictions" not present in GPLv2. Both the ASF and FSF agree on this incompatibility.

**The GPL-3.0 one-way rule** means: if an addon is GPL-3.0, it can import Apache-2.0 gdev, but the resulting combined binary must be distributed under GPL-3.0 terms. This is legally valid but creates a "viral" effect -- anyone distributing the combined binary must comply with GPL-3.0 for the whole work. The reverse is not true: an Apache-2.0 project cannot incorporate GPL-3.0 code.

**AGPL-3.0** extends GPL-3.0 with network use provisions. Google's policy: "Cannot be used in google3 under any circumstances." The ASF lists it as Category X (prohibited).

**Sources**: [ASF GPL Compatibility](https://www.apache.org/licenses/GPL-compatibility.html), [FSF License Compatibility](https://www.gnu.org/licenses/license-compatibility.en.html), [The Hyve License Combining](https://www.thehyve.nl/articles/open-source-software-licenses-part-3)

#### Weak Copyleft: Conditionally Compatible

| License | Compatible? | Notes |
|---------|------------|-------|
| **MPL-2.0** | **Yes** | File-level copyleft; modified MPL files must stay MPL, but other files are free |
| **EPL-2.0** | **Conditional** | Requires secondary license designation |
| **CDDL-1.0** | **Conditional** | File-level copyleft |

**MPL-2.0 is the standout here.** It permits static linking (unlike LGPL), maintains copyleft at the file level only, and is well-established. The ASF classifies MPL-2.0 as Category B (allowed in binary form with appropriate labeling). Google classifies it as "Reciprocal" (must make component source available, but no taint of surrounding code).

**Sources**: [ASF Third-Party Policy](https://www.apache.org/legal/resolved.html), [Google License Classification](https://opensource.google/documentation/reference/thirdparty/licenses)

#### Proprietary / Closed Source: Fully Compatible

| License | Compatible? | Notes |
|---------|------------|-------|
| **Proprietary** | **Yes** | Must satisfy Apache 2.0 redistribution requirements for the framework code |

Apache 2.0 explicitly permits incorporation into proprietary software. You do NOT need to open-source your addon code. See Section 4 for detailed requirements.

**Sources**: [Snyk Apache License Guide](https://snyk.io/articles/apache-license/), [FOSSA Apache 2.0 Guide](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/)

### Summary: What Can Addon Authors Choose?

For a Go addon that **imports** an Apache-2.0 framework:

| License Choice | Viable? | Practical Rating |
|---------------|---------|-----------------|
| Apache-2.0 | Yes | **Recommended** -- same ecosystem, patent protections |
| MIT | Yes | Good -- simpler but no patent grant |
| BSD-2/3-Clause | Yes | Good -- minimal obligations |
| ISC | Yes | Good -- MIT-equivalent |
| MPL-2.0 | Yes | Good -- if weak copyleft desired |
| GPL-3.0 | Yes, but... | Caution -- entire binary becomes GPL-3.0 |
| GPL-2.0 | **No** | Incompatible with Apache-2.0 |
| LGPL | **No** (in Go) | Static linking makes this unworkable |
| AGPL-3.0 | Technically yes, but... | **Avoid** -- toxic to downstream adoption |
| Proprietary | Yes | Must include Apache notices for gdev |

---

## 2. The "Linking" Question for Go

### Go's Compilation Model

Go compiles all dependencies -- including imported libraries -- into a **single statically-linked binary**. There is no dynamic linking in the conventional sense (Go's `plugin` package exists but is limited to Linux/macOS and rarely used). Every `import` in Go source code causes the imported package's compiled code to be physically embedded in the output binary.

### Why This Matters for Copyleft Licenses

The FSF's position (from the GPL FAQ) is clear: **the GPL makes no distinction between static and dynamic linking.** Both create a "combined work" subject to GPL terms. However, the LGPL *does* make this distinction -- LGPL requires that users be able to re-link the application with a modified version of the library. This requirement is satisfiable with dynamic linking (swap the .so file) but **nearly impossible with Go's static compilation model**.

### Practical Implications for Each License

**GPL-3.0 addon importing Apache-2.0 gdev**: Legal. The combined binary must be distributed under GPL-3.0. The Apache 2.0 license permits this (it is a permissive license). Users who receive the binary get GPL-3.0 rights to the whole thing.

**Apache-2.0 addon importing GPL-3.0 dependency**: Illegal. You cannot include GPL-3.0 code in a work distributed under Apache-2.0 terms. The GPL-3.0 would require the whole binary to be GPL-3.0, which conflicts with your Apache-2.0 licensing.

**LGPL addon or dependency in Go**: Effectively non-viable. The LGPL's re-linking requirement cannot be satisfied because Go produces monolithic binaries. As the Go community has widely noted: "For Go specifically, LGPL requirements to 'provide complete object files to recipients' make it functionally equivalent to GPL due to static linking." The recommended alternative for weak copyleft in Go is **MPL-2.0**, which operates at the file level and permits static linking.

**MPL-2.0 addon importing Apache-2.0 gdev**: Fully workable. MPL-2.0's copyleft applies only to modified MPL-2.0 files. New files in the addon can be any license. The Apache-2.0 gdev code is unaffected by the addon's MPL-2.0 choice.

### Go Ecosystem License Statistics

87% of Go code on GitHub uses permissive licenses (MIT, Apache-2.0, BSD). This overwhelming preference for permissive licensing is partly driven by Go's static linking model making copyleft licenses impractical.

**Sources**: [makeworld.space LGPL Go Analysis](https://www.makeworld.space/2021/01/lgpl_go.html), [Medium: Open Source and Go](https://medium.com/@henvic/opensource-and-go-what-license-f6b36c201854), [golang-nuts discussion](https://groups.google.com/g/golang-nuts/c/JqOAWBpL-70)

---

## 3. Real-World Examples

### Kubernetes Ecosystem (Framework: Apache-2.0)

Kubernetes is Apache-2.0 licensed. The vast majority of the K8s operator/controller ecosystem also uses Apache-2.0:

| Project | Role | License |
|---------|------|---------|
| Kubernetes | Framework | Apache-2.0 |
| cert-manager | Certificate operator | Apache-2.0 |
| prometheus-operator | Monitoring operator | Apache-2.0 |
| Argo CD | GitOps controller | Apache-2.0 |
| Flux CD | GitOps controller | Apache-2.0 |
| Operator SDK | Operator framework | Apache-2.0 |
| Crossplane | Infrastructure operator | Apache-2.0 |
| external-secrets | Secrets operator | Apache-2.0 |
| Istio | Service mesh | Apache-2.0 |
| Cilium | CNI/networking | Apache-2.0 |
| Kong Ingress Controller | Ingress | Apache-2.0 |
| containerd | Container runtime | Apache-2.0 |
| etcd | Distributed KV store | Apache-2.0 |
| OpenTelemetry Collector | Observability | Apache-2.0 |

**Pattern**: The Kubernetes ecosystem overwhelmingly standardizes on Apache-2.0 for addons/operators. This creates a homogeneous licensing environment that simplifies compliance.

### Terraform Provider Ecosystem (Framework: Was MPL-2.0, Now BUSL-1.1)

Terraform providers show a more varied pattern:

| Project | Role | License |
|---------|------|---------|
| Terraform (core) | Framework | BUSL-1.1 (was MPL-2.0) |
| terraform-provider-aws | AWS provider | MPL-2.0 |
| terraform-provider-google | GCP provider | MPL-2.0 |
| terraform-provider-kubernetes | K8s provider | MPL-2.0 |
| terraform-provider-github | GitHub provider | MIT |
| terraform-provider-cloudflare | Cloudflare provider | Apache-2.0 |
| terraform-provider-digitalocean | DO provider | MPL-2.0 |

**Pattern**: HashiCorp's own providers use MPL-2.0 (matching the original framework license). Third-party providers choose their own license -- MIT, Apache-2.0, or MPL-2.0 are all common. This demonstrates that addon authors are free to pick their own license when the framework is permissively licensed.

### Grafana Plugin Ecosystem (Framework: AGPL-3.0)

| Project | Role | License |
|---------|------|---------|
| Grafana | Framework | AGPL-3.0 |
| grafana-plugin-sdk-go | Plugin SDK | Apache-2.0 |
| clock-panel | Plugin | MIT |
| piechart-panel | Plugin | MIT |

**Pattern**: Grafana itself is AGPL-3.0, but its **plugin SDK is Apache-2.0**. This is a deliberate architectural choice -- by licensing the SDK permissively, Grafana allows plugins to use any license (including proprietary). The plugins communicate with Grafana via gRPC (process boundary), avoiding the AGPL's linking implications. Grafana community plugins commonly use MIT or Apache-2.0.

### Other Go Projects

| Project | Role | License |
|---------|------|---------|
| Traefik | Reverse proxy | MIT |
| Caddy | Web server (with plugin system) | Apache-2.0 |

Caddy's plugin ecosystem is notable: the framework is Apache-2.0 and plugins are compiled in at build time (static linking). Plugins can be any compatible license.

**Sources**: License data verified via `gh api repos/<owner>/<repo> --jq '.license.spdx_id'` on 2026-05-13.

---

## 4. Apache-2.0 and Proprietary Addon Code

### Yes, You Can Keep Addons Proprietary

Apache 2.0 is a permissive license that explicitly permits incorporation into proprietary, closed-source products. You are NOT required to distribute your source code. This is the fundamental difference between Apache 2.0 and copyleft licenses like the GPL.

### Specific Requirements When Distributing a Binary Containing Apache-2.0 Code

When you distribute a binary that includes Apache-2.0 licensed gdev code, Section 4 of the Apache License requires:

1. **Include a copy of the Apache License 2.0** -- provide the license text alongside your distribution
2. **Mark modified files** -- if you modified any gdev source files, those modified files must carry "prominent notices stating that You changed the files"
3. **Retain attribution notices** -- keep all copyright, patent, trademark, and attribution notices from the original gdev source
4. **Propagate the NOTICE file** -- if gdev includes a NOTICE file, include a readable copy of its attribution notices in your distribution (see Section 5 below)

### What You Do NOT Need To Do

- You do NOT need to open-source your addon code
- You do NOT need to license your addon under Apache 2.0
- You do NOT need to share your source code with anyone
- You do NOT need to disclose your modifications (only mark that files were changed, not reveal what was changed)

### The Patent Grant

Apache 2.0 includes an explicit patent grant (Section 3). Contributors to gdev automatically grant users a "perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable" patent license. This patent grant terminates if you initiate patent litigation against any contributor over the licensed work. This is a benefit for addon authors: you receive patent protections from gdev contributors.

### Practical Approach

Most companies distributing Go binaries containing Apache-2.0 dependencies:
1. Include a `LICENSES/` directory in their distribution containing the Apache-2.0 license text
2. Include a `NOTICE` file aggregating attributions from all Apache-2.0 dependencies
3. Use automated tooling (`go-licenses`, `license-detector`) to track compliance

**Sources**: [Apache License 2.0 Text](https://www.apache.org/licenses/LICENSE-2.0), [Snyk Apache License Guide](https://snyk.io/articles/apache-license/), [FOSSA Apache 2.0 Guide](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/)

---

## 5. NOTICE File Propagation in Go Binaries

### The Requirement

Apache License 2.0 Section 4(d) requires: if a dependency includes a NOTICE file, any Derivative Works you distribute must include a readable copy of the attribution notices from that NOTICE file. This applies to each Apache-2.0 dependency that ships a NOTICE file.

### The Go-Specific Challenge

A typical Go binary may pull in dozens of Apache-2.0 licensed dependencies (especially in the Kubernetes ecosystem). Each may have its own NOTICE file. Since Go statically links everything into a single binary, all of these are "bundled" and their NOTICE requirements apply.

### The ASF's Guidance on Aggregation

Per the Apache Infrastructure team's [Licensing HOWTO](https://infra.apache.org/licensing-howto.html):

- "The LICENSE and NOTICE files must exactly represent the contents of the distribution they reside in"
- Only components physically included in a distribution affect these files
- Transitive dependencies receive the same treatment -- "only modifications are necessary if their contents are physically bundled"
- NOTICE files should be "as minimal as possible to reduce burden on downstream users"
- Copyright notices embedded within BSD and MIT license texts don't require duplication in NOTICE

### Practical Tooling for Go

**Google's `go-licenses` tool** is the industry-standard solution for Go projects:

```bash
# Install
go install github.com/google/go-licenses/v2@latest

# Generate a license report for your binary
go-licenses report ./cmd/my-addon

# Collect all compliance artifacts (licenses, notices, source) into a directory
go-licenses save ./cmd/my-addon --save_path=./third-party-licenses

# Check for forbidden licenses in your dependency tree
go-licenses check ./cmd/my-addon
```

The `save` command automatically determines what needs to be redistributed for each dependency based on its license type. For "notice" licenses (Apache-2.0, MIT, BSD), it collects the license file and copyright notices. For "reciprocal" licenses (MPL-2.0), it also collects source code.

**Elastic's `go-licence-detector`** is another option used in the Elastic ecosystem.

### Real-World Compliance Patterns

Teams typically adopt one of these approaches:

1. **Embedded `NOTICE` file in the binary distribution**: A single aggregated NOTICE file ships alongside the binary (or in a `.tar.gz`/`.zip` archive). This is the most common approach.

2. **`LICENSES/` directory**: A directory containing individual license files for each dependency, plus an aggregated NOTICE. Used by Kubernetes and many CNCF projects.

3. **Build-time generation**: CI/CD pipelines run `go-licenses save` to automatically generate the compliance directory, which is included in release artifacts. This ensures the NOTICE file stays synchronized with actual dependencies.

4. **License scanning in CI**: Teams add `go-licenses check` to CI pipelines to catch newly-added dependencies with incompatible licenses before they reach production.

### Minimal Viable NOTICE File

For a Go addon importing gdev (Apache-2.0), the NOTICE file needs to:
1. Include gdev's NOTICE content (if gdev has a NOTICE file)
2. Include NOTICE content from any transitive Apache-2.0 dependencies that have NOTICE files
3. NOT include entries for dependencies that don't have NOTICE files (MIT/BSD dependencies only need their license text preserved, not a NOTICE entry)

**Sources**: [ASF Licensing HOWTO](https://infra.apache.org/licensing-howto.html), [Google go-licenses](https://github.com/google/go-licenses), [Apache License 2.0 Section 4](https://www.apache.org/licenses/LICENSE-2.0)

---

## 6. Recommendations for gdev Addon Authors

### Recommended License: Apache-2.0

For open-source gdev addons, **Apache-2.0 is the strongest default choice**:

1. **Ecosystem alignment**: The overwhelming majority of Go/Kubernetes projects use Apache-2.0. Matching the framework license eliminates compatibility questions.
2. **Patent protection**: Apache-2.0's explicit patent grant protects both addon authors and users from patent claims by contributors.
3. **Permissive for downstream**: Users of your addon can incorporate it into proprietary products, maximizing adoption.
4. **Corporate acceptance**: Google, Red Hat, Microsoft, and most tech companies have pre-approved Apache-2.0 for both consumption and contribution. Fewer legal reviews required.

### Alternative: MIT

If simplicity is preferred and patent protection is not a concern, MIT is a solid alternative. It is fully compatible with Apache-2.0 and requires less boilerplate.

### If Weak Copyleft Is Desired: MPL-2.0

If addon authors want to ensure modifications to their specific files are shared back (but still allow proprietary addons to exist alongside), **MPL-2.0 is the correct choice for Go** -- not LGPL. MPL-2.0 works with static linking and is well-understood by corporate legal teams.

### If Proprietary: No License File (All Rights Reserved)

Proprietary addons are fully legal. Include Apache-2.0's license text and NOTICE file for gdev in your distribution. No source code disclosure required.

### Licenses to Avoid

- **GPL-2.0**: Incompatible with Apache-2.0. Do not use.
- **LGPL (any version)**: Unworkable with Go's static linking. Use MPL-2.0 instead.
- **AGPL-3.0**: Technically compatible but toxic to corporate adoption. Google bans it entirely. Most enterprises will refuse to use AGPL-licensed dependencies.
- **GPL-3.0**: Legally compatible but forces the entire binary to be GPL-3.0, which may surprise users and limit downstream adoption.

### Compliance Checklist for Addon Authors

Regardless of your addon's license, when distributing binaries that include Apache-2.0 gdev:

- [ ] Include a copy of the Apache License 2.0 text
- [ ] Include gdev's NOTICE file content (if one exists)
- [ ] Mark any modified gdev source files with change notices
- [ ] Run `go-licenses check` in CI to catch incompatible transitive dependencies
- [ ] Run `go-licenses save` at release time to generate compliance artifacts
- [ ] Include aggregated license/notice files in your release archives

---

## Sources

All raw source material is saved in the spike's `docs/` directory with `apache-compat-` prefix:

- `apache-compat-asf-gpl-compatibility.md` -- ASF's official GPL compatibility statement
- `apache-compat-asf-third-party-license-policy.md` -- ASF Category A/B/X license classifications
- `apache-compat-asf-licensing-howto.md` -- ASF guidance on assembling LICENSE/NOTICE files
- `apache-compat-license-text-key-clauses.md` -- Apache 2.0 Section 1 (Derivative Works) and Section 4 (Redistribution)
- `apache-compat-fossa-apache-2-guide.md` -- FOSSA's Apache 2.0 compatibility overview
- `apache-compat-snyk-apache-license-guide.md` -- Snyk's Apache 2.0 requirements summary
- `apache-compat-licensecheck-compatibility-guide.md` -- LicenseCheck.io bidirectional compatibility matrix
- `apache-compat-thehyve-combining-licenses.md` -- The Hyve's license combination analysis
- `apache-compat-wikipedia-license-compatibility.md` -- Wikipedia license compatibility overview
- `apache-compat-google-license-classification.md` -- Google's internal license classification system
- `apache-compat-google-go-licenses-tool.md` -- Google's go-licenses tool for Go compliance
- `apache-compat-lgpl-go-static-linking.md` -- Analysis of LGPL incompatibility with Go
- `apache-compat-go-license-statistics.md` -- Go ecosystem license usage statistics
