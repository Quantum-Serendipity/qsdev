---
source: https://stackoverflow.blog/2010/04/26/new-linked-posts/
retrieved: 2026-05-14
---

# Stack Overflow Linked Posts Feature

## Sidebar Display

Two distinct sidebar panels:

**Linked Sidebar**: Community-provided connections in a list. Links are bi-directional - 
if question A links to question B, the link appears on BOTH questions' sidebars.

**Related Sidebar**: Machine-generated suggestions using weighted algorithm:
- Tags: +10 weight
- Titles: +5 weight  
- Body text: +1 weight

## Link Creation

**Manual**: Community members add links through answers, comments, or question edits.
Any link to another SO question in these areas automatically creates a Linked sidebar entry.

**Automatic**: Related panel uses full-text matching across question metadata.

## Statistics

No quantitative data provided in this blog post about volume or frequency of linked posts.
Feature announced April 2010.

## Implication for Tag-Filtered Subsets

The Linked sidebar is populated from PostLinks.xml entries. These are cross-question 
references that frequently cross tag boundaries (e.g., a Python question linking to a 
Linux question about file permissions). In a tag-filtered subset, many of these linked 
sidebar entries would point to non-existent pages.
