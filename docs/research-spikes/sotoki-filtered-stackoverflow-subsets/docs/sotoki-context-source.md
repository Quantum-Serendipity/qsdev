# Sotoki context.py Source Code
- **Source URL**: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/context.py
- **Retrieved**: 2026-05-14

---

## Context Dataclass Fields

### Required:
- domain: str — StackExchange domain
- mirror: str — URL to download archives
- title: str — ZIM title
- description: str — ZIM description

### ZIM Metadata:
- name: str | None
- long_description: str | None
- creator: str = "Stack Exchange"
- publisher: str = "openZIM"
- fname: str = ""
- tags: list[str] = [] — **ZIM metadata tags, NOT content filtering tags**
- flavour: str | None
- favicon: str | None

### Censorship/Content Options:
- censor_words_list: str
- without_images: bool
- without_user_profiles: bool
- without_external_links: bool
- without_unanswered: bool
- without_users_links: bool
- without_names: bool

### Performance/Debug:
- nb_threads: int = 1
- redis_url: str
- output_dir: Path
- tmp_dir: Path
- keep_build_dir: bool
- keep_redis: bool
- debug: bool
- prepare_only: bool
- Various dev-skip flags

### Key Observation:
The `--tags` CLI flag maps to `tags: list[str]` in Context, but this is for **ZIM file metadata tags** (like "stackexchange", "_category:stack_exchange"), NOT for filtering which StackExchange posts to include. There is no field for content/tag filtering.
