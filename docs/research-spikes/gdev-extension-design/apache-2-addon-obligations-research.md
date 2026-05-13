# Apache License 2.0 Obligations for gdev Addon Authors

## Context

This analysis examines the specific legal obligations that apply when writing custom addon packages that import the gdev framework (`fastcat.org/go/gdev`, Apache-2.0 licensed) via Go's module system. The addons use gdev's public API (specifically its `Addon` registration interface) without modifying gdev's source code. Because Go statically links all dependencies into a single binary, the compiled output contains both gdev and addon code in one executable.

**Disclaimer**: This is a technical analysis based on the license text and published legal commentary, not legal advice. Consult a qualified attorney for binding guidance.

---

## 1. Are Our Addons "Derivative Works" Under Apache-2.0?

### The License Text

Section 1 of Apache License 2.0 defines "Derivative Works" as:

> any work, whether in Source or Object form, that is based on (or derived from) the Work and for which the editorial revisions, annotations, elaborations, or other modifications represent, as a whole, an original work of authorship. **For the purposes of this License, Derivative Works shall not include works that remain separable from, or merely link (or bind by name) to the interfaces of, the Work and Derivative Works thereof.**

This definition contains a critical carve-out: works that "remain separable from, or merely link (or bind by name) to the interfaces of" the licensed work are explicitly excluded from the derivative works definition.

### Application to gdev Addons

Our addon code:
- Imports gdev packages and calls their public API (interfaces, functions, types)
- Does not modify gdev source code
- Remains in separate Go packages with their own module paths
- Is "separable" in that the addon source code stands alone and could be compiled against any compatible interface

Under the Apache-2.0 definition, **our addon source code is almost certainly NOT a derivative work of gdev**. The addons "merely link (or bind by name) to the interfaces of" gdev. The addon packages are separable works that happen to call gdev's public API — precisely the scenario the carve-out was designed for.

### The Static Linking Question

Go compiles everything into a single statically-linked binary. Does this change the analysis?

**Short answer: No, not under Apache-2.0.**

The derivative works carve-out in Apache-2.0 uses the word "link" explicitly, which covers both static and dynamic linking. The license text does not distinguish between linking methods — it says works that "merely link...to the interfaces of" the Work are excluded. This is a deliberate drafting choice.

This is a crucial distinction from the GPL/LGPL debate. Under GPL, the question of whether static linking creates a derivative work is genuinely contested (the FSF says yes; many lawyers disagree — see the LWN analysis by Michael Kerrisk covering arguments from Lawrence Rosen, James Bottomley, and others). But Apache-2.0 sidesteps this entire debate by explicitly defining the boundary: if you link to the interfaces, you are not creating a derivative work regardless of the linking method.

**However**, the *compiled binary* that contains both gdev code and addon code is a different matter. The binary is a distribution that includes Apache-2.0 licensed code (gdev's compiled object code). Even though the addon source code is not a derivative work, the binary redistribution triggers Section 4 obligations because you are redistributing "copies of the Work" (gdev's compiled code) within it.

### What Legal Experts Say

- **Lawrence Rosen** (author of *Open Source Licensing*): argues that "no kind of linking (dynamic or static) creates a derivative work per se" (cited in LWN.net analysis)
- **Claus-Peter Wiedemann**: suggests a multi-factor test — functional dependency, data exchange, interface standardization — but notes linking alone is not determinative
- The **Apache Foundation FAQ** confirms that code using Apache-licensed libraries can be distributed under different terms, which implicitly acknowledges that such code is not a derivative work

### Conclusion on Derivative Works

The addon **source code** is not a derivative work of gdev under Apache-2.0's own definition. The compiled **binary** contains gdev code and triggers redistribution obligations (Section 4), but the addon code within that binary can be under any license. This is consistent across all authoritative sources consulted.

