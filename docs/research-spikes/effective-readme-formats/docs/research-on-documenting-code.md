# What Research Tells Us About Documenting Code

- **Source URL**: https://idratherbewriting.com/learnapidoc/docapiscode_research_on_documenting_code.html
- **Retrieved**: 2026-05-15

---

## Academic Studies Examined

### 1. "When Not to Comment: Questions and Tradeoffs with API Documentation for C++ Projects"

**Authors:** Head, Sadowski, Murphy-Hill, and Knight (2018 ACM/IEEE Conference)

**Research Focus:** This study investigated where developers seek information — header files (formal documentation) versus implementation files (inline code comments) — and what content they need.

**Key Findings:**

- **Complexity Matters:** Simple code doesn't necessarily require documentation; developers prefer reading straightforward code directly rather than consulting docs.
- **Documentation's Role:** Complex code with intricate signatures or generated components demands formal documentation that developers actively consult.
- **Timing is Critical:** Documentation should be written during active development when developers retain detailed knowledge. Post-release documentation efforts suffer from developer disengagement and knowledge loss.
- **Content Priority:** Developers most frequently seek parameter information — data types, default values, constraints, and usage examples. The research emphasizes that "input values" receive the heaviest consultation.

### 2. "How Developers Use API Documentation: An Observation Study"

**Authors:** Meng, Steinhardt, and Shubert (2019, Communication Design Quarterly)

**Research Methodology:** Researchers observed developers solving predefined tasks using unfamiliar APIs while tracking documentation usage patterns.

**Key Behavioral Findings:**

- **Learning Styles Identified:** Developers employ three approaches:
  - **Systematic:** Read comprehensive overviews before attempting tasks
  - **Opportunistic:** Experiment first, consulting documentation only when stuck
  - **Pragmatic:** Blend both approaches flexibly based on problem complexity

- **Non-Linear Navigation:** Developers don't necessarily distinguish between documentation categories (concepts, reference, tutorials). They seek information based on problem domains rather than information type.

- **Time Allocation:** Users spend disproportionate time reviewing API reference documentation, particularly parameter descriptions.

## Practical Implications for Technical Writers

**Documentation Structure Recommendations:**

Rather than organizing by information type (separating concepts, reference, and tasks), structure documentation around functionality domains so related information clusters naturally around user problems.

**Supporting Experimentation:**

Make code immediately functional — users learn through trial-and-error with working examples, Swagger UI interfaces, and interactive API explorers rather than passive reading.

**Search and Navigation:**

Implement robust search and transparent navigation patterns to support non-linear exploration without forcing sequential reading.

**Information Redundancy:**

Repeat crucial parameter details across multiple sections rather than assuming single-reference documentation suffices.

**Reference Documentation Quality:**

Prioritize accuracy and completeness in parameter documentation since developers consult this section most frequently.
