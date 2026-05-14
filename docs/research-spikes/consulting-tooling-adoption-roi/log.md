# Research Log: Consulting Tooling Adoption ROI

## 2026-03-20 12:00 — Spike Created
- **Type**: decision
- **Status**: success
- **Depth**: surface
- **Summary**: Spike initialized. Goal is to build a dollar-cost quantification framework for dev environment tooling adoption in consulting contexts. Needs to close gaps: OQ-NIX-LL-5 ("$X per new hire"), unsourced "50-75% CI build time reduction", constructed "20-45 minute" onboarding estimate, and missing cost-benefit framework. Source: `synthesized-reports/cop-research-gap-analysis.md` § Spike #6.
- **Next**: Define research question and create Phase 1 tasks.

## 2026-03-20 14:30 — Consultant Billing Rates & Utilization Benchmarks Research
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Consultancy.org](https://www.consultancy.org/consulting-industry/fees-rates) → `docs/consultancy-org-fees-rates.md`
  - [Consulting.us](https://www.consulting.us/consulting-industry/fees-rates) → `docs/consulting-us-fees-rates.md`
  - [Mosaic Utilization Metrics](https://www.mosaicapp.com/post/the-utilization-metrics-every-consulting-firm-should-track) → `docs/mosaicapp-utilization-metrics.md`
  - [EVX Utilization Benchmarks](https://www.evxsoftware.com/blog/utilization-in-consulting-how-to-measure-improve-and-optimize-consultant-utilization-rates) → `docs/evx-consultant-utilization-benchmarks.md`
  - [Mosaic Utilization Statistics](https://www.mosaicapp.com/post/billable-utilization-rate-statistics-in-professional-services-firms) → `docs/mosaicapp-utilization-rate-statistics.md`
  - [Slideworks McKinsey Fees](https://slideworks.io/resources/management-consulting-fees-how-mc-kinsey-prices-projects) → `docs/slideworks-mckinsey-consulting-fees.md`
  - [Modernization Intel Rates 2026](https://softwaremodernizationservices.com/insights/application-modernization-consulting-rates/) → `docs/modernization-intel-consulting-rates-2026.md`
  - [MOR Software IT Rates 2025](https://morsoftware.com/blog/it-consulting-rates) → `docs/morsoftware-it-consulting-rates-2025.md`
  - [Timetta Profitability Model](https://timetta.com/blog/profitability-in-consulting-and-professional-services-simple-financial-model) → `docs/timetta-consulting-profitability-model.md`
  - [Scoro Billable Rates](https://www.scoro.com/blog/billable-rate/) → `docs/scoro-billable-rates-guide.md`
  - [SPI Research 2025 Benchmark](https://spiresearch.com/reports/2025-ps-maturity-benchmark/) → `docs/spi-research-2025-ps-benchmark.md`
  - [Salary-to-Billing Multiplier](multiple sources) → `docs/salary-to-billing-rate-multiplier-research.md`
  - [IT Consulting Costs](https://financialmodelslab.com/blogs/operating-costs/it-consulting-services) → `docs/financialmodelslab-it-consulting-costs.md`
- **Summary**: Completed comprehensive research on billing rates by seniority (junior $50-120, mid $80-180, senior $130-250, principal $200-400/hr for IT consulting), utilization benchmarks (68.9% actual vs 75% target per SPI 2025), cost of non-billable time ($773/day direct + $2,000/day opportunity = $2,773/day), loaded cost multiplier (1.99x salary per Deltek; 3x rule for billing rate), and firm tier comparisons. Established blended rate of $175/hr for ROI modeling. Key insight: 30 min/day saved per developer = $19,250/year value at blended rate, or $385K/year for 20-person team.
- **Next**: Integrate findings into ROI framework synthesis task. These rates serve as denominator for all tooling ROI calculations.

## 2026-03-20 — Credential & Security Incident Costs Research
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [IBM Cost of Data Breach 2025](https://www.bakerdonelson.com/ten-key-insights-from-ibms-cost-of-a-data-breach-report-2025) → `docs/ibm-cost-of-data-breach-2025.md`
  - [Verizon DBIR 2025 Credentials](https://www.descope.com/blog/post/dbir-2025) → `docs/verizon-dbir-2025-credentials.md`
  - [Accenture Cloud Leak](https://www.upguard.com/breaches/cloud-leak-accenture) → `docs/accenture-cloud-leak-case-study.md`
  - [Deloitte Breach 2017](https://krebsonsecurity.com/2017/09/source-deloitte-breach-affected-all-company-email-admin-accounts/) → `docs/deloitte-breach-2017-krebs.md`
  - [Deloitte Breach 2025](https://nhimg.org/the-story-behind-deloitte-2025-breach) → `docs/deloitte-2025-breach.md`
  - [Small Business Breach Costs](https://purplesec.us/learn/data-breach-cost-for-small-businesses/) → `docs/small-business-breach-costs-purplesec.md`
  - [Insider Threat Statistics 2025](https://deepstrike.io/blog/insider-threat-statistics-2025) → `docs/insider-threat-statistics-2025.md`
  - [Cyber Insurance Costs](https://www.embroker.com/blog/cyber-insurance-cost/) → `docs/cyber-insurance-costs-embroker.md`
  - [Cyber Insurance TechInsurance](https://www.techinsurance.com/technology-business-insurance/cybersecurity/cost) → `docs/cyber-insurance-costs-techinsurance.md`
  - [MSA Liability](https://www.loeb.com/en/insights/publications/2024/07/navigating-service-provider-liability-in-managed-security-services-agreements) → `docs/msa-liability-managed-services-loeb.md`
  - [GDPR Fines](https://www.cookieyes.com/blog/gdpr-fines/) → `docs/gdpr-fines-overview-cookieyes.md`
  - [AWS Misconfiguration Costs](https://shardsecure.com/blog/real-cost-aws-misconfiguration) → `docs/aws-misconfiguration-costs-shardsecure.md`
  - [Cybersecurity ROI Framework](https://safe.security/resources/blog/measuring-cybersecurity-roi-a-framework-for-2026-decision-makers/) → `docs/cybersecurity-roi-framework-safe.md`
  - [Git Identity Risks](https://iambacon.co.uk/blog/the-pitfalls-of-using-a-global-author-identity-in-git) → `docs/git-global-identity-risks-iambacon.md`
- **Summary**: Comprehensive research on credential and security incident costs in consulting contexts. Established that breaches cost $120K-$10.2M depending on scale, stolen credentials are the #1 attack vector (22% of breaches), negligent insider incidents average $676K each with 13.5/year frequency, and consulting firms face amplified risk from multi-client credential juggling. Built ALE-based ROI model showing QubesOS isolation pays for itself in ~11 months with 668% annual ROSI thereafter. Documented Accenture and Deloitte credential-related breaches as case studies. Analyzed MSA liability structures, regulatory penalties (GDPR/HIPAA/PCI/CCPA), and cyber insurance costs ($2,500-$6,000/year for IT consultants).
- **Next**: Integrate credential risk costs into overall ROI framework synthesis task.

## 2026-03-20 15:00 — Developer Onboarding Cost Models Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Cortex 2024 State of Developer Productivity](https://www.cortex.io/report/the-2024-state-of-developer-productivity) → `docs/cortex-2024-state-developer-productivity.md`
  - [80 Employee Onboarding Statistics 2025](https://www.newployee.com/blog/employee-onboarding-statistics) → `docs/newployee-80-onboarding-statistics-2025.md`
  - [Staff Augmentation Onboarding Case Study](https://fullscale.io/blog/staff-augmentation-onboarding-timeline/) → `docs/fullscale-staff-augmentation-onboarding-case-study.md`
  - [GitLab: Accelerate Developer Onboarding](https://about.gitlab.com/the-source/platform/how-to-accelerate-developer-onboarding-and-why-it-matters/) → `docs/gitlab-accelerate-developer-onboarding.md`
  - [HackerNoon: Engineer Onboarding Ramp-Up Time](https://hackernoon.com/engineer-onboarding-the-ugly-truth-about-ramp-up-time-7e323t9j) → `docs/hackernoon-engineer-onboarding-ramp-up-time.md`
  - [Shopify Developer Onboarding](https://shopify.engineering/developer-onboarding-at-shopify) → `docs/shopify-developer-onboarding.md`
  - [Projectworks: Bench Time Costs](https://www.projectworks.com/blog/how-expensive-are-my-unassigned-consultants) → `docs/projectworks-bench-time-costs.md`
  - [BenchBee: IT Consultancy Bench Time Costs](https://benchbee.io/blog/5-ways-your-it-consultancy-is-losing-money-on-bench-time/) → `docs/benchbee-it-consultancy-bench-time-costs.md`
  - [ARDURA IT Recruitment Cost Calculator](https://ardura.consulting/blog/it-recruitment-cost-calculator-2026-true-cost-of-bad-hire/) → `docs/ardura-it-recruitment-cost-calculator.md`
  - [OneUpTime: Platform Eng Onboarding Tracking](https://oneuptime.com/blog/post/2026-01-30-platform-eng-onboarding-time/view) → `docs/oneuptime-platform-eng-onboarding-time-tracking.md`
  - [Valorem Reply: Developer Onboarding Framework](https://www.valoremreply.com/resources/insights/blog/azure/developer-onboarding-cut-your-ramp-time-in-half-with-this-framework/) → `docs/valorem-reply-developer-onboarding-framework.md`
  - [Devlin Peck: Onboarding Statistics](https://www.devlinpeck.com/content/employee-onboarding-statistics) → `docs/devlinpeck-onboarding-statistics.md`
  - [Spotify Backstage Onboarding Metrics](https://blog.container-solutions.com/how-developer-experience-portal-backstage-solved-spotifys-complexity) → `docs/spotify-backstage-onboarding-metrics.md`
- **Summary**: Comprehensive research on developer onboarding costs and time-to-productivity. Industry data shows 3-9 month ramp to full productivity, 25-40% efficiency in month 1, $7,500-$28,000 per onboarding event. Consulting firms face a unique multiplier: 3-4 project onboardings/year at billing rates ($1,500-$3,500/day), making environment setup automation worth $12,000-$70,000/consultant/year. Environment setup (2-5 days manual) is the most mechanically reducible component. Spotify achieved 67% reduction in time-to-10th-PR via Backstage. Shopify reduced tool ramp-up from 1 month to 1 week. Report includes productivity ramp curve, cost models, consulting utilization economics, and documented case studies. Limitations noted: small sample sizes, vendor bias in some sources, productivity ramp curve is folk wisdom rather than rigorous primary research.
- **Next**: Use these findings in ROI framework synthesis. Cross-reference with billing rate/utilization research and environment management overhead research.

## 2026-03-20 — Environment Management Overhead & Tooling Friction Research
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Stripe Developer Coefficient 2018](https://stripe.com/files/reports/the-developer-coefficient.pdf) → `docs/stripe-developer-coefficient-2018.md`, `docs/stripe-developer-coefficient-detailed.md`
  - [Microsoft Time Warp Study 2024](https://arxiv.org/html/2502.15287) → `docs/microsoft-time-warp-study-2024.md`
  - [Atlassian DX 2024](https://www.atlassian.com/software/compass/resources/state-of-developer-2024) → `docs/atlassian-developer-experience-2024-detailed.md`
  - [Atlassian DX 2025](https://www.atlassian.com/teams/software-development/state-of-developer-experience-2025) → `docs/atlassian-developer-experience-2025.md`
  - [GitLab DevSecOps 2025](https://about.gitlab.com/press/releases/2025-11-10-gitlab-survey-reveals-the-ai-paradox/) → `docs/gitlab-2025-devsecops-survey.md`
  - [Retool Internal Tools 2023](https://retool.com/blog/state-of-internal-tools-2023) → `docs/retool-state-of-internal-tools-2023.md`
  - [Context Switching Research](multiple academic sources) → `docs/context-switching-research-compilation.md`
  - [DX Newsletter — Actual vs Ideal Workweek](https://newsletter.getdx.com/p/developer-ideal-and-actual-workdays) → `docs/dx-newsletter-actual-vs-ideal-workweek.md`
  - [Spotify Backstage Metrics](https://backstage.spotify.com/) → `docs/spotify-backstage-productivity-metrics.md`
  - [DORA 2024 State of DevOps](https://dora.dev/research/2024/dora-report/) → `docs/dora-2024-state-of-devops.md`
  - [Platform Engineering Onboarding Case Study](https://platformengineering.com/) → `docs/platform-engineering-onboarding-case-study.md`
  - [GitHub Enterprise ROI](https://github.blog/) → `docs/github-enterprise-onboarding-roi.md`
  - [Coralogix Developer Time](https://coralogix.com/) → `docs/coralogix-developer-time-debugging.md`
  - [Coder Works on My Machine](https://coder.com/) → `docs/coder-works-on-my-machine.md`
  - [DEV.to Hidden Cost](https://dev.to/) → `docs/dev-to-hidden-cost-works-on-my-machine.md`
  - [Valorem Reply Onboarding](https://www.valoremreply.com/) → `docs/valorem-reply-developer-onboarding.md`
- **Summary**: Comprehensive research on developer environment management overhead. Key findings: developers lose 5-10 hrs/week to environment-related friction (subset of 8-17 hrs/week total maintenance). Microsoft regression analysis proves environment work has statistically significant negative impact on productivity AND satisfaction. Atlassian: 69% of devs lose 8+ hrs/week to inefficiencies. GitLab: 7 hrs/week lost to tooling fragmentation. Environment setup is #2 automation priority (27% of devs). Consulting multiplier estimated at 1.5-2x due to multi-project switching. At $175/hr blended rate, environment overhead = $39K-78K/year per developer. Full report at `environment-overhead-research.md`.
- **Next**: Integrate into ROI framework synthesis with billing rates and onboarding data.

## 2026-03-20 16:00 — CI/CD Build Time Benchmarks Research Complete
- **Type**: research
- **Status**: success
- **Depth**: deep
- **Sources**:
  - [Garnix CI Benchmarks](https://garnix-io.github.io/benchmarks/) → `docs/garnix-nix-ci-benchmarks.md`
  - [Garnix Benchmarks Discourse](https://discourse.nixos.org/t/nix-ci-benchmarks/71086) → `docs/garnix-benchmarks-discourse-discussion.md`
  - [Nix Binary Cache Tools Comparison](https://zenn.dev/trifolium/articles/1a2eeca4775e56) → `docs/nix-binary-cache-tools-comparison-github-actions.md`
  - [Nix-Based CI](https://compilersaysno.com/posts/nix-based-continuous-integration/) → `docs/nix-based-continuous-integration-compilersaysno.md`
  - [Magic Nix Cache](https://determinate.systems/blog/magic-nix-cache/) → `docs/magic-nix-cache-determinate-systems.md`
  - [Adopting Nix](https://dennybritz.com/posts/adopting-nix/) → `docs/adopting-nix-denny-britz.md`
  - [Ryan Rasti 3-Year Production](https://ryanrasti.com/blog/why-nix-will-win/) → `docs/ryan-rasti-why-nix-will-win-3-year-production.md`
  - [Fast CI with Nix](https://quentin.dufour.io/blog/2024-08-10/fast-ci-build-with-nix/) → `docs/fast-ci-build-with-nix-quentin-dufour.md`
  - [Nix Package Caches](https://www.jetify.com/blog/dont-rebuild-yourself-an-intro-to-nix-package-caches) → `docs/jetify-nix-package-caches-intro.md`
  - [Actuated Nix Builds](https://actuated.com/blog/faster-nix-builds) → `docs/actuated-faster-nix-builds-github-actions.md`
  - [NixOS Evaluation Times](https://discourse.nixos.org/t/a-look-at-nixos-nixpkgs-evaluation-times-over-the-years/65114) → `docs/nixos-nixpkgs-evaluation-times-discourse.md`
  - [Shopify Faster CI](https://shopify.engineering/faster-shopify-ci) → `docs/shopify-faster-ci-engineering-blog.md`
  - [Docker Layer Caching](https://grahamc.com/blog/nix-and-layered-docker-images/) → `docs/nix-layered-docker-images-grahamc.md`
  - [Numtide Nix Docker](https://numtide.com/blog/nix-docker-or-both/) → `docs/numtide-nix-docker-or-both.md`
  - [Avoid Nix Docker Images](https://www.mccurdyc.dev/posts/2024/09/why-i-avoid-using-nix-to-build-docker-images/) → `docs/why-avoid-nix-docker-images-mccurdyc.md`
  - [Docker Caching 70%](https://www.netdata.cloud/academy/docker-layer-caching/) → `docs/docker-layer-caching-ci-netdata.md`
  - HN discussion, Pinterest, Channable, Flox — via web search (full fetch blocked for some)
- **Summary**: The "50-75% CI build time reduction" claim is **unsubstantiated**. No controlled benchmark comparing Nix CI to conventional approaches exists. The claim traces to a single anecdotal data point (Ryan Rasti's 4-person Elixir/React team). Published benchmarks compare Nix CI tools against each other, not against Docker/conventional pipelines. Conventional caching strategies (Docker layer caching, GitHub Actions cache) claim comparable 40-80% improvements. Nix's caching model has theoretical structural advantages (content-addressed, DAG-based) but also introduces overhead (evaluation time, cold builds, store locking, complexity). Detailed report with evidence table written to `ci-cd-benchmarks-research.md`.
- **Next**: Integrate findings into ROI framework. Flag the 50-75% claim as unsubstantiated in any presentation materials.

## 2026-03-20 17:00 — ROI Framework Synthesis Complete
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Synthesized all five research reports (73+ sources) into comprehensive ROI framework at `roi-framework-synthesis.md`. Four ROI categories modeled: (1) onboarding cost reduction — $8K-$17K per new hire, $166K-$887K/year for 20-person team; (2) environment overhead reduction — $320K-$1.25M/year; (3) QubesOS security risk reduction — $38K/year, 668% ROSI, 11-month payback; (4) CI/CD — honestly assessed as unsubstantiated, recommended retiring the "50-75%" claim. Combined framework: $524K-$1.18M/year conservative-to-moderate value against $119K-$144K first-year investment = 264-889% Year 1 ROI with ~3 month payback. Includes presentation-ready talking points for CXO and engineering manager audiences, confidence assessment by evidence tier, and claims to avoid. Updated research.md with conclusions. All four gaps from gap analysis closed: OQ-NIX-LL-5 answered ("$8K-$17K per new hire"), CI claim flagged, onboarding estimate contextualized, framework created.
- **Next**: Spike ready for completion. Consider running /complete-spike to finalize.

## 2026-03-20 17:30 — Spike Completed
- **Type**: analysis
- **Status**: success
- **Depth**: deep
- **Summary**: Spike finalized. All 6 tasks completed successfully. All 5 research reports pass the depth checklist (mechanisms, tradeoffs, alternatives, failure modes, examples, standalone readability). 73+ sources saved to docs/. Conclusions written in research.md with 7 key findings. Core deliverable: `roi-framework-synthesis.md` — a dollarized cost-benefit framework closing all four gaps from `cop-research-gap-analysis.md` § Spike #6. The framework provides $524K-$1.18M/year conservative-to-moderate value estimate for a 20-person consulting team, with presentation-ready talking points and honest confidence assessments. The "50-75% CI reduction" claim has been flagged as unsubstantiated with recommended alternative framing.
