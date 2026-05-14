# Warcsigner — CLI Tool for Signing WARC Files

- **Source URL**: https://github.com/ikreymer/warcsigner
- **Retrieved**: 2026-05-14

## Overview
Python-based CLI tool providing cryptographic signing and verification for WARC (Web ARChive) and gzip-chunked files.

## Signing Mechanism
**RSA Cryptography**: Uses the python-rsa library. Public and private keys must be in PEM format.

## Signature Storage
Signatures are stored within gzip structure: the signature is stored in an extra gzip chunk containing no data but using a custom extra field to store the signature. This allows quick signature access at a fixed offset from the file's end. Most standard gzip tools ignore these custom headers, making the modification transparent.

## Verification Process
Verifies signatures and can optionally remove them via a `remove=True` parameter. On successful verification with removal, the file is unaltered if the verification fails.

## API Capabilities
- Sign/verify via filename or file-like objects with `read()` methods
- Streaming support with optional `size=` parameter to eliminate `seek()` calls
- Extract unsigned streams from signed files using `get_unsigned_stream()`

## Maintenance Status
**Inactive**: 14 total commits, no release versions published. Uses .travis.yml (legacy CI). Unclear current maintenance.

## Language
100% Python implementation.