**Sources**: [Apache License 2.0 text](https://www.apache.org/licenses/LICENSE-2.0.txt), [LWN — Dynamic Linking and Derivative Works](https://lwn.net/Articles/548216/), [Apache Foundation License FAQ](https://www.apache.org/foundation/license-faq.html)

---

## 2. Redistribution Obligations When Distributing a Binary

When we distribute a Go binary that includes compiled gdev code, Section 4 of the Apache License 2.0 applies because we are distributing "copies of the Work...in Object form." Here are the specific obligations:

### 2a. Include a Copy of the Apache-2.0 License

**Required.** Section 4(a): "You must give any other recipients of the Work or Derivative Works a copy of this License."

In practice, this means the full text of the Apache License 2.0 must accompany the binary distribution. For a Go binary, this is typically done by:
- Including a `LICENSE` or `THIRD_PARTY_LICENSES` file alongside the binary
- Embedding license texts into the binary itself (some Go projects use `embed` for this)
- Including them in a documentation archive or alongside the download

### 2b. State Changes to Modified Files

**Not applicable in our case.** Section 4(b): "You must cause any modified files to carry prominent notices stating that You changed the files."

Since we are not modifying gdev's source code, this requirement does not apply to us.

### 2c. Retain Copyright and Attribution Notices

**Required.** Section 4(c): "You must retain, in the Source form of any Derivative Works that You distribute, all copyright, patent, trademark, and attribution notices from the Source form of the Work."

Note: This requirement explicitly applies to "Source form" distribution. If we only distribute binaries (not gdev source), the literal text of 4(c) is scoped to source distributions. However, best practice — and the intent of the license — is to preserve attribution in all distribution forms.

### 2d. Include NOTICE File Contents

**Conditionally required.** Section 4(d): "If the Work includes a 'NOTICE' text file as part of its distribution, then any Derivative Works that You distribute must include a readable copy of the attribution notices contained within such NOTICE file."

The gdev repository needs to be checked for a NOTICE file. If gdev includes one, we must reproduce its attribution notices in at least one of:
1. A NOTICE text file distributed alongside the binary
2. Within documentation provided with the binary
3. Within a display generated by the binary (e.g., a `--licenses` or `--version` flag output)

Per the Apache Infrastructure guidance on assembling LICENSE and NOTICE files: "The LICENSE and NOTICE files must exactly represent the contents of the distribution they reside in." Only bundled components need to be documented — non-bundled dependencies should not be included.

### 2e. Transitive Dependencies

Go binaries include all transitive dependencies. Each Apache-2.0 dependency (and dependencies under other licenses) has its own redistribution requirements. If gdev depends on other Apache-2.0 libraries that include their own NOTICE files, those attribution notices must also be propagated into our distribution.

This is where tooling becomes essential — see Section 5 below.

### Summary of Binary Distribution Requirements

| Obligation | Applies? | Action Required |
|---|---|---|
| Include Apache-2.0 license text | Yes | Ship LICENSE file with binary |
| State changes to modified files | No | We don't modify gdev |
| Retain copyright/attribution notices | Yes | Preserve in source and/or binary distribution |
| Include NOTICE file contents | If gdev has one | Reproduce attribution notices |
| Handle transitive dependencies | Yes | Each dependency's license must be satisfied |

**Sources**: [Apache License 2.0 Section 4](https://www.apache.org/licenses/LICENSE-2.0.txt), [Apache Infrastructure — Assembling LICENSE and NOTICE](https://infra.apache.org/licensing-howto.html), [Applying the Apache License](https://www.apache.org/legal/apply-license.html), [FOSSA — Apache License 101](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/), [Sbomify — Apache 2.0 Guide](https://sbomify.com/2026/01/07/apache-license-2-guide/)

---

## 3. Can Our Addons Use a Different License (Including Proprietary)?

### Yes — Unequivocally

This is one of the core features of permissive licenses like Apache-2.0. Multiple authoritative sources confirm this:

**The license itself** (Section 4, final paragraph): "You may add Your own copyright statement to Your modifications and may provide additional or different license terms and conditions for use, reproduction, or distribution of Your modifications, or for any such Derivative Works as a whole, provided Your use, reproduction, and distribution of the Work otherwise complies with the conditions stated in this License."

**The Apache Foundation FAQ**: Confirms you may distribute modifications under different licenses, stating you "must comply with [the Apache license's] terms" but may otherwise choose your own licensing.

**FOSSA**: "Companies can include the licensed code in proprietary software."

**Snyk**: The license "gives users permission to reuse code for nearly any purpose, including using the code as part of proprietary software."

**Sbomify**: "Unlike copyleft licenses such as GPL, Apache 2.0 does not mandate...licensing derivative works under Apache 2.0 [or] open-sourcing modifications."

### What This Means in Practice

Our addon source code can be:
- **Proprietary / closed source** — no obligation to share addon source
- **Licensed under any other open source license** (MIT, BSD, GPL, etc.)
- **Licensed under Apache-2.0** if we choose to align with gdev

The key constraint is that **gdev's own code retains its Apache-2.0 license**. We cannot relicense gdev itself. The binary we distribute must still comply with Section 4 obligations for the gdev code it contains (include license text, attribution, NOTICE if applicable).

### The Boundary

Think of it as two layers:
1. **Our addon code**: any license we choose, including proprietary
2. **gdev code compiled into the binary**: remains Apache-2.0, and redistribution obligations apply to it

We do NOT need to:
- Release our addon source code
- License our addons under Apache-2.0
- Grant patent licenses on our own code (unless we choose to via our own license)

We DO need to:
- Include the Apache-2.0 license text with our binary
- Preserve gdev's copyright and attribution notices
- Include gdev's NOTICE file contents if one exists

**Sources**: [Apache License 2.0 Section 4](https://www.apache.org/licenses/LICENSE-2.0.txt), [Apache Foundation FAQ](https://www.apache.org/foundation/license-faq.html), [FOSSA](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/), [Snyk](https://snyk.io/articles/apache-license/), [Sbomify](https://sbomify.com/2026/01/07/apache-license-2-guide/)

---

## 4. Patent Implications for Addon Authors

### The Patent Grant (Section 3)

Each gdev contributor grants every user:

> a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable...patent license to make, have made, use, offer to sell, sell, import, and otherwise transfer the Work, where such license applies solely to those patent claims licensable by such Contributor that are necessarily infringed by their Contribution(s) alone or by combination of their Contribution(s) with the Work to which such Contribution(s) was submitted.

**What this means for addon authors:**
- We receive a patent license from gdev's contributors covering patents that read on gdev's code
- This protects us from patent claims by gdev contributors related to the functionality they contributed
- The grant covers both using gdev directly and combining it with other code (our addons)

### Scope Limitations

The patent grant is narrower than it might first appear:

1. **Only contributor-held patents**: The grant covers only patents "licensable by such Contributor." If a third party (not a gdev contributor) holds a patent that gdev's code infringes, the Apache-2.0 patent grant does not protect us from that third party.

2. **Only patents reading on contributions as submitted**: Per the Apache Foundation FAQ, "patent claims are licensed only to the extent they read on contributions as originally submitted or on combinations with the specific ASF product at contribution time." If gdev evolves and later happens to infringe a contributor's patent that wasn't infringed when they contributed, that patent is not automatically licensed.

3. **No coverage for addon-specific patents**: The patent grant covers gdev's code, not our addon code. If our addons independently infringe someone's patent, the Apache-2.0 grant from gdev contributors provides no protection.

### The Patent Retaliation Clause

> If You institute patent litigation against any entity (including a cross-claim or counterclaim in a lawsuit) alleging that the Work or a Contribution incorporated within the Work constitutes direct or contributory patent infringement, then any patent licenses granted to You under this License for that Work shall terminate as of the date such litigation is filed.

**What triggers termination:**
- Filing a lawsuit claiming that gdev itself (or a contribution to gdev) infringes your patent
- This includes cross-claims and counterclaims

**What does NOT trigger termination:**
- Filing patent litigation about unrelated software
- Filing patent litigation about your addon code
- Defensive patent litigation where someone else sues you first (the clause says "You institute" — initiating, not defending)
- Non-patent legal claims (copyright, trademark, contract disputes)

**Practical implications for addon authors:**
- If we use gdev and later discover we hold a patent that gdev infringes, we cannot sue gdev's contributors for that infringement without losing our own patent license to gdev
- This is a "patent peace" provision — it discourages patent aggression within the community
- For most addon authors who are consumers (not patent holders in the gdev space), this clause is unlikely to ever be triggered

### What Addon Authors Do NOT Owe

Importantly, by merely using gdev's API, addon authors:
- Do NOT grant any patent license on their own code to anyone
- Do NOT expose their own patents to any retaliation clause
- Are NOT required to include patent grants in their own license

If addon authors choose a license for their code that includes patent provisions (e.g., Apache-2.0), those provisions apply independently to their own contributions, not as a consequence of using gdev.

**Sources**: [Apache License 2.0 Section 3](https://www.apache.org/licenses/LICENSE-2.0.txt), [Opensource.com — Apache 2 Patent License](https://opensource.com/article/18/2/apache-2-patent-license), [PatentPC — Patent Provisions in OSS Licenses](https://patentpc.com/blog/understanding-the-patent-provisions-in-popular-open-source-licenses), [EU IP Helpdesk — Apache 2.0 Guide](https://intellectual-property-helpdesk.ec.europa.eu/news-events/news/how-apache-20-2020-02-13_en), [Apache Foundation FAQ](https://www.apache.org/foundation/license-faq.html)

---

## 5. Practical Compliance Steps

### For the Addon Source Repository

1. **Choose your addon's license.** You may use any license (Apache-2.0, MIT, BSD, proprietary, etc.). The choice has no impact on gdev's license and gdev's license imposes no constraints on this choice.

2. **Include gdev in `go.mod` normally.** No special annotations are needed. Go module declarations are factual dependency statements, not license grants.

3. **No license headers from gdev are needed in your addon source files.** Your files are yours. You only need license headers matching your chosen license.

### For Binary Distribution

This is where compliance work is required, because the compiled binary contains gdev's code.

4. **Include the Apache License 2.0 text.** Ship a copy of the full Apache-2.0 license alongside your binary. Common approaches:
   - A `LICENSE` or `THIRD_PARTY_LICENSES` directory/file next to the binary
   - A `--licenses` CLI flag that prints license information
   - A `licenses/` directory in your distribution archive

5. **Check for and include gdev's NOTICE file.** If gdev includes a NOTICE file, its attribution notices must be reproduced in your distribution. Check the gdev repository root for a `NOTICE` or `NOTICE.txt` file.

6. **Handle all transitive dependencies.** Your binary will include code from gdev's dependencies too, each with their own license. Use automated tooling to enumerate and collect these.

### Recommended Tooling for Go

7. **Use `google/go-licenses`** to automate compliance:

   ```bash
   # Install
   go install github.com/google/go-licenses@latest

   # Check for forbidden licenses in your dependency tree
   go-licenses check ./cmd/your-binary

   # Generate a compliance report (CSV of all dependencies and their licenses)
   go-licenses report ./cmd/your-binary

   # Save all license files and notices to a directory for redistribution
   go-licenses save ./cmd/your-binary --save_path=./third_party_licenses
   ```

   The `save` command collects all license files, copyright notices, and (for copyleft dependencies) source code needed for redistribution compliance. The output directory can be included in your distribution archive.

8. **Alternatively, use Go's built-in `go version -m`** to list modules in a compiled binary, then collect licenses manually:

   ```bash
   go version -m ./your-binary | grep dep
   ```

### Distribution Checklist

For each release of a binary containing gdev:

- [ ] Full text of Apache License 2.0 included in distribution
- [ ] gdev's NOTICE file contents reproduced (if one exists)
- [ ] All transitive dependency licenses collected and included
- [ ] No GPL-v2-only dependencies pulled in (incompatible with Apache-2.0; GPL-v3 is fine)
- [ ] Copyright and attribution notices from gdev preserved
- [ ] If gdev source was modified: prominent change notices added to modified files (N/A if unmodified)
- [ ] License compliance artifacts are up to date with current `go.sum` contents

### What You Do NOT Need to Do

- Share your addon source code
- License your addon under Apache-2.0
- Include Apache boilerplate headers in your addon source files
- Grant patent licenses on your own code
- Get permission from gdev's authors to write addons
- Attribute gdev in your marketing or product name (though it's good practice)
- Include gdev's license in your addon source repository (only in binary distributions that contain compiled gdev code)

**Sources**: [Apache Infrastructure — Assembling LICENSE and NOTICE](https://infra.apache.org/licensing-howto.html), [Applying the Apache License](https://www.apache.org/legal/apply-license.html), [google/go-licenses](https://github.com/google/go-licenses), [Go OpenSource and Licensing](https://medium.com/@henvic/opensource-and-go-what-license-f6b36c201854), [TLDRLegal — Apache 2.0](https://www.tldrlegal.com/license/apache-license-2-0-apache-2-0)

---

## Summary Matrix

| Question | Answer |
|---|---|
| Is addon source code a derivative work? | **No** — it "merely links to the interfaces of" gdev |
| Does the compiled binary trigger obligations? | **Yes** — it contains copies of gdev's compiled code |
| Must we include the Apache-2.0 license text? | **Yes** — with every binary distribution |
| Must we include gdev's NOTICE file? | **Yes, if one exists** — check the gdev repo |
| Must we state changes to gdev? | **Only if we modify gdev source** (we don't) |
| Can addons be proprietary? | **Yes** — any license, including closed source |
| Must we share addon source code? | **No** — Apache-2.0 never requires this |
| Do we receive a patent grant from gdev? | **Yes** — covering contributor-held patents on gdev code |
| Must we grant patents on our addon code? | **No** — only if our chosen license includes such a grant |
| Can we sue gdev for patent infringement? | **Technically yes, but it terminates our patent license from gdev** |
| Does Go static linking change any of this? | **No** — Apache-2.0 explicitly covers linking to interfaces |

---

## Key Risk: GPL-v2 Transitive Dependencies

One practical risk worth noting: Apache-2.0 is **incompatible with GPL-v2** (per both the Apache Foundation and the Free Software Foundation). If gdev or any of its transitive dependencies pulls in GPL-v2-only code, it creates a license conflict. This is worth verifying with `go-licenses check` before distribution. Apache-2.0 IS compatible with GPL-v3, MIT, BSD, and MPL-2.0.

---

## Sources Consulted

1. [Apache License, Version 2.0 — Full Text](https://www.apache.org/licenses/LICENSE-2.0.txt) → `docs/apache-2-license-full-text.md`
2. [Apache Licensing and Distribution FAQ](https://www.apache.org/foundation/license-faq.html) → `docs/apache-foundation-license-faq.md`
3. [Applying the Apache License, Version 2.0](https://www.apache.org/legal/apply-license.html) → `docs/apache-applying-license-guidance.md`
4. [Assembling LICENSE and NOTICE Files — Apache Infrastructure](https://infra.apache.org/licensing-howto.html) → `docs/apache-licensing-howto-notice-files.md`
5. [Open Source Licenses 101: Apache License 2.0 — FOSSA](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/) → `docs/fossa-apache-2-license-101.md`
6. [Apache License 2.0 Explained — Snyk](https://snyk.io/articles/apache-license/) → `docs/snyk-apache-license-explained.md`
7. [Apache License 2.0 Explained in Plain English — TLDRLegal](https://www.tldrlegal.com/license/apache-license-2-0-apache-2-0) → `docs/tldrlegal-apache-2.md`
8. [Apache License — Wikipedia](https://en.wikipedia.org/wiki/Apache_License) → `docs/wikipedia-apache-license.md`
9. [How to Apache 2.0? — EU IP Helpdesk](https://intellectual-property-helpdesk.ec.europa.eu/news-events/news/how-apache-20-2020-02-13_en) → `docs/eu-ip-helpdesk-apache-2-guide.md`
10. [How to Make Sense of the Apache 2 Patent License — Opensource.com](https://opensource.com/article/18/2/apache-2-patent-license) → `docs/opensource-com-apache-patent-license.md`
11. [Understanding Patent Provisions in Popular OSS Licenses — PatentPC](https://patentpc.com/blog/understanding-the-patent-provisions-in-popular-open-source-licenses) → `docs/patentpc-patent-provisions-oss-licenses.md`
12. [Apache License 2.0 Guide — Sbomify](https://sbomify.com/2026/01/07/apache-license-2-guide/) → `docs/sbomify-apache-2-license-guide.md`
13. [Dynamic Linking and Derivative Works — LWN.net](https://lwn.net/Articles/548216/) → `docs/lwn-dynamic-linking-derivative-works.md`
14. [Open Source and Go: What License? — Medium](https://medium.com/@henvic/opensource-and-go-what-license-f6b36c201854) → `docs/go-opensource-licensing-static-linking.md`
15. [google/go-licenses — GitHub](https://github.com/google/go-licenses) → `docs/google-go-licenses-tool.md`
