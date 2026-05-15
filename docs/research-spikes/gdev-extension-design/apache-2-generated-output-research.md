# Apache 2.0 License and Generated Output: Comprehensive Analysis

## Executive Summary

When an Apache-2.0 licensed tool generates output files (configuration, templates, scaffolding), the Apache 2.0 license does **not automatically apply** to that output. The critical factor is whether the output contains **copyrightable expression originating from the tool itself**. This is a well-established principle in open source licensing, supported by FSF guidance, the Bison precedent, and consistent real-world practice across major scaffolding tools.

For gdev's addon system, the practical answer is:

- **Generated config files (devenv.nix, .envrc, YAML):** Almost certainly NOT covered by Apache 2.0. These are functional configuration with minimal creative expression, generated from user input.
- **Verbatim-copied files via embed.FS (skills .md files, CLAUDE.md templates):** These ARE covered by Apache 2.0 to the extent they contain copyrightable expression from the Apache-licensed source. However, Apache 2.0 is permissive -- the user can use them freely with only attribution obligations.
- **Template-rendered output:** Depends on how much of the output originates from the template vs. user input. Mostly-template output with minimal variable substitution leans toward the template's license applying; heavily user-driven output with minimal template structure leans toward user ownership.

The best practice is to be explicit: document in gdev's README/docs what license applies to generated output, and give users a mechanism to control license headers on generated files (as kubebuilder does).

---

## 1. The Core Legal Principle: Tool License vs. Output License

### 1.1 The FSF's Authoritative Statement

The Free Software Foundation provides the clearest articulation of the governing principle:

> "The output of a program is not, in general, covered by the copyright on the code of the program."

The exception:

> "...when the program displays a full screen of text and/or art that comes from the program. Then the copyright on that text and/or art covers the output."

