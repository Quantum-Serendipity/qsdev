<!-- Source: https://github.com/openzim/sotoki/issues/287 -->
<!-- Retrieved: 2026-05-14 via gh API -->

# Issue #287: Support for tag filtering — Full Discussion

**Created**: 2023-07-15 by natamox
**Status**: Open (14 comments)
**Updated**: 2023-07-20

## Original Request

> Because the whole thing is really too big, more than 70 GB. For example, I only want to grab javascript or other tags, how to do it, thank you

## Maintainer Responses

### kelson42 (Kiwix founder):
> This is not possible for now. But should be possible, though maybe not that trivial. Main problem of the moment is that the scraper can not scrape recent dumps, because StackExchange does not provide them anymore.

> I renamed the ticket. What you'd want is to be able to supply a list of tags and get a ZIM with just the listed tags' content, right?

> It would be a great feature indeed, but it's not implemented yet.

> I think this is not only a valid feature request, but actually a pretty good one. We could for example do one ZIM per mainstream programming language. How complex would that be?

### rgaudin (sotoki maintainer):
> As kelson42 said, sotoki's future is unclear ATM because Stack Exchange stopped providing XML dumps. [Note: SE later resumed dumps]

> For extracting and repacking, it's possible but since we only bundle the HTML version of questions without metadata, you'd have to parse the HTML of every entry to find the tags… Also, the tag-less navigation would need to be fixed and the related links would not work either.

> You're better off implementing this ticket, way less work and outcome is clear and solid 😉

### rgaudin's Implementation Sketch:

> It's quite easy:
> - cli param to capture wanted tags ; parse to list
> - in `tags.py`, skip if `TagName` not in the list (-> tag not recorded to db)
> - in `posts.py` in both passes parsers, list of tags should be filtered to requested one instead of just retrieved from XML. Cleaner approach is too check if in DB.tags_ids.inverse
> - about template should mention that it's restricted to this list.
> And I think that should do it

### benoit74 (sotoki contributor):
> Don't you need something to fix broken links to questions that have been filtered out? You might for instance redirect these links to a static page which indicate that this question has not be scraped due to tag filters, like I did for iFixit. Or is this already handled in the scraper / not wished?

### rgaudin's response on links:
> We already remove links to questions that are not in the DB. What's not handled is the `a/{aId}` shortcut because we don't store answers in the DB. For valid links it's not a problem because we create a redirect elsewhere but for missing target this would lead to a dead link.

## Key Takeaways

1. **Maintainer endorsement**: Both kelson42 (Kiwix founder) and rgaudin (sotoki maintainer) view this as a valid, desirable feature
2. **Implementation guidance from maintainer**: rgaudin provided a concrete implementation sketch
3. **Post-processing ZIM explicitly discouraged**: rgaudin specifically said pre-filtering XML or modifying sotoki is better than extracting/repacking a ZIM
4. **Link handling mostly covered**: Existing code already removes links to questions not in the DB; the only gap is answer shortcut URLs (`a/{aId}`)
5. **No one has implemented it**: Despite the guidance, no PR has been submitted (as of 2026-05-14)
6. **natamox attempted to set up dev environment** but hit dependency issues; unclear if they continued work
