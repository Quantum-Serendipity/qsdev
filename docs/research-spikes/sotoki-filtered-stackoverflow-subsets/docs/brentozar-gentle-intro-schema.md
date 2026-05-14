---
source: https://www.brentozar.com/archive/2018/02/gentle-introduction-stack-overflow-schema/
retrieved: 2026-05-14
---

# A Gentle Introduction to the Stack Overflow Schema

Covers four primary tables with relationships:

## Tables and Relationships
- **Users**: Central hub connecting to Posts (OwnerUserId), Comments, Badges, and Votes. Contains Age, CreationDate, LastAccessDate, Reputation.
- **Posts**: Connects to Comments and Votes via PostId. References Users through OwnerUserId. Includes AcceptedAnswerId, AnswerCount, CommentCount, FavoriteCount, Score.
- **Comments**: References Users, Posts. Includes CreationDate and Score.
- **Badges**: Links to Users.
- **Votes**: References Users, Posts. Contains BountyAmount and CreationDate. Largely anonymized.

Note: Article states "This isn't exhaustive, it's just for the four main tables that we use when writing demo queries." Directs to official Meta SE documentation for complete schema.
