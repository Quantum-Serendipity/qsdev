# Stack Exchange Data Dump Structure Research

## Overview

This report documents the complete structure of Stack Exchange data dumps, with a focus on how tags are represented and the feasibility of pre-filtering dumps by tag before feeding them to sotoki (the openZIM StackExchange-to-ZIM scraper).

Stack Exchange publishes quarterly data dumps containing the complete public content of all ~368 sites in the network. Each site is a separate 7z archive containing XML files. All sites use an identical schema, so findings apply across the network.

---

## 1. Complete XML Schema

### Files in Each Site Archive

Each site archive contains 8 XML files (all use `<row>` elements with attributes corresponding to columns):

| File | Description | SO Row Count | SO Data Size |
|------|-------------|-------------|-------------|
| **Posts.xml** | Questions, answers, tag wikis, and other post types | 59.82M | ~97-105 GB uncompressed |
| **PostHistory.xml** | Edit history for all posts | 160.79M | ~67 GB |
| **Comments.xml** | Comments on posts | 90.38M | ~11 GB (28 GB in some estimates) |
| **Votes.xml** | Upvotes, downvotes, favorites, bounties, close/reopen votes | 238.98M | ~2 GB |
| **Users.xml** | User profiles | 22.48M | ~1.4 GB |
| **Badges.xml** | Badges awarded to users | 51.29M | ~800 MB |
| **Tags.xml** | Tag definitions (name, count, wiki references) | ~65K | ~1.1 MB compressed |
| **PostLinks.xml** | Duplicate and related-post links | 6.55M | ~130 MB |

**Total for Stack Overflow**: ~21 GB compressed (Posts.7z alone), ~92 GB total compressed across all SO files. PostHistory.7z is actually the largest compressed file at 35.2 GB.

### Posts.xml Schema

```xml
<row Id="4" PostTypeId="1" AcceptedAnswerId="7" CreationDate="2008-07-31T21:42:52.667"
     Score="743" ViewCount="62210" Body="&lt;p&gt;..." OwnerUserId="8"
     LastEditorUserId="6786713" LastEditDate="2023-01-06T04:13:35.570"
     LastActivityDate="2023-01-06T04:13:35.570" Title="When setting a form's opacity..."
     Tags="&lt;c#&gt;&lt;floating-point&gt;&lt;type-conversion&gt;&lt;double&gt;&lt;decimal&gt;"
     AnswerCount="13" CommentCount="3" FavoriteCount="47" ContentLicense="CC BY-SA 4.0"
     CommunityOwnedDate="..." ClosedDate="..." />
```

Key fields:
- **Id**: Primary key
- **PostTypeId**: 1=Question, 2=Answer, 3=Orphaned tag wiki, 4=Tag wiki excerpt, 5=Tag wiki, 6=Moderator nomination, 7=Wiki placeholder (election description), 8=Privilege wiki
- **ParentId**: For answers (PostTypeId=2), references the question's Id. NOT present on questions.
- **AcceptedAnswerId**: For questions, references the accepted answer's Id
- **Tags**: Only present on questions (PostTypeId=1). Angle-bracket-delimited string, e.g., `<c#><floating-point><double>`. In raw XML, angle brackets are entity-encoded as `&lt;` and `&gt;`.
- **Body**: HTML content, entity-escaped in the XML
- **OwnerUserId**: References Users.Id
- **LastEditorUserId**: References Users.Id
- **ContentLicense**: Typically "CC BY-SA 4.0" (or earlier versions)

### Tags.xml Schema

```xml
<row Id="3" TagName="javascript" Count="2528543" ExcerptPostId="3624960" WikiPostId="3607052" />
```

Fields:
- **Id**: Primary key, used as TagId in derived join tables
- **TagName**: The canonical tag string (e.g., "javascript")
- **Count**: Number of questions using this tag
- **ExcerptPostId**: References Posts.Id for the tag wiki excerpt (PostTypeId=4)
- **WikiPostId**: References Posts.Id for the tag wiki body (PostTypeId=5)

### Comments.xml Schema

