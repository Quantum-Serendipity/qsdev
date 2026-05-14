# Signed Exchanges (SXG) — Technical Details

- **Source URL**: https://web.dev/articles/signed-exchanges
- **Retrieved**: 2026-05-14

## Signing Process

A site signs a request/response pair (an "HTTP exchange") in a way that makes it possible for the browser to verify the origin and integrity of the content independently.

The SXG format includes a signature header containing parameters like `cert-sha256`, `sig`, and `integrity`. The signing covers both the HTTP exchange and a binary-encoded CBOR file structure.

## Certificate Requirements

- Production use requires a certificate that supports the `CanSignHttpExchanges` extension
- Certificates must have a validity period no longer than 90 days
- Requires the requesting domain have a DNS CAA record configured
- Certificates can be obtained automatically from the Google certificate authority using any ACME client

## Verification Process

Enables browsers to verify the origin and integrity of the content independently of how the content was distributed, allowing the browser to display the URL of the origin site in the address bar rather than the URL of the server that delivered the content.

## Relationship to Web Bundles

SXG and Web Bundles are two distinct technologies that don't depend on each other — Web Bundles can be used with both signed and unsigned exchanges. Both advance the creation of a "web packaging" format that allows sites to be shared in their entirety for offline consumption.

## Offline Use Cases

SXG implementation advances a variety of use cases such as offline internet experiences and serving from third-party caches.

## Browser Support

SXG is supported by Chromium-based browsers (starting with versions: Chrome 73, Edge 79, and Opera 64). Not supported in Firefox or Safari.
