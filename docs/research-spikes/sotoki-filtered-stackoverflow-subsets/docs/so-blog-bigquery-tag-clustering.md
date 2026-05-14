# Making Sense of the Metadata: Clustering 4,000 Stack Overflow tags with BigQuery k-means

- **Source**: https://stackoverflow.blog/2019/07/24/making-sense-of-the-metadata-clustering-4000-stack-overflow-tags-with-bigquery-k-means/
- **Retrieved**: 2026-05-14

## Quantitative Data

- Filtered for tags with >180 questions since 2018
- Used percent>0.03 threshold (tags needed to co-occur in at least 3% of questions)
- javascript is related to html, python is related to pandas (qualitative relationships)
- Clustering methodology focus, not raw statistics

## Missing Data

No specific question counts per tag or detailed co-occurrence percentages provided.
The underlying dataset would need to be accessed via BigQuery directly.
