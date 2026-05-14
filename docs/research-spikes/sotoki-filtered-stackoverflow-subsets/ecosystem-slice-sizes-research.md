# Ecosystem Slice ZIM Size Estimates

## Overview

This report estimates realistic ZIM file sizes for tag-filtered Stack Overflow subsets organized by developer ecosystem. The goal is to determine whether per-ecosystem ZIMs are practical for local developer workstations, targeting <10 GB ideal and <20 GB acceptable.

All tag question counts were retrieved from the Stack Exchange API on 2026-05-14 and reflect cumulative totals across Stack Overflow's full history (~24M total questions).

---

## 1. Methodology

### 1.1 Tag Question Counts (Raw Data)

Question counts per tag were obtained directly from the SE API (`/2.3/tags/{tag}/info`). These counts represent the number of questions carrying each tag. Since SO questions carry 1-5 tags, a single question may be counted under multiple tags.

### 1.2 Tag Overlap Estimation

When combining multiple tags into an ecosystem group, naive addition double-counts questions that carry multiple tags from the group. For example, a question tagged both `python` and `django` would be counted twice.

**Overlap model used**: Framework/library tags almost always co-occur with their parent language tag on SO. Based on SO's tagging culture:

- **Language + framework overlap**: ~85-95% of framework-tagged questions also carry the parent language tag. A `django` question is almost always also tagged `python`. A `reactjs` question is almost always also tagged `javascript`.
- **Framework-to-framework overlap**: Minimal. Questions tagged `django` and `flask` simultaneously are rare (~1-2%).
- **Library overlap with language**: High but variable. `pandas` questions are ~90% also tagged `python`. `numpy` is ~80% also tagged `python` (some are tagged only `numpy` + `python-3.x` or similar).

**Estimation formula**: For a group with a dominant language tag L and framework/library tags F1..Fn:

```
Unique questions ≈ Count(L) + Σ Count(Fi) × (1 - overlap_with_L)
```

Where `overlap_with_L` is typically 0.85-0.95 for framework tags and 0.75-0.85 for library/tool tags.

For ecosystems without a single dominant language tag (DevOps), overlap between tools is lower (~10-20%), so:

```
Unique questions ≈ Σ Count(Ti) × (1 - pairwise_overlap_factor)
```

### 1.3 Questions-to-ZIM Size Calibration

We calibrate using known SE site ZIM sizes and their question counts:

| SE Site | Questions | Answers | ZIM Size | MB per 1K Questions | Source |
|---------|-----------|---------|----------|---------------------|--------|
| Ask Ubuntu | 425,281 | 539,512 | 2.6 GB | 6.1 | API verified |
| Unix & Linux | 245,877 | 365,706 | 1.2 GB | 4.9 | API verified |
| Server Fault | ~340,000 | ~460,000 | 1.5 GB | 4.4 | Estimated |
| Super User | ~530,000 | ~670,000 | 3.7 GB | 7.0 | Estimated |
| TeX | ~260,000 | ~360,000 | 4.2 GB | 16.2 | Estimated |
| Math | ~1,350,000 | ~1,800,000 | 6.9 GB | 5.1 | Estimated |
| Electronics | ~200,000 | ~280,000 | 3.8 GB | 19.0 | Estimated |
| Blender | ~130,000 | ~150,000 | 2.6 GB | 20.0 | Estimated |
| Full SO | ~24,000,000 | ~36,000,000 | 75 GB | 3.1 | Known |

**Key observations**:
- Sites with heavy image content (Electronics, Blender, TeX) have dramatically higher MB-per-question ratios (16-20 MB/1K questions) due to embedded images in posts.
- Text-heavy sites (Server Fault, Math, Full SO) are more efficient (3-5 MB/1K questions).
- Ask Ubuntu and Unix & Linux are good proxies for programming Q&A: ~6 MB per 1K questions.
- Full SO at scale achieves 3.1 MB/1K questions — likely due to ZIM compression being more effective at scale and SO questions being relatively short on average.

