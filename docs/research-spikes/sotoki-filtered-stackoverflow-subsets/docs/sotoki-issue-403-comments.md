---
source: https://github.com/openzim/sotoki/issues/403
retrieved: 2026-05-14
type: github-issue-comments
---

# Issue #403 Comments: stackoverflow.com still fails while processing images

## @benoit74 (2026-04-05)
"It stopped at 85624 images over 4099807 (4M). Problem is that:
- other domains run fine
- images download is the very last step
- SO is so big it takes days to reach the image download step

We probably have an issue linked to this specific domain, and we cannot realistically repeat runs until we find the root cause.

I've hence build a custom image which will:
- run the whole logic except it will not add anything the ZIM (should save memory and time)
- dump the whole list of image URLs to download in a TXT file (so that we can repeat only this part)

It is currently running on `kathrin` machine (by hand)"

## @its-me-maady (2026-04-06)
Bug report: `get_version_ident_for()` in imager.py has a scoping issue where `resp` is unbound if `requests.head()` raises.

## @benoit74 (2026-04-21)
"Now I have the full list of URLs needed by stackoverflow.com run (can't upload to Github, it is too big).

Discovered the main issue: `i.sstatic.net` domain is where we have 98% of images to download for stackoverflow.com (4,016,201 out of 4,099,803 total -- ~4M images).

When throttling, Cloudflare returns 403; this was not accounted for in codes which should slow scraper down; pretty sure it caused `kathrin` to be currently completely blocked from any requests on SE domains hosted by Cloudflare."

## @benoit74 (2026-04-21 - follow-up)
"Scraper has already downloaded 114k images in about 3 hours, everything seems to indicate including 403 in codes causing scraper to slow down is sufficient to not being blocked anymore."

## Key Findings
- SO ZIM requires downloading ~4.1 MILLION images
- 98% come from i.sstatic.net (Cloudflare-protected)
- Cloudflare throttling/blocking was a major bottleneck
- Building SO ZIM takes DAYS just to reach the image download step
- The build runs on a machine called "kathrin"
- At 114k images per 3 hours, downloading all 4M images would take ~105 hours (~4.4 days) just for images
