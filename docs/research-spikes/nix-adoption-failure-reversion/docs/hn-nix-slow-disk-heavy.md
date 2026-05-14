# HN Thread: "I tried using Nix but stopped — very slow and extremely disk heavy"
- **Source URL**: https://news.ycombinator.com/item?id=42355376
- **Retrieved**: 2026-03-20
- **Type**: Hacker News comment thread

## Original Comment (brabel)
"I tried using Nix but stopped for two very practical reasons: it's very slow and it's extremely disk heavy."

The poster elaborated that installing packages caused the nix store to reach approximately 100 GB. They expressed frustration with update times consuming "a good part of an hour" and noted their 250GB laptop couldn't accommodate Nix's demands while apt worked fine.

## Key Replies

**dlahoda** suggested optimization strategies: using stable Nix, overriding nixpkgs inputs, applying offline flags after initial builds, implementing nixdirenv, and configuring garbage collection settings.

**microtonal** countered that Nix typically performs faster than competitors like apt/dnf/pacman. They noted their development VM reached 188GB due to multiple CUDA/Torch versions while running nixos-unstable, arguing that storage concerns are overblown given affordable SSD costs.

**robinsonb5** raised valid concerns about physical storage constraints in laptops and slow rural internet connections, emphasizing that "digital wastefulness is a problem."

**nh2** provided comparative data: a NixOS laptop closure measured 33GB versus Ubuntu's 27GB across one generation, while highlighting that Ubuntu version upgrades frequently required hours and caused configuration conflicts.