**Calibration ratio for SO tag subsets**: Programming Q&A on SO is mostly text with code blocks and occasional screenshots. We use **4-6 MB per 1K questions** as our primary estimate, acknowledging:
- Lower bound (~4 MB/1K): Text-heavy tags like `python`, `javascript`, `java` where most answers are code
- Upper bound (~6 MB/1K): Tags with more visual content (UI frameworks, CSS, mobile dev)
- Image-heavy outlier (~8-10 MB/1K): Tags like `css`, `android` with many screenshot-heavy posts

**Note on answers**: Each SO question averages ~1.5 answers. When we say "questions," the ZIM includes the question plus all its answers, comments, and user profiles. The calibration ratios already account for this.

### 1.4 ZIM Compression Context

The full SO dump is ~200 GB uncompressed XML. The ZIM (which renders to HTML with images) is 75 GB. ZIM uses LZMA compression internally. The XML-to-ZIM pipeline (sotoki) renders HTML, downloads/embeds user avatars and any linked images, and compresses everything into the ZIM format.

---

## 2. Per-Tag Question Counts

Source: SE API, retrieved 2026-05-14.

### Python Ecosystem
| Tag | Questions |
|-----|-----------|
| python | 2,220,306 |
| django | 312,308 |
| flask | 55,522 |
| fastapi | 7,650 |
| pandas | 289,705 |
| numpy | 115,536 |
| scipy | 21,945 |
| pytorch | 24,354 |
| tensorflow | 82,593 |
| python-3.x | 342,364 |
| python-2.7 | 94,600 |
| matplotlib | 73,077 |
| tkinter | 52,989 |
| pyspark | 41,165 |
| beautifulsoup | 32,926 |
| django-rest-framework | 31,859 |
| keras | 42,291 |

### JavaScript/TypeScript Ecosystem
| Tag | Questions |
|-----|-----------|
| javascript | 2,531,546 |
| typescript | 235,981 |
| reactjs | 478,691 |
| node.js | 472,798 |
| next.js | 43,339 |
| vue.js | 108,427 |
| angular | 307,800 |
| angularjs | 261,480 |
| express | 95,427 |
| svelte | 6,364 |
| sveltekit | 3,070 |
| jquery | 1,031,016 |
| react-native | 138,927 |
| react-hooks | 30,779 |
| npm | 50,219 |
| webpack | 42,514 |
| d3.js | 39,171 |
| nuxt.js | 13,098 |
| ecmascript-6 | 29,938 |
| redux | 35,393 |

### Rust Ecosystem
| Tag | Questions |
|-----|-----------|
| rust | 44,478 |
| rust-tokio | 1,295 |
| actix-web | 536 |
| cargo | 198 |

### Go Ecosystem
| Tag | Questions |
|-----|-----------|
| go | 75,041 |
| go-gin | 955 |
| gorilla | 614 |

### Java/Kotlin Ecosystem
| Tag | Questions |
|-----|-----------|
| java | 1,921,579 |
| kotlin | 98,851 |
| spring | 212,669 |
| spring-boot | 152,058 |
| spring-mvc | 58,475 |
| android | 1,418,485 |
| hibernate | 95,259 |
| maven | 89,358 |
| gradle | 53,320 |
| jpa | 52,245 |

### C#/.NET Ecosystem
| Tag | Questions |
|-----|-----------|
| c# | 1,626,519 |
| .net | 341,713 |
| asp.net | 373,611 |
| asp.net-mvc | 200,807 |
| asp.net-core | 85,754 |
| asp.net-web-api | 37,997 |
| blazor | 15,786 |
| unity-game-engine | 77,704 |
| entity-framework | 91,818 |
| wpf | 170,186 |
| winforms | 99,370 |
| vb.net | 140,160 |
| linq | 86,736 |
| xamarin | 50,742 |
| xamarin.forms | 34,622 |

