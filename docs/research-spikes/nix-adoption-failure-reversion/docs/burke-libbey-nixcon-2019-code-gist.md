<!-- Source: https://gist.github.com/burke/694d504be69998dbe4477f80ffa90951 -->
<!-- Retrieved: 2026-03-20 -->

# Code Release for NixCon 2019 - Burke Libbey (Shopify)

**License:** MIT (Shopify, 2019)

## Overview

Code extracted from Shopify's private codebases and presented at NixCon 2019. References external libraries including `cli-ui` and `cli-kit` that aren't included.

## Files & Content

### 1. dev-nix-post-build-hook
Bash script that spools Nix output paths to `/opt/dev/var/spool/nix-copy` for later cache uploading, creating directories and touching files for each completed build output.

### 2. dev.up.minio.plist.in
launchd configuration template for running MinIO gateway with GCS backend, configured with environment variables for access credentials and address binding.

### 3. gcloud.lua
Lua module providing Google Cloud authentication. Functions include:
- Token caching mechanism with expiry checking
- Support for both service accounts and authorized user credentials
- JWT generation and refresh token exchange

### 4. nix-cache.conf
Nginx configuration proxying binary cache requests. Handles both direct paths and `/nar/` locations, managing URL encoding for GCS bucket access while maintaining Nix compatibility.

### 5. nix-minio.conf
Nginx server proxying local MinIO and falling back to cache.nixos.org. Includes sophisticated error handling converting 404 responses to XML format that Nix understands.

### 6. output_parser.rb
Ruby class parsing verbose Nix build output. Tracks derivation states across: initial state, listing builds/fetches, active work, awaiting build completion.

### 7. output_ui.rb
UI wrapper displaying real-time build progress using spinners and status widgets showing download/build counts.

### 8. setup-hook-to-shadowenv
Bash script emulating nixpkgs setup.sh functionality, converting Nix package setup-hooks into shadowenv (environment variable management) format without implementing full build machinery.

### 9. status.rb
Ruby state machine managing fetch and build status transitions. Enforces allowed transitions (waiting->running->succeeded/failed) and logs completion metrics via Monorail.

### 10. upload_to_cache.rb
Periodic daemon task uploading completed Nix derivations to private cache. Processes spooled entries, signs paths with configured keys, and copies to MinIO backend.

## Key Observations

The codebase reveals Shopify built substantial custom infrastructure around Nix:
- Binary cache backed by GCS via MinIO
- Custom build output parsing and UI (Ruby)
- shadowenv integration for environment variable management
- launchd integration for macOS services
- Nginx-based cache proxy layer

This represents significant engineering investment in making Nix work for their development workflow in 2019.