```sql
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,     -- References Posts.Id (question OR answer)
    user_id INTEGER,              -- References Users.Id
    score SMALLINT NOT NULL,
    content_license VARCHAR(64) NOT NULL,
    user_display_name VARCHAR(64),
    text TEXT,
    creation_date TIMESTAMP NOT NULL
);
```

### Users.xml Schema

```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    account_id INTEGER,           -- Network-wide account ID
    reputation INTEGER NOT NULL,
    views INTEGER DEFAULT 0,
    down_votes INTEGER DEFAULT 0,
    up_votes INTEGER DEFAULT 0,
    display_name VARCHAR(255) NOT NULL,
    location VARCHAR(512),
    profile_image_url VARCHAR(255),
    website_url VARCHAR(255),
    about_me TEXT,
    creation_date TIMESTAMP NOT NULL,
    last_access_date TIMESTAMP NOT NULL
);
```

### Votes.xml Schema

```sql
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER,              -- Often NULL (anonymized)
    post_id INTEGER NOT NULL,     -- References Posts.Id
    vote_type_id SMALLINT NOT NULL,
    bounty_amount SMALLINT,
    creation_date TIMESTAMP NOT NULL
);
```

VoteTypeId values: 1=AcceptedByOriginator, 2=UpMod (upvote), 3=DownMod (downvote), 4=Offensive, 5=Favorite, 6=Close, 7=Reopen, 8=BountyStart, 9=BountyClose, 10=Deletion, 11=Undeletion, 12=Spam, 15=ModeratorReview.

### PostHistory.xml Schema

```sql
CREATE TABLE post_history (
    id SERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,     -- References Posts.Id
    user_id INTEGER,
    post_history_type_id SMALLINT NOT NULL,
    user_display_name VARCHAR(64),
    content_license VARCHAR(64),
    revision_guid uuid,
    text TEXT,                    -- Contains edit diffs, tag changes, etc.
    comment TEXT,                 -- Edit summary
    creation_date TIMESTAMP NOT NULL
);
```

### PostLinks.xml Schema

```sql
CREATE TABLE post_links (
    id SERIAL PRIMARY KEY,
    related_post_id INTEGER NOT NULL,  -- References Posts.Id
    post_id INTEGER NOT NULL,          -- References Posts.Id
    link_type_id SMALLINT NOT NULL,    -- 1=Linked, 3=Duplicate
    creation_date TIMESTAMP NOT NULL
);
```

### Badges.xml Schema

```sql
CREATE TABLE badges (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,     -- References Users.Id
    class SMALLINT NOT NULL,      -- 1=Gold, 2=Silver, 3=Bronze
    name VARCHAR(64) NOT NULL,    -- Badge name (some are tag-based)
    tag_based BOOL NOT NULL,      -- Whether badge is tag-specific
    date TIMESTAMP NOT NULL
);
```

---

## 2. How Tags Are Represented and Linked to Posts

### Tag Storage Architecture

Tags use a **denormalized string field** on questions, not a normalized join table:

1. **Posts.xml `Tags` attribute**: Present only on questions (PostTypeId=1). Contains a concatenated string of tag names delimited by angle brackets: `<python><django><orm>`. This is the primary tag-to-post link in the raw dump.

2. **Tags.xml**: A lookup table mapping tag names to IDs, counts, and wiki post references. It does NOT contain post-tag associations.

3. **No PostTags join table in the dump**: The raw XML dump does not include a many-to-many join table. Some import tools (like Networks-Learning/stackexchange-dump-to-postgres) create derived `PostTags` and `AllPostTags` tables during import by parsing the Tags string field.

### Relationship Model

```
Questions (PostTypeId=1) --[Tags field]--> Tag names (denormalized string)
                         --[ParentId]----> Answers (PostTypeId=2) (reverse: answers point to questions)
Tags.xml                 --[ExcerptPostId]--> Posts (PostTypeId=4, tag wiki excerpt)
                         --[WikiPostId]----> Posts (PostTypeId=5, tag wiki body)
Comments.xml             --[PostId]--------> Posts (any type)
Votes.xml                --[PostId]--------> Posts (any type)
PostHistory.xml          --[PostId]--------> Posts (any type)
PostLinks.xml            --[PostId, RelatedPostId]--> Posts (questions)
```

