---
source: https://raw.githubusercontent.com/openzim/sotoki/main/src/sotoki/constants.py
retrieved: 2026-05-14
---

# Sotoki Constants

## Pagination Limits
- Questions: NB_PAGINATED_QUESTIONS = 15 per page * 100 pages = 1,500 total
- Tags: NB_PAGINATED_QUESTIONS_PER_TAG = 15 * 100 = 1,500 questions per tag
- Users: NB_PAGINATED_USERS = 36 * 100 = 3,600 users max

## Download Resilience
- Maximum 5 retry attempts per file
- Adaptive timing: 10ms minimum to 60-second maximum intervals
- Failure threshold monitoring with 50-file minimum before evaluation
