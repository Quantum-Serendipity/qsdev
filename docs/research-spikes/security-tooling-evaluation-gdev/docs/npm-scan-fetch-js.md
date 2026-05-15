<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/backend/fetch.js -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — backend/fetch.js (Full Source)

Core scan pipeline: fetches package tarball from npm registry, extracts to temp dir, reads all .js files.

```javascript
import fs from 'fs';
import os from 'os';
import path from 'path';
import { extract } from 'tar';
import zlib from 'zlib';
import { Readable } from 'stream';
import { pipeline } from 'stream/promises';

export async function fetchPackage(target, options = {}) {
  const { cacheDir, cacheTTL = 604800, cacheMaxSize = 1000000000 } = options;
  // Parse name@version from target string
  // Check cache if cacheDir set
  // Fetch metadata from https://registry.npmjs.org/{name}/{version|latest}
  // Download tarball, enforce 500MB limit
  // Extract to tmpDir, return { pkgJson, jsFiles, tmpDir, meta }
}

// Cache management: getFromCache, saveToCache, pruneCache (LRU by age)
// Local tarball scanning: scanLocalTarball(filePath)
// Extraction: extractTarball -> mkdirSync, gunzip, tar extract, read package.json + all .js files
// File walking: walkFiles(dir, ext) — recursive, skips node_modules
// Cleanup: cleanup(tmpDir) -> fs.rmSync

export async function scanLocalTarball(filePath) {
  const buffer = fs.readFileSync(filePath);
  const tmpDir = path.join(os.tmpdir(), 'npm-scan-local-' + Date.now());
  return await extractTarball(buffer, tmpDir);
}
```

Key implementation details:
- Uses native fetch (Node 18+), no external HTTP deps
- Tarball size limit: 500MB
- Cache TTL: 7 days default, LRU eviction with 80% target
- Extracts only .js files for analysis
- Skips node_modules during file walk