Key insight: **Answers do not carry tags**. Only the parent question has the Tags field. An answer's tags are implicitly inherited from its parent question via ParentId.

### Tag Format Parsing

The Tags field in the XML is entity-encoded: `Tags="&lt;c#&gt;&lt;floating-point&gt;"` which resolves to `<c#><floating-point>` after XML parsing.

Sotoki's parser handles both formats seen in practice:
```python
re.split(r"\||><", post["Tags"][1:-1])
```
This splits on either `|` (pipe) or `><` (angle bracket boundary), after stripping the outer delimiters.

---

## 3. Tag Synonyms and Hierarchies

### Tag Synonyms Are NOT in the Data Dump

The `TagSynonyms` table exists in the live Stack Exchange database but is **not exported** in the public data dump. This is an intentional omission by Stack Exchange -- they only export a subset of their internal tables.

### How Synonyms Work in Practice

- When a synonym is defined (e.g., `[js]` -> `[javascript]`), all future posts tagged with the synonym are **silently remapped** to the canonical tag.
- The remapping is applied at write time, so **Posts.xml already contains canonical tag names**. You will never see a synonym tag in the dump's Tags field.
- This means tag-based filtering on the dump data will correctly capture all posts, including those originally tagged with a synonym.

### Tag Hierarchy

Stack Exchange does not have a formal tag hierarchy or taxonomy. Tags are a flat folksonomy. However:
- **Tag wikis** (PostTypeId=4 for excerpt, PostTypeId=5 for body) provide descriptive context
- **Tag badges** (in Badges.xml, `TagBased=true`) are awarded per-tag
- **PostLinks** with LinkTypeId=3 mark duplicate questions, which can cross tag boundaries

### Obtaining Tag Synonyms if Needed

If synonym data is required, it can be queried via SEDE (Stack Exchange Data Explorer) at data.stackexchange.com, which has access to the full database including TagSynonyms. The SEDE query would be:
```sql
SELECT SourceTagName, TargetTagName, CreationDate, ApprovalDate
FROM TagSynonyms
WHERE ApprovalDate IS NOT NULL
```

---

## 4. Feasibility of Pre-Filtering by Tag

### The Cascade Problem

Filtering Posts.xml by tag is straightforward for questions but creates a cascade of related data that must also be filtered to produce a coherent subset:

#### Tier 1: Direct Tag Match (Questions)
- Scan Posts.xml for rows where PostTypeId=1 AND Tags field contains the target tag(s)
- Collect all matching question IDs into a set

#### Tier 2: Dependent Posts (Answers + Tag Wikis)
- **Answers**: Scan Posts.xml again for PostTypeId=2 where ParentId is in the question set
- **Tag wiki excerpts**: Posts where Id matches Tags.xml ExcerptPostId for the target tags
- **Tag wiki bodies**: Posts where Id matches Tags.xml WikiPostId for the target tags

#### Tier 3: Post-Referenced Data
For all post IDs collected in Tiers 1 and 2:
- **Comments.xml**: Keep rows where PostId is in the post set
- **Votes.xml**: Keep rows where PostId is in the post set
- **PostHistory.xml**: Keep rows where PostId is in the post set (THIS IS THE LARGEST FILE)
- **PostLinks.xml**: Keep rows where PostId OR RelatedPostId is in the post set

#### Tier 4: User Data
- Collect all unique user IDs referenced by OwnerUserId, LastEditorUserId (from Posts), UserId (from Comments, Votes, PostHistory)
- **Users.xml**: Keep rows where Id is in the user set
- **Badges.xml**: Keep rows where UserId is in the user set

#### Tier 5: Tag Metadata
- **Tags.xml**: Keep rows for the target tag(s) and any other tags that appear on the filtered questions (a question tagged `<python><django>` filtered for `python` should also retain the `django` tag definition)

### Implementation Strategy

A two-pass streaming approach is feasible:

**Pass 1** (Posts.xml, ~100 GB):
1. Stream through Posts.xml once
2. For PostTypeId=1, check Tags field for target tag(s). If match, record question ID and all referenced user IDs.
3. For PostTypeId=2, buffer or record ParentId. (Cannot filter yet -- need to know which questions matched.)

