# Sotoki README
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/README.md
- **Retrieved**: 2026-05-14

---

# Sotoki — Stack Overflow to Kiwix

An openZIM scraper to create offline versions of Stack Exchange websites such as Stack Overflow.

Based on Stack Exchange's Data Dumps hosted by The Internet Archive.

## Usage

```
sotoki --help
```

Users must specify:
- A mirror URL (--mirror)
- Domain name (--domain)
- ZIM title (--title, required)
- ZIM description (--description, required)

## Example

```
sotoki --domain sports.stackexchange.com \
       --mirror https://archive.org/download/stackexchange_20230101 \
       --title "Sports StackExchange" \
       --description "Sports Q&A from StackExchange"
```

## Installation

Available via Docker and pip (Python virtual environment).

## Notes

- Pre-built ZIM files available at library.kiwix.org
- See CONTRIBUTING.md for developer setup
- No tag filtering, content selection, or advanced filtering options documented in README