### DevOps Ecosystem
| Tag | Questions |
|-----|-----------|
| docker | 140,453 |
| kubernetes | 58,249 |
| terraform | 20,395 |
| ansible | 23,047 |
| github-actions | 11,476 |
| continuous-integration | 14,121 |
| nginx | 54,918 |
| docker-compose | 32,623 |
| jenkins | 50,691 |
| amazon-web-services | 159,680 |
| azure | 146,191 |
| google-cloud-platform | 51,019 |
| azure-devops | 33,706 |
| aws-lambda | 32,171 |

---

## 3. Ecosystem Size Estimates

### 3.1 Python Ecosystem

**Scope**: python + django + flask + fastapi + pandas + numpy + scipy + pytorch + tensorflow

**Raw tag sum**: 3,130,000 (with significant double-counting)

**Overlap analysis**:
- `python` is the dominant tag at 2.22M questions
- `django` (312K): ~90% overlap with `python` → adds ~31K unique
- `flask` (56K): ~90% overlap → adds ~6K
- `fastapi` (8K): ~90% overlap → adds ~1K
- `pandas` (290K): ~90% overlap with `python` → adds ~29K
- `numpy` (116K): ~85% overlap → adds ~17K
- `scipy` (22K): ~85% overlap → adds ~3K
- `pytorch` (24K): ~80% overlap → adds ~5K
- `tensorflow` (83K): ~75% overlap (many tf questions use only tf+keras tags) → adds ~21K

**Estimated unique questions: ~2,335,000**

**Including answers**: ~2.33M questions × ~2.5 avg answers for python (popular tag, high answer rate) = ~5.8M answers. Total posts ~8.1M.

**ZIM size estimate**:
- At 4-5 MB/1K questions: **9.3 - 11.7 GB**
- Best estimate: **~10 GB**

**Verdict**: At the upper edge of "ideal" (<10 GB). Trimming to core web development (dropping ML/data-science tags) would bring it to ~2.3M questions / ~9.5 GB. A "Python core" slice (just python+django+flask+fastapi) of ~2.26M questions would be ~9 GB.

### 3.2 JavaScript/TypeScript Ecosystem

**Scope**: javascript + typescript + reactjs + node.js + next.js + vue.js + angular + express + svelte

**Raw tag sum**: 4,290,000 (massive double-counting)

**Overlap analysis**:
- `javascript` is dominant at 2.53M
- `typescript` (236K): ~60% overlap with `javascript` (TS has its own identity) → adds ~94K
- `reactjs` (479K): ~80% overlap with `javascript` → adds ~96K
- `node.js` (473K): ~75% overlap with `javascript` → adds ~118K
- `next.js` (43K): ~70% overlap with javascript, ~50% with reactjs → adds ~13K
- `vue.js` (108K): ~80% overlap with `javascript` → adds ~22K
- `angular` (308K): ~75% overlap with `javascript` → adds ~77K
- `express` (95K): ~85% overlap with `node.js`/`javascript` → adds ~14K
- `svelte` (6K): ~70% overlap → adds ~2K

**Estimated unique questions: ~2,970,000**

**ZIM size estimate**:
- At 4-5 MB/1K questions: **11.9 - 14.9 GB**
- Best estimate: **~13 GB**

**Verdict**: Acceptable (<20 GB) but exceeds ideal. The JS ecosystem is massive because `javascript` alone is the largest SO tag. A "modern JS/TS" slice excluding legacy jQuery/AngularJS questions and focusing on post-2018 content could reduce this significantly.

**Practical split option**: Split into "Frontend" (react + vue + angular + svelte + next.js + css) and "Backend" (node.js + express + typescript) subsets, each ~5-8 GB.

### 3.3 Rust Ecosystem

**Scope**: rust + rust-tokio + actix-web + cargo

**Raw tag sum**: 46,507

**Overlap analysis**: Minimal — these are small tags with high overlap to `rust`:
- `rust-tokio` (1,295): ~95% overlap → adds ~65
- `actix-web` (536): ~95% overlap → adds ~27
- `cargo` (198): ~95% overlap → adds ~10

**Estimated unique questions: ~44,600**

**ZIM size estimate**:
- At 5 MB/1K questions: **~223 MB**
- Best estimate: **~200-250 MB**