This creates a chicken-and-egg problem: you need to know which questions match before you can filter answers, but questions and answers are interleaved in Posts.xml.

**Solutions**:
- **Two-pass on Posts.xml**: First pass collects matching question IDs. Second pass writes questions + their answers. Memory cost: storing the set of matching question IDs (a few million integers for a popular tag = ~50 MB).
- **Single-pass with buffering**: Buffer answers whose ParentId status is unknown. Write or discard when parent question is encountered. Risky with 100 GB files.
- **Database intermediate**: Import to SQLite/PostgreSQL first, then query with joins. More disk space but simpler logic.

**Pass 2** (all other files):
Stream through each file once, keeping only rows that reference a post ID or user ID in the collected sets. These are simple set-membership lookups.

### Memory/Performance Estimates

For a single popular tag like `python` (~2.2M questions):
- Question ID set: ~2.2M integers = ~17 MB
- Answer ID set: ~3-5M integers = ~40 MB
- User ID set: ~1-2M integers = ~16 MB
- Total working memory: < 200 MB

Processing time dominated by I/O: streaming through ~100 GB Posts.xml + ~67 GB PostHistory.xml + smaller files. With SAX parsing at ~50-100 MB/s, expect 30-60 minutes for the full pipeline.

### Tag Co-occurrence Consideration

Questions typically have 2-5 tags. Filtering for `python` will include questions also tagged with `django`, `pandas`, `numpy`, etc. The resulting subset will contain substantial content from related tags. This is actually desirable for a useful offline reference.

---

## 5. Existing Tools for Dump Filtering

### Direct Tag-Filtering Tools

**Seekoff** (github.com/Caspia/seekoff)
- Offline Stack Overflow reader with tag inclusion/exclusion
- Indexes XML dumps into Elasticsearch, serves via web UI
- Does NOT produce a filtered XML dump -- creates a filtered search index
- Architecture: XML -> Elasticsearch -> Node.js web server
- Not suitable as a pre-filter for sotoki (wrong output format)

**PyStack** (github.com/zhenv5/PyStack)
- Python scripts for processing SE data dumps
- Extracts tag mappings into pickle files (question_id -> [tags])
- Does NOT filter by tag during processing; tag filtering is a post-processing step
- Could be adapted for tag-based filtering but would need work

**StackLite** (github.com/dgrtwo/StackLite)
- Lightweight dataset of SO questions and tags only
- Question data and tag pairings stored separately for easy analysis
- Useful for identifying which question IDs match a tag, but doesn't include post content

### General Dump Processing Tools

**sodata** (github.com/sth/sodata)
- Imports XML dumps into SQLite, PostgreSQL, or CSV
- Schema defined in soschema.hpp
- Could be used as step 1 of a filter pipeline: import, query by tag, export subset

**SODDI** (github.com/BrentOzarULTD/soddi)
- Stack Overflow Data Dump Importer for SQL Server
- Full import tool, no tag filtering

**stackexchange-xml-converter** (github.com/SkobelevIgor/stackexchange-xml-converter)
- Converts XML dumps to CSV format
- No tag filtering, but CSV output could be filtered with standard tools

**Networks-Learning/stackexchange-dump-to-postgres**
- Python scripts to import into PostgreSQL
- Creates derived PostTags and AllPostTags join tables
- Best candidate for a database-intermediate filtering approach

### LLM Training Tools

**EleutherAI/stackexchange-dataset** (github.com/EleutherAI/stackexchange-dataset)
- Converts dumps to text datasets for LLM training
- Filters by score and response count, not by tag
- Outputs question-answer pairs as plain text

### No Existing Tool Does What We Need

None of the existing tools produce a **filtered XML dump** (same format as input, but subset by tag) suitable for feeding to sotoki. A custom filtering script would be needed.

---

## 6. Stack Exchange Data Explorer (SEDE)

SEDE (data.stackexchange.com) provides SQL query access to the live SE database:

### Capabilities
- Full SQL (T-SQL) query access to all public tables including TagSynonyms
- Parameterized queries with tag parameters: `WHERE Tags LIKE '%<##TagName##>%'`
- Updated weekly with latest data
- Results downloadable as CSV

