---
source: https://raw.githubusercontent.com/zhenv5/PyStack/master/README.md
retrieved: 2026-05-14
---

# PyStack - Scripts for Processing Stack Exchange Data Dump

## XML Files Handled
Posts.xml, PostLinks.xml, Votes.xml, Badges.xml, Comments.xml

## Processing Capabilities
- **Posts**: Extracts relationships between questions and answerers, generates tag mappings, stores Q&A content in pickle format
- **PostLinks**: Identifies related questions and duplicate question pairs by link type
- **Votes**: Extracts bounty information
- **Badges**: Compiles user badge achievements with dates
- **Comments**: Maps comments to posts with user scores and text content

## Tag Handling
Does NOT filter by tag during processing. Instead, extracts question tags into a dictionary pickle file (key=question_id, value=list_of_tags). Users apply tag-based filtering in their own analysis phase.

## Execution
Individual task scripts or pystack.py --task parameter. Outputs as CSV and pickle files.