**Verdict**: Trivially small. This is comparable to a small SE site. Even including related tags like `async-await` (28K, ~10% overlap) and `webassembly` would keep it well under 500 MB. Rust developers get an extremely portable offline reference.

### 3.4 Go Ecosystem

**Scope**: go + go-gin + gorilla

**Raw tag sum**: 76,610

**Overlap analysis**:
- `go` dominant at 75K
- `go-gin` (955): ~90% overlap → adds ~96
- `gorilla` (614): ~85% overlap → adds ~92

**Estimated unique questions: ~75,200**

**ZIM size estimate**:
- At 5 MB/1K questions: **~376 MB**
- Best estimate: **~350-400 MB**

**Verdict**: Very small. Comparable to a medium SE site. Even adding `gorilla-mux`, `golang-migrate`, `protobuf` etc. would keep it under 500 MB.

### 3.5 Java/Kotlin Ecosystem

**Scope**: java + kotlin + spring + spring-boot + android

**Raw tag sum**: 3,903,000 (heavy double-counting)

**Overlap analysis**:
- `java` is dominant at 1.92M
- `kotlin` (99K): ~30% overlap with `java` (Kotlin has separate identity) → adds ~69K
- `spring` (213K): ~90% overlap with `java` → adds ~21K
- `spring-boot` (152K): ~90% overlap with `java`, ~60% overlap with `spring` → adds ~15K
- `android` (1.42M): ~55% overlap with `java` (many Android questions use only `android` tag, and post-2019 many are Kotlin) → adds ~639K

**Estimated unique questions: ~2,666,000**

**Note**: Android inflates this dramatically. Without Android:

**Java/Kotlin (no Android)**: ~2,025,000 questions → **~8-10 GB**
**Android-only**: ~1,000,000 unique questions → **~4-5 GB**
**Full Java/Kotlin + Android**: ~2,666,000 questions → **~11-13 GB**

**ZIM size estimate (full)**:
- At 4-5 MB/1K questions (android has screenshots → higher): **11-15 GB**
- Best estimate: **~13 GB**

**Verdict**: Acceptable as a whole, or split into "Java/Kotlin server-side" (~8 GB) and "Android" (~5 GB) for ideal sizes.

### 3.6 C#/.NET Ecosystem

**Scope**: c# + .net + asp.net + blazor + unity3d (unity-game-engine)

**Raw tag sum**: 2,444,000

**Overlap analysis**:
- `c#` dominant at 1.63M
- `.net` (342K): ~80% overlap with `c#` → adds ~68K
- `asp.net` (374K): ~85% overlap with `c#` → adds ~56K
- `asp.net-core` (86K): ~90% overlap with `c#` → adds ~9K
- `blazor` (16K): ~90% overlap with `c#` → adds ~2K
- `unity-game-engine` (78K): ~70% overlap with `c#` → adds ~23K
- `entity-framework` (92K): ~90% overlap → adds ~9K
- `wpf` (170K): ~90% overlap → adds ~17K

**Estimated unique questions: ~1,814,000**

**ZIM size estimate**:
- At 4-5 MB/1K questions: **7.3 - 9.1 GB**
- Best estimate: **~8 GB**

**Verdict**: Comfortably within the ideal range. The C#/.NET ecosystem is well-contained.

### 3.7 DevOps Ecosystem

**Scope**: docker + kubernetes + terraform + ansible + github-actions + ci-cd + nginx

**Raw tag sum**: 323,000

**Overlap analysis**: DevOps tags have LOW inter-tag overlap (questions are typically about one specific tool):
- `docker` (140K): standalone dominant tag
- `kubernetes` (58K): ~15% overlap with `docker` → adds ~49K
- `terraform` (20K): ~5% overlap with others → adds ~19K
- `ansible` (23K): ~5% overlap → adds ~22K
- `github-actions` (11K): ~5% overlap → adds ~10K
- `continuous-integration` (14K): ~20% overlap with jenkins/github-actions → adds ~11K
- `nginx` (55K): ~10% overlap with docker → adds ~50K
- `docker-compose` (33K): ~70% overlap with `docker` → adds ~10K
- `jenkins` (51K): ~15% overlap with CI → adds ~43K