### Limitations
- **50,000 row limit** per query result (confirmed by Stack Overflow blog post about BigQuery)
- Stack Overflow has ~60M posts -- even filtering to a single tag, popular tags like `python` have ~2.2M questions + their answers, far exceeding the 50K limit
- No way to export full XML dumps -- only tabular CSV results
- Query timeout limits apply

### Verdict for Our Use Case
SEDE is **not viable** for extracting tag-filtered datasets at the scale needed. The 50K row limit means you would need 44+ paginated queries just for `python` questions, and you'd still get CSV output rather than the XML format sotoki expects.

SEDE is useful for:
- Obtaining TagSynonyms data (small table, fits in 50K limit)
- Exploratory queries to understand tag distributions
- Validating filtering logic before running against the full dump

---

## 7. Size Estimates

### Stack Overflow Compressed Archive Sizes (as of 2024-12)
| File | Compressed (7z) | Estimated Uncompressed |
|------|-----------------|----------------------|
| Posts.7z | 21.4 GB | ~97-105 GB |
| PostHistory.7z | 35.2 GB | ~67+ GB |
| Comments.7z | 6.5 GB | ~11-28 GB |
| Votes.7z | 2.1 GB | ~2 GB |
| Users.7z | 944 MB | ~1.4 GB |
| Badges.7z | 515 MB | ~800 MB |
| PostLinks.7z | 144 MB | ~130 MB |
| Tags.7z | 1.1 MB | ~few MB |
| **Total** | **~67 GB** | **~180-210 GB** |

### Filtered Subset Size Estimates

For a single popular tag like `python` (~2.2M questions, ~4% of all SO questions):
- Posts subset: ~4-5 GB uncompressed (questions + answers + tag wikis)
- Comments subset: ~0.5-1 GB
- PostHistory subset: ~3-4 GB
- Votes subset: ~100 MB
- Users subset: ~100 MB
- **Total filtered subset**: ~8-11 GB uncompressed, ~2-3 GB compressed

For a niche tag like `haskell` (~50K questions, ~0.08%):
- Total filtered subset: ~200-400 MB uncompressed, ~50-100 MB compressed

---

## 8. Conclusions and Recommendations

### Pre-filtering is feasible and practical

1. **Tags are queryable**: The denormalized `<tag1><tag2>` format in Posts.xml is trivially parseable with string matching or regex. No join tables needed.

2. **Synonym transparency**: Tag synonyms are resolved before data reaches the dump, so filtering by canonical tag name captures all relevant posts.

3. **Cascade is manageable**: The dependency chain (questions -> answers -> comments/votes/history -> users -> badges) requires tracking ID sets across files, but the memory footprint is small (~200 MB for popular tags).

4. **No existing tool does this**: A custom XML streaming filter needs to be built. The approach is:
   - Pass 1: Stream Posts.xml, collect matching question IDs
   - Pass 2: Stream Posts.xml again, write matching questions + their answers + tag wikis
   - Pass 3: Stream each remaining file, write rows referencing collected post/user IDs

5. **Sotoki compatibility**: The output must be valid XML in the same format as the original dump. Sotoki uses SAX parsing and expects the standard dump file structure.

6. **Scale is reasonable**: Even for Stack Overflow's 100 GB Posts.xml, SAX-based streaming with set lookups should complete in under an hour on modern hardware.

### Key Risk: PostHistory.xml

PostHistory.xml is the largest file by row count (160M rows) and the second largest compressed file (35.2 GB). It contains edit diffs and is essential for a complete offline reference. Filtering it requires the same post-ID set lookup as other files but on a much larger dataset. This is the bottleneck.

### Recommended Approach

Build a Python script using `xml.sax` (or `lxml.iterparse`) that:
1. Accepts a list of target tags as input
2. Makes two passes over Posts.xml (first to collect IDs, second to write filtered output)
3. Makes one pass over each remaining XML file with set-membership filtering
4. Outputs a complete, valid dump directory that sotoki can consume directly

This avoids the need for database intermediaries and keeps the pipeline simple: download dump -> filter by tag -> run sotoki -> get tag-specific ZIM file.
