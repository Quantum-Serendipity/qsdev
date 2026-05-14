# Sotoki archives.py Source Code Summary
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/archives.py
- **Retrieved**: 2026-05-14

---

## ArchiveManager Class

### Download Flow:
1. Downloads 7z archives from `{mirror}/{domain}.7z`
2. Uses wget (if available) or streaming download as fallback
3. Extracts to build directory

### Required XML Files:
The scraper requires these six XML files from SE dumps:
- Badges.xml
- Comments.xml
- PostLinks.xml
- Posts.xml
- Tags.xml
- Users.xml

### Post-Extraction Processing:
- Removes non-XML files and XML files not in the required set
- Merges user data with badges
- Merges posts with answers and comments into `posts_complete.xml`
- Counts total tags, users, and questions

### Significance for Tag Filtering:
The archive download is all-or-nothing — the entire domain dump is downloaded as a single 7z file. There is no way to download only specific tags' data from the SE dump. Tag filtering would need to happen AFTER extraction, during the XML processing phase.