**Estimated unique questions: ~354,000** (including docker-compose and jenkins)

**ZIM size estimate**:
- At 5-6 MB/1K questions: **1.8 - 2.1 GB**
- Best estimate: **~2 GB**

**Verdict**: Very comfortable. Even adding AWS/Azure/GCP tags would keep it under 5 GB.

---

## 4. Summary Table

| Ecosystem | Unique Questions (est.) | ZIM Size (est.) | Fits Workstation? |
|-----------|------------------------|-----------------|-------------------|
| **Python** (full) | ~2,335,000 | ~10 GB | Acceptable |
| **Python** (web only) | ~2,260,000 | ~9 GB | Ideal |
| **JS/TS** (full) | ~2,970,000 | ~13 GB | Acceptable |
| **JS/TS** (modern only) | ~1,800,000 | ~8 GB | Ideal |
| **Rust** | ~44,600 | ~250 MB | Trivial |
| **Go** | ~75,200 | ~400 MB | Trivial |
| **Java/Kotlin** (no Android) | ~2,025,000 | ~9 GB | Ideal |
| **Java/Kotlin + Android** | ~2,666,000 | ~13 GB | Acceptable |
| **C#/.NET** | ~1,814,000 | ~8 GB | Ideal |
| **DevOps** (core tools) | ~354,000 | ~2 GB | Trivial |
| **DevOps** (+ cloud providers) | ~600,000 | ~3 GB | Comfortable |

### Comparison to Full SO

| Metric | Full SO | Python Slice | DevOps Slice |
|--------|---------|-------------|--------------|
| Questions | 24,000,000 | 2,335,000 (~10%) | 354,000 (~1.5%) |
| ZIM Size | 75 GB | ~10 GB | ~2 GB |
| Practical? | No (workstation) | Yes | Yes |

---

## 5. Calibration Against Existing SE Site ZIMs

To validate our estimates, we compare tag-filtered SO subsets to similarly-sized standalone SE sites:

| Comparison | Questions | ZIM Size | Notes |
|-----------|-----------|----------|-------|
| **Rust SO subset** | ~45K | ~250 MB | Similar to haskell tag (~52K) |
| Haskell (est. from Unix SE ratio) | ~52K | ~300 MB | Haskell is text-heavy |
| **Go SO subset** | ~75K | ~400 MB | Between small-medium SE sites |
| **DevOps SO subset** | ~354K | ~2 GB | Similar to Ask Ubuntu (425K → 2.6 GB) |
| Ask Ubuntu | 425K | 2.6 GB | Good proxy — similar content type |
| Server Fault | ~340K | 1.5 GB | Sysadmin Q&A, less visual content |
| **Python SO subset** | ~2.3M | ~10 GB | Between Math SE and full SO |
| Math SE | ~1.35M | 6.9 GB | MathJax rendering inflates size |

The DevOps estimate (~2 GB for ~354K questions) aligns well with Ask Ubuntu (2.6 GB for 425K questions). Server Fault at 1.5 GB for ~340K questions suggests our estimate might even be slightly high, which is conservative and appropriate.

---

## 6. Critical Caveats

### 6.1 Images Are the Wildcard

Images are the single biggest variable in ZIM file sizes. Sites with heavy image content (Electronics: 19 MB/1K questions, Blender: 20 MB/1K questions) are 3-5x larger per question than text-heavy sites.

For SO tag subsets:
- **Low image density**: python, java, go, rust, devops (mostly code in answers) → 4-5 MB/1K
- **Medium image density**: reactjs, angular, css, android (UI screenshots) → 6-8 MB/1K
- **High image density**: unity-game-engine, flutter (visual output) → 8-12 MB/1K

**Mitigation**: Sotoki has a `--no-images` flag (or can be configured to skip image downloads). A no-image ZIM would be roughly 40-60% smaller, bringing even the largest ecosystem slices under 8 GB.

### 6.2 Answers Inflate Size Non-Linearly

