---
source: https://samadhiweb.com/blog/2012.11.15.stackoverflow.html
retrieved: 2026-05-14
---

# Parsing StackOverflow Data Dump

## Posts.xml Schema Fields:
- Id: Unique identifier
- PostTypeId: 1=Question, 2=Answer
- ParentID: Present only for answers (PostTypeId=2)
- AcceptedAnswerId: Present only for questions (PostTypeId=1)
- CreationDate, Score, ViewCount
- Body: Post content
- OwnerUserId, LastEditorUserId, LastEditorDisplayName
- LastEditDate, LastActivityDate, CommunityOwnedDate, ClosedDate
- Title, Tags, AnswerCount, CommentCount, FavoriteCount

## Other Files in Sep 2011 dump:
badges.xml, comments.xml, posthistory.xml, users.xml, votes.xml

Note: Article does not describe the angle-bracket tag format explicitly.
