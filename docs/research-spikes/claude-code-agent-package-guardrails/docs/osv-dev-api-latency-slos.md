<!-- Source: https://osv.dev/blog/posts/api-latency-improvements-and-revised-slos/ -->
<!-- Retrieved: 2026-05-12 -->

# OSV.dev API Performance Enhancement and Revised SLOs

## Architecture Changes

OSV.dev restructured its database indexing strategy to improve query efficiency. Previously, the system retrieved entire database entities for all queries. The new approach stores complete computed OSV records directly and maintains a separate table with only affected version information for matching operations.

## Performance Improvements (Before/After)

**Mean Latency Reductions:**
- `GET /v1/vulns/{id}`: 0.2s -> 0.04s (5x faster)
- `POST /v1/query`: 0.3s -> 0.12s (2.5x faster)
- `POST /v1/querybatch`: 1.8s -> 0.6s (3x faster)

**Percentile Performance:**
The batch query endpoint showed particularly notable gains, with P95 latency dropping from approximately 10 seconds to 3 seconds, significantly benefiting dependency scanning operations.

## Updated Service Level Objectives

The platform established endpoint-specific SLO targets:

| Endpoint | P50 | P90 | P95 |
|----------|-----|-----|-----|
| GET /v1/vulns/{id} | <=100ms | <=200ms | <=500ms |
| POST /v1/query | <=300ms | <=500ms | <=1s |
| POST /v1/querybatch | <=500ms | <=4s | <=6s |

## Future Direction

The team plans to speed up the ingestion and export pipelines to accelerate vulnerability information availability on the platform.