Popular tags (python, javascript) have more answers per question and longer answers than niche tags. A python question averages ~2.5 answers vs ~1.2 for a niche tag. This means the effective content per "question" is ~2x higher for popular tags.

Our calibration against SE sites (which have similar answer rates to SO) accounts for this, but it means you cannot simply scale linearly from niche-tag estimates to popular-tag estimates.

### 6.3 PostHistory.xml Is Not in ZIMs

ZIM files contain rendered HTML content, not edit histories. The PostHistory.xml file (~67 GB for full SO) is consumed during sotoki processing but is NOT stored in the output ZIM. This is why the ZIM (75 GB) is much smaller than the raw XML dump (~200 GB).

### 6.4 Tag Boundary Is Fuzzy

A question tagged `python` + `django` + `postgresql` would appear in both a "Python" ZIM and a hypothetical "Database" ZIM. This is by design — the question is relevant to both ecosystems. But it means ZIM sizes are not additive across ecosystems.

### 6.5 Temporal Filtering Could Help

Many SO questions are outdated (Python 2.x, AngularJS 1.x, jQuery-era patterns). Filtering to questions from 2018+ would reduce volumes by ~40-50% for mature tags like `javascript` and `java`, with minimal loss of practical value. This would bring every ecosystem comfortably under 8 GB.

### 6.6 The Full SO ZIM Hasn't Been Built Since 2023

The Kiwix project's full SO ZIM (75 GB, Nov 2023) hasn't been updated in over 2.5 years. This is itself evidence that full-SO ZIMs are impractical to produce and distribute. Tag-filtered subsets solve this problem by being small enough to build and serve regularly.

---

## 7. Assessment: Is "2-3 GB Per Ecosystem" Realistic?

**For small/medium ecosystems (Rust, Go, DevOps): Yes, absolutely.** These come in well under 2 GB even with generous tag inclusion.

**For large ecosystems (Python, JS/TS, Java, C#): No, not without additional filtering.** The major language ecosystems produce 8-13 GB ZIMs because the parent language tag alone contains 1.6-2.5M questions.

**Strategies to reach 2-3 GB for large ecosystems:**

1. **Temporal filter**: Questions from 2020+ only → ~40-50% reduction → Python drops to ~5-6 GB
2. **Score filter**: Questions with score >= 1 only → ~30-40% reduction (removes unanswered/low-quality)
3. **Combined temporal + score**: → ~60% reduction → Python drops to ~4 GB
4. **No images**: → additional 40-60% reduction → Python drops to ~2-3 GB
5. **Framework-only slices**: "Python Web" (django+flask+fastapi only, without the parent `python` tag) → ~350K questions → ~1.8 GB

The most practical approach for gdev would be:
- **Small ecosystems**: Ship full tag-filtered ZIMs (Rust: 250 MB, Go: 400 MB, DevOps: 2 GB)
- **Large ecosystems**: Apply temporal + quality filters to keep ZIMs under ~5 GB, or offer tiered options ("Python essentials" at 3 GB vs "Python complete" at 10 GB)
- **All ecosystems**: Offer a `--no-images` build option for size-constrained environments

---

## 8. Conclusions

1. **Tag-filtered ZIMs are viable for all ecosystems.** Even the largest (JS/TS at ~13 GB) fits on a developer workstation.

2. **Small ecosystems are trivial.** Rust (250 MB), Go (400 MB), and DevOps (2 GB) are smaller than many individual SE site ZIMs.

3. **Large ecosystems benefit from secondary filtering.** Temporal (post-2018) and quality (score >= 1) filters can halve ZIM sizes without significant utility loss.

4. **The "2-3 GB per ecosystem" target is achievable** for small/medium ecosystems natively, and for large ecosystems with temporal+quality filtering or no-images builds.

5. **Images are the dominant size variable.** A `--no-images` option provides an easy 40-60% size reduction.

6. **Tag-filtered subsets solve the "full SO is too big" problem** that has left the Kiwix SO ZIM stale since November 2023. A 2-10 GB ecosystem ZIM can be rebuilt quarterly with modest infrastructure.