**Source:** [FSF Licensing Lab FAQ](https://www.fsf.org/blogs/licensing/licensing-and-compliance-lab-the-most-frequently-asked-frequently-asked-questions)

This principle is license-agnostic -- it applies equally to GPL, Apache 2.0, MIT, and any other license. The question is never "what license is the tool?" but rather "does the output contain copyrightable expression from the tool?"

### 1.2 Two Distinct Scenarios

**Scenario A: Tool transforms user input into output.**
Example: A compiler transforms user-written source code into machine code. The compiler's license does not apply to the output because the creative expression in the output originates from the user, not the compiler.

**Scenario B: Tool copies parts of itself into output.**
Example: GNU Bison copies its parser skeleton (the yyparse() function, parser tables) verbatim into generated parser files. The GPL applies to those copied portions because they are copyrightable expression originating from Bison's source code. Bison needed an explicit exception to allow proprietary use of its output.

**Source:** [Bison Licensing Conditions](https://www.gnu.org/software/bison/manual/html_node/Conditions.html)

### 1.3 The GCC Runtime Exception Model

GCC faces the same issue: compiled programs may incorporate portions of GCC's header files and runtime libraries. The FSF created a specific "GCC Runtime Library Exception" to permit this:

> "When you use GCC to compile a program, GCC may combine portions of certain GCC header files and runtime libraries with the compiled program. The purpose of this Exception is to allow compilation of non-GPL (including proprietary) programs to use, in this way, the header files and runtime libraries covered by this Exception."

The existence of these exceptions **proves the principle**: without them, the tool's license WOULD apply to those copied portions.

---

## 2. Applying the Principle to Apache 2.0

### 2.1 Apache 2.0 License Definitions

The Apache 2.0 license defines key terms relevant to this analysis:

- **"Work"**: "the work of authorship, whether in Source or Object form, made available under the License"
- **"Derivative Works"**: "any work, whether in Source or Object form, that is based on (or derived from) the Work" -- but explicitly **excludes** "works that remain separable from, or merely link (or bind by name) to the interfaces of, the Work"
- **"Object form"**: "any form resulting from mechanical transformation or translation of a Source form, including but not limited to compiled object code, generated documentation, and conversions to other media types"

**Source:** [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0)

### 2.2 The Separability Clause

The Apache 2.0 license's exclusion of "works that remain separable" is critical. Generated configuration files are inherently separable from the tool that generated them:

- They exist as independent files in the user's project
- They function without the generating tool being present
- They serve a different purpose than the tool itself
- They can be created manually without the tool

This separability strongly suggests generated config files are NOT "Derivative Works" under Apache 2.0.

### 2.3 Why Apache 2.0 Makes This Less Concerning

Even in the worst case -- where generated output IS considered a Derivative Work containing Apache-2.0-licensed material -- the consequences are minimal because Apache 2.0 is permissive:

- Users CAN use the output commercially
- Users CAN modify the output
- Users CAN distribute the output under ANY license
- Users only MUST retain copyright/attribution notices and state changes
- There is NO copyleft obligation (unlike GPL)

This is fundamentally different from the GPL situation where Bison and GCC needed explicit exceptions. Under Apache 2.0, the practical impact of the license applying to output is just an attribution requirement.

---

## 3. Analysis of Each gdev Output Category

### 3.1 devenv.nix / devenv.yaml -- Generated from Code Logic

**How generated:** Go code marshals a struct to YAML, or uses text/template to produce Nix expressions based on user selections from a wizard.

**License analysis:** The output is a **transformation of user input**. The user makes choices (language, services, tools), and the code translates those choices into configuration syntax. The creative expression in the output (which packages to include, which services to enable) originates from the user, not the tool.

The template structure itself (Nix syntax patterns, YAML schema) is **functional** -- there is only one correct way to express "enable Go language support in devenv.nix." Functional elements are not copyrightable.

**Conclusion:** Apache 2.0 almost certainly does NOT apply. These files are analogous to compiler output -- a transformation of user input into a different format. Additionally, configuration files with minimal creative expression may fall below the **threshold of originality** required for copyright protection entirely.

**Supporting precedent:** The FSFE explicitly identifies "config files that contain no creative expression" and "files automatically generated by code" as lacking the intellectual creativity required for copyright protection. ([FSFE Legal Corner](https://fsfe.org/news/2025/news-20250515-01.en.html))

### 3.2 CLAUDE.md -- Template-Rendered with Substantial Template Content

**How generated:** A Go text/template containing substantial prose (instructions, conventions, workflow descriptions) is rendered with project-specific variables (project name, language, team conventions) substituted in.

**License analysis:** This is the most nuanced case. The template itself contains significant **creative expression** -- paragraphs of instruction text, structured guidance, workflow descriptions. If the generated CLAUDE.md is 90% template prose and 10% variable substitution, most of the copyrightable expression in the output originates from the Apache-2.0-licensed template.

This is closer to the **Bison scenario**: the tool copies substantial portions of its own copyrighted content into the output. The Apache 2.0 license would apply to those copied portions.

**Conclusion:** Apache 2.0 likely applies to the template-derived content. However, because Apache 2.0 is permissive, the user can freely use, modify, and redistribute the generated CLAUDE.md. The obligation is limited to retaining attribution if they redistribute the file.

**Recommended approach:** Include a small comment like `<!-- Generated by gdev (Apache-2.0) -->` and note in documentation that generated CLAUDE.md files may be freely modified.

### 3.3 .claude/settings.json -- Generated from Code Logic

**How generated:** Go code marshals a struct (containing permission lists, MCP server configs, etc.) to JSON based on user wizard selections.

**License analysis:** JSON configuration files are **functional** -- they express machine-readable settings with no creative prose. The structure is dictated by the consuming application's schema (Claude Code), not by creative choices of the tool author. The specific values come from the user's selections.

**Conclusion:** Apache 2.0 almost certainly does NOT apply. JSON config files are unlikely to meet the threshold of originality for copyright protection, and the content is a direct transformation of user input.

### 3.4 Skills Files (.md) -- Verbatim Copy via embed.FS

**How generated:** Markdown files are bundled into the Go binary via `//go:embed` and then copied verbatim to the user's project directory.

**License analysis:** This is the **clearest case where Apache 2.0 applies**. The files are literally copied from the Apache-2.0-licensed source code. They are part of the "Work" as defined by the license. The `embed.FS` mechanism is just a delivery vehicle -- whether the files are copied from a directory on disk or extracted from a binary, they remain the same copyrighted work.

The Go `embed.FS` mechanism does not change the licensing status of embedded files. Embedding a file in a binary is analogous to including it in a distribution archive -- the file's license travels with it.

**Conclusion:** Apache 2.0 applies. The copied skill files are part of the Apache-2.0-licensed work. Users receive them under Apache 2.0 terms: they can use, modify, and redistribute them, but must retain attribution.

**Recommended approach:** Include an SPDX identifier comment in each skills file: `<!-- SPDX-License-Identifier: Apache-2.0 -->` and optionally a brief attribution line.

### 3.5 .envrc -- Generated from Template

**How generated:** A small template producing a few lines of direnv configuration (typically `use devenv` or similar).

**License analysis:** An .envrc file is typically 1-5 lines of functional shell commands. It falls well below the threshold of originality for copyright protection. There is effectively only one way to write "activate devenv in direnv" -- the merger doctrine (where idea and expression merge because there's only one way to express an idea) applies.

**Conclusion:** Apache 2.0 does NOT apply. The file is too trivial and functional to be copyrightable.

---

## 4. Real-World Precedent: How Other Tools Handle This

### 4.1 Kubebuilder (Apache-2.0) -- The Gold Standard

Kubebuilder is the most directly relevant precedent: an Apache-2.0 licensed Go tool that generates scaffolded project files.

**Approach:**
- Defaults to Apache 2.0 license headers on all generated .go files
- Provides `--license` flag to specify a different license (or `--license none` for no headers)
- Provides `--owner` flag to set the copyright owner in generated files
- Uses `hack/boilerplate.go.txt` as the source for license headers, which users can customize
- The generated project is treated as belonging to the user, with kubebuilder providing the scaffolding

**Key insight:** Kubebuilder's design acknowledges that while the tool is Apache 2.0, the generated project should be owned and licensed by the user. The default Apache 2.0 headers are a reasonable starting point, not a legal requirement.

**Real-world issue:** The [Azure Databricks Operator](https://github.com/Azure/azure-databricks-operator/issues/146) encountered a license mismatch when kubebuilder's default Apache 2.0 headers appeared in their MIT-licensed project, demonstrating the importance of explicit license control.

**Source:** [Kubebuilder Boilerplate Docs](https://book-v1.book.kubebuilder.io/beyond_basics/boilerplate)

### 4.2 Create-React-App (MIT) -- Clean Separation

**Approach:**
- The tool itself is MIT licensed
- The template package (cra-template) is MIT licensed
- Generated projects have **NO license field by default** -- the user is expected to add their own
- The generated project contains no license headers from the tool

**Key insight:** CRA treats generated output as entirely user-owned. The tool's license applies to the tool, not to its output.

### 4.3 Cookiecutter (BSD-3-Clause) -- User Ownership by Design

**Approach:**
- Templates use Jinja variables for copyright: `Copyright (c) {{ cookiecutter.year }}, {{ cookiecutter.full_name }}`
- The generated project's license is determined by the template, not the tool
- Generated files have the user's name and year in copyright notices

**Key insight:** Cookiecutter makes user ownership explicit by injecting the user's identity into generated license files.

### 4.4 Terraform Provider Scaffolding (MPL-2.0) -- Ambiguity Example

**Approach:**
- The scaffolding template carries MPL 2.0
- HashiCorp stated the license is "not a placeholder"
- But HashiCorp also documents that providers can use "various compatible licenses"
- Community confusion exists about whether generated providers inherit MPL 2.0

**Key insight:** This is an example of what happens when a tool doesn't clearly document the licensing status of generated output -- developers are confused.

**Source:** [HashiCorp Discuss Thread](https://discuss.hashicorp.com/t/licence-for-provider-using-the-scaffholding/45822)

---

## 5. The Threshold of Originality Question

A separate but important consideration is whether generated config files are even **copyrightable** in the first place.

### 5.1 What Falls Below the Threshold

The FSFE's legal analysis identifies categories that lack sufficient originality for copyright:

- "Config files that contain no creative expression"
- "Files automatically generated by code"
- Generic/trivial code (the "Hello, World!" example)
- Functional expressions with no creative choices

### 5.2 Applying to gdev's Output

| Output File | Creative Expression Level | Likely Copyrightable? |
|---|---|---|
| devenv.nix | Low -- functional config, dictated by schema | Unlikely |
| devenv.yaml | Very low -- key-value pairs, schema-dictated | Unlikely |
| .envrc | Minimal -- 1-5 lines, single correct expression | No |
| settings.json | None -- machine-generated key-value pairs | No |
| CLAUDE.md | Moderate-to-high -- substantial prose content | Yes, the prose portions |
| Skills files | High -- substantial original markdown content | Yes |

### 5.3 Implications

For files that fall below the threshold of originality, **no license applies at all** -- they are in the public domain (or more precisely, they are not subject to copyright protection). This means the Apache 2.0 question is moot for most of gdev's generated config files.

---

## 6. Best Practices for gdev

Based on the research, here are concrete recommendations:

### 6.1 Documentation

Add a section to gdev's README or docs explicitly stating:

> **License of Generated Files:** Files generated by `qsdev init` are intended for use in your project under whatever license you choose. Configuration files (devenv.nix, devenv.yaml, .envrc, settings.json) are generated from your inputs and are not subject to the Apache 2.0 license. Content files copied from gdev's embedded resources (skills files, CLAUDE.md templates) contain material originally licensed under Apache 2.0; you may freely use, modify, and redistribute these files, with attribution appreciated but not strictly required for most use cases given the functional nature of the content.

### 6.2 License Headers on Generated Files

Follow kubebuilder's pattern with adaptations:

| File Type | Recommended Header | Rationale |
|---|---|---|
| devenv.nix | `# Generated by gdev` (no license header) | Functional config, not copyrightable |
| devenv.yaml | `# Generated by gdev` (no license header) | Functional config, not copyrightable |
| .envrc | `# Generated by gdev` (no license header) | Trivial, not copyrightable |
| settings.json | None (JSON doesn't support comments) | Not copyrightable |
| CLAUDE.md | `<!-- Generated by gdev - modify freely -->` | Contains template prose |
| Skills .md | `<!-- SPDX-License-Identifier: Apache-2.0 -->` | Verbatim copy of Apache-2.0 content |

### 6.3 User Override Mechanism

Provide a `--license` flag or config option (following kubebuilder's precedent) that allows users to:
- Specify what license header appears on generated files
- Use `--license none` to suppress all license headers
- Customize attribution text

### 6.4 embed.FS Best Practice

For files shipped via `embed.FS` and copied verbatim:
- Include SPDX identifiers in the source files (they'll travel with the copy)
- Document in the addon's README that these files are Apache-2.0 licensed
- Make it clear users can modify them freely (Apache 2.0 permits this)

### 6.5 NOTICE File

If gdev or its addons include a NOTICE file (as Apache 2.0 contemplates), document which portions of generated output originate from the Apache-2.0 codebase. This provides clean attribution without creating confusion about the user's rights.

---

## 7. Summary Decision Matrix

| Question | Answer | Confidence |
|---|---|---|
| Does Apache 2.0 auto-apply to all generated output? | **No** | High |
| Does it apply to verbatim-copied embed.FS files? | **Yes** (to copyrightable portions) | High |
| Does it apply to template-rendered config files? | **No** (functional, below originality threshold) | High |
| Does it apply to template-rendered prose (CLAUDE.md)? | **Partially** (to template-originated prose) | Medium |
| Does it apply to code-logic-generated JSON/YAML? | **No** | High |
| Can users relicense generated output? | **Yes** (Apache 2.0 is permissive) | High |
| Should we add license headers to generated files? | **Selectively** (only to files with copyrightable content) | High |
| Should we document the licensing of generated output? | **Yes, always** | High |

---

## 8. Sources

All raw sources saved to `research-spikes/gdev-extension-design/docs/` with `apache-generated-` prefix:

1. [Apache License 2.0 Full Text](https://www.apache.org/licenses/LICENSE-2.0) -- License definitions and redistribution terms
2. [ASF Source Header Policy](https://apache.org/legal/src-headers.html) -- Apache's own guidance on which files need headers
3. [FSF Licensing Lab FAQ](https://www.fsf.org/blogs/licensing/licensing-and-compliance-lab-the-most-frequently-asked-frequently-asked-questions) -- "Output of a program" principle
4. [GNU Bison Licensing Conditions](https://www.gnu.org/software/bison/manual/html_node/Conditions.html) -- Canonical precedent for tool output licensing
5. [FSFE Originality Threshold](https://fsfe.org/news/2025/news-20250515-01.en.html) -- When code/config is too trivial to be copyrightable
6. [Kubebuilder Boilerplate Docs](https://book-v1.book.kubebuilder.io/beyond_basics/boilerplate) -- Apache-2.0 scaffolding tool approach
7. [Azure Databricks Operator Issue #146](https://github.com/Azure/azure-databricks-operator/issues/146) -- Real-world license mismatch from scaffolding
8. [Terraform Scaffolding License Discussion](https://discuss.hashicorp.com/t/licence-for-provider-using-the-scaffholding/45822) -- Community confusion example
9. [FOSSA Apache 2.0 Analysis](https://fossa.com/blog/open-source-licenses-101-apache-license-2-0/) -- Permissive license scope
10. [TLDRLegal Apache 2.0 Summary](https://www.tldrlegal.com/license/apache-license-2-0-apache-2-0) -- Plain-English obligations
11. [Linux Foundation License Best Practices](https://www.linuxfoundation.org/licensebestpractices) -- SPDX identifier recommendations
12. [Create-React-App Template Structure](https://github.com/facebook/create-react-app/tree/main/packages/cra-template) -- MIT tool with unlicensed generated output
13. [Kubebuilder README Template Source](https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/golang/v4/scaffolds/internal/templates/readme.go) -- How kubebuilder injects license into generated files

---

*Report prepared 2026-05-13. This is legal analysis based on publicly available guidance, precedent, and industry practice -- not legal advice. For binding guidance, consult an attorney specializing in open source licensing.*
