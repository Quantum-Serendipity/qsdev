# We studied 100 dev tool landing pages — here's what really works in 2025
- **Source URL**: https://evilmartians.com/chronicles/we-studied-100-devtool-landing-pages-here-is-what-actually-works-in-2025
- **Retrieved**: 2026-05-15

## Overview
Evil Martians analyzed 100+ developer tool landing pages (Linear, Vercel, Supabase) to identify what works in 2025. Key finding: "No salesy BS. Clever and simple wins."

## General Layout Principles

**Two Core Rules:**
- Avoid flashy interactions; prioritize clean design with solid typography and breathing room
- Use centered layouts with max-width containers (most common approach)

Wide, edge-to-edge layouts are rarer but can look premium if executed well.

---

## Section-by-Section Breakdown

### 1. Hero Section

**Composition:** Centered layout dominates (headline + visual below). Side-by-side text/image layouts are exceptions.

**Main Visual Element Options:**
- **Animated product UI:** Shows movement and product breadth; requires more effort
- **Static product UI:** Faster to implement; still effective
- **Switchable multiple UIs:** Works when product has multiple use cases
- **Live product embed:** Power move for narrow-scope tools (image upscalers, etc.)
- **Code snippets:** Common for libraries, SDKs, infrastructure products
- **Abstract illustrations:** Used for products without UI, under-the-hood tools, or stealth projects

**Eyebrows & Banners:**
- Small text above titles highlighting releases, funding, or events
- Full-width banners for longer announcements

**Call-to-Action Pattern:**
- Primary CTA: Bold, specific language ("Start building," "Download now")
- Secondary CTA: Lighter styling (docs, waitlist, GitHub)

---

### 2. Trust Block

**Placement:** Immediately after hero section.

**B2B Approach:**
- Display recognizable customer logos
- Auto-scrolling carousels fit more logos without excessive vertical space
- Early-stage products: Short testimonials with client representative photos

**Individual User Approach:**
- Big numbers: GitHub stars, usage stats, awards
- Examples: "App Store Editor's Choice"
- User reviews for credibility

---

### 3. Feature Block

**Storytelling Approaches (ranked by effectiveness):**

1. **List of functions:** Weakest — disconnected from user needs
2. **Action-oriented stories:** Common but surface-level ("Build faster," "Run anywhere")
3. **Problem-oriented stories:** More engaging; shows how product solves pain points
4. **Bold statements:** Punchy, opinionated; works better for established products
5. **Mission statements:** Rare but powerful; presents broader vision

**Layout Formats:**

| Format | Best For |
|--------|----------|
| Full screenshots + short descriptions | UI-heavy tools |
| Chess layout (alternating image/text) | Features needing explanation |
| Text with icons | SDKs, libraries, many small features |
| Belts (scrollable card strips) | Integrations, capabilities |
| Bento blocks (grid layout) | Visual variety; condensing info |
| Tabbed features | Logical categories or multiple personas |
| Step-by-step | Onboarding flows, setup processes |
| Rich cards | Design-heavy, polished approach |
| Video tutorials | Demo/explainer shortcut |

**Supporting Content:**
- "How it works" sections: Explain "magic" (AI, automation, syncing)
- Usage examples: Show real outputs to inspire use cases
- Compatibility/integrations: Display logos of compatible services

---

### 4. Social Proof Block

**Key Pattern:** Curated testimonials (manually selected, not auto-pulled).

Why: "Guarantees only relevant and positive feedback is shown... reads better, too: no broken formatting, no off-topic noise."

**Design Elements:**
- Avatar, name, company logo (for B2B)
- Positioned near page bottom after product story
- Even one testimonial from early clients adds credibility

**Advanced Pattern:** Integrate quotes contextually with features rather than clustering as separate section.

---

### 5. Supporting Blocks (Optional)

**Comparison Table:**
- Rarely used; differentiates in competitive markets
- Side-by-side checkmarks and feature differences

**Pricing:**
- Most teams defer to separate page
- When included: Clean design, monthly/yearly toggles

**FAQ:**
- Accordion-style blocks near page end
- Practical questions about trials, data storage, login requirements

**Blog/Changelog Preview:**
- Only mature teams include this
- Signals active development; better kept on socials for early stage

---

### 6. Final CTA

**Requirements:**
- Big, loud, visually distinct
- Clearly separated background (colorful or dark-on-light)
- Short motivating message + single concise button

**Advanced Approach:** Embed calendar widget for instant meeting scheduling — particularly effective for early-stage startups with few leads.

---

## Key Recommendations

- Focus on user problems, not feature lists
- Keep interactions minimal; prioritize readability
- Use specific, action-oriented button copy
- Include social proof even with limited testimonials
- Make final CTA a safety net for scrolled visitors

## Resources

Evil Martians created an open-source template implementing these findings:
- Website: launchkit.evilmartians.io
- GitHub: evilmartians/devtool-template
- Webflow template available
