# The Curse of NixOS
- **Source URL**: https://blog.wesleyac.com/posts/the-curse-of-nixos
- **Retrieved**: 2026-03-20
- **Type**: Blog post

## Author
Wesley Aptekar-Cassels

## Duration
Approximately three years as sole operating system on their laptop

## Use Case
Primary laptop operating system (neovim editor user)

## Key Complaints and Pain Points

### Programming Language Issues
- Created proprietary configuration language that is "not very good and is extremely difficult to learn"
- Most users "simply copy/paste example configurations" without understanding the underlying language
- Poor documentation connecting language learning to practical NixOS usage
- Syntax is described as ugly, though the author acknowledges this may be unfixable

### Isolation and Compatibility Problems
- Software must know it runs from the Nix store; cannot provide "real isolation"
- All packages require recompilation for NixOS compatibility
- "All software needs to be recompiled to work on NixOS, often with some terrifying hacks involved"
- Dependency detection relies on grepping for `/nix/store/` paths — admittedly inadequate
- Binaries linking to standard library paths won't function without patching

### Additional Issues
- Configuration exhibits "spooky action-at-a-distance" effects
- Patching packages is theoretically simple but practically difficult

## Outcome
**Stayed with NixOS** despite criticisms. The author states: "I'm going to keep using it, since I can't stand anything else after having a taste of NixOS"

## Alternatives Considered
None explicitly named, though the author "rooted for something new" incorporating NixOS's dependency management philosophy without its flaws.
