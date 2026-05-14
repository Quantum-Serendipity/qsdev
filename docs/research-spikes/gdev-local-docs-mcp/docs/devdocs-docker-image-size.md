<!-- Source: https://github.com/orgs/freeCodeCamp/packages/container/devdocs/385738214?tag=latest-alpine -->
<!-- Retrieved: 2026-05-14 -->

# DevDocs Docker Image Size (Alpine)

## Image Details
- **Digest:** sha256:70aa78d81e7643913139bee057efc128b98f53dce4f0475aa6c83d511bb27893
- **Title:** "API Documentation Browser"
- **License:** MPL-2.0
- **Created:** April 1, 2025

## Layer Breakdown (8 layers, compressed)
1. Base layer: 3.6 MB
2. Configuration: 140 bytes
3. Application: 39.1 MB
4. Additional: 189 bytes
5. Metadata: 97 bytes
6. Data layer: 3.4 MB
7. **Documentation content: ~3.48 GB** (largest layer — ALL docs)
8. Config blob: 8,029 bytes

## Total Compressed Size: ~3.52 GB

## Key Observations
- The documentation content layer dominates (99%+ of image size)
- The application itself is tiny (~43 MB including Ruby + dependencies)
- Using `--default` instead of `--all` would dramatically reduce size
- A selective download would be on the order of hundreds of MB
- Monthly automatic updates mean re-pulling this large layer periodically
