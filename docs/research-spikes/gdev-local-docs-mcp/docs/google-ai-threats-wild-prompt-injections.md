# AI Threats in the Wild: Current State of Prompt Injections on the Web

- **Source**: https://security.googleblog.com/2026/04/ai-threats-in-wild-current-state-of.html (redirected to https://blog.google/security/prompt-injections-web/)
- **Retrieved**: 2026-05-14
- **Publisher**: Google Security Blog

---

## Overview
Google's Threat Intelligence teams conducted a comprehensive web scan to assess real-world indirect prompt injection (IPI) exploitation. The research reveals that while attackers are experimenting with these techniques, current sophistication remains limited -- though activity is accelerating.

## Key Statistics
- **32% increase** in malicious IPI detections between November 2025 and February 2026
- Analysis drew from Common Crawl's repository of 2-3 billion monthly webpage snapshots
- Most detections proved benign (false positives), primarily educational or research content

## Attack Categories Identified

**Harmless Pranks**: Invisible injections altering AI conversational tone or behavior for entertainment purposes.

**Helpful Guidance**: Website authors embedding instructions to improve AI summaries, though these could easily weaponize misinformation delivery.

**SEO Manipulation**: Attempts to bias AI recommendations toward specific businesses, ranging from basic to highly sophisticated automated implementations.

**AI Deterrence**: Instructions preventing AI crawling, including resource-wasting redirects to infinite-loading content.

**Data Exfiltration**: Low-sophistication theft attempts, with researchers noting "advanced exfiltration prompts" haven't been productionized at scale.

**Destructive Attacks**: Commands attempting file deletion or system vandalism, considered unlikely to succeed given current AI constraints.

## Detection Methodology
Google employed a three-stage approach:
1. Pattern matching for known injection signatures
2. Gemini-based classification for intent assessment
3. Human validation ensuring high-confidence findings

## Critical Insights

Current threat actors demonstrate "limited sophistication," yet the 32% quarterly increase signals escalating interest. Google attributes future growth risks to two converging factors: increasingly capable AI systems (valuable targets) and automated agentic AI reducing attack costs.

## Google's Response
- Red team pressure-testing of Gemini systems
- AI Vulnerability Reward Program for external researchers
- Real-time threat detection across global-scale data
- Layered defense strategies documented in companion resources
