<!-- Source: https://iambacon.co.uk/blog/the-pitfalls-of-using-a-global-author-identity-in-git -->
<!-- Retrieved: 2026-03-20 -->

# Git Global Author Identity Risks

## Key Problems Identified

**Mismatched identities across projects:** Developers with mixed work and personal projects frequently commit with the wrong identity. "I get miss matches all the time. I have personal projects and work projects all on the same machine and I invariably get it wrong."

**Public repository exposure:** A significant concern involves inadvertently associating an employer's name with publicly visible repositories, which may violate company policies regarding open-source contributions.

**Multiple machine inconsistencies:** When working across different devices, developers risk using varying identities for the same project, creating fragmented commit histories.

## How Wrong Identity Commits Occur

Global configuration applies automatically across all repositories. Without per-repository setup, developers may forget to configure identity and "Git will auto create the identity and still commit. So you could still end up committing with the wrong author identity."

## Recommended Solutions

Two approaches:

1. **Per-repository configuration:** Setting identity individually for each project (though this requires manual setup each time)

2. **Conditional configuration files:** Using multiple `.gitconfig` files organized by directory structure. Repositories can be separated into "work" and "personal" folders with corresponding config files conditionally included based on location.

The article emphasizes awareness: "be mindful of the author identity you are committing with especially for new repos or when working from a different machine."
