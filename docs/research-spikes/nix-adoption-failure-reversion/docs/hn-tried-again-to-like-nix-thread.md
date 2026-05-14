# HN Thread: "I've tried again and again to like Nix, but at this point I have to throw in the towel"
- **Source URL**: https://news.ycombinator.com/item?id=39723701
- **Retrieved**: 2026-03-20
- **Type**: Hacker News comment thread

## Original Post (kstenerud)
"I've tried again and again to like Nix, but at this point I have to throw in the towel."

**Main criticisms:**
- Broke both systems requiring reinstall despite idempotency claims
- Documentation is "complete and technically accurate, but maddeningly obtuse"
- Stepping beyond the standard path creates "strange results and absolutely bizarre errors"
- Compared to alchemy rather than science; requires years of experiential knowledge
- Docker's strength is flexibility; Nix demands deep arcane knowledge

## Notable Responses

### janjongboom - Created StableBuild alternative
- Went "down the same path" seeking deterministic builds
- Found Docker's problem: packages update unpredictably over time
- With 40+ containers, "fixing containers is a significant part of your job"
- Founded stablebuild.com offering "deterministic builds w/ Docker" using immutable mirrors of Ubuntu/Debian/Alpine packages

### fransje26
- "gave up on Nix about 5 years ago because of" documentation issues
- Notes "not much has changed on that front since then"

### jokethrowaway
- NixOS user switching back to Arch
- "rebuilding everything at every update take forever"
- Prefers "binary bleeding edge packages and AUR"

### mtmk
- Struggled with .NET 8 AOT deployment on Nix
- "spiraled into a plethora of issues, ultimately forcing me to back down"
- Concluded "you need solid understanding before reasonably productive"

### laerus
- Recommends Fedora Atomic/Kinoite as alternative
- Claims "rpm-ostree will obsolete Nix"
