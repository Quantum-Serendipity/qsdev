<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/backend/lockfile.js -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — backend/lockfile.js (Full Source)

```javascript
import { readFileSync } from 'fs';
import { resolve, dirname } from 'path';
import yaml from 'js-yaml';

export function parseLockfile(filePath, options = {}) {
  const { autoDetect = false } = options;
  try {
    const content = readFileSync(filePath, 'utf8');
    const ext = filePath.split('.').pop().toLowerCase();

    if (ext === 'json' || ext === 'jsonc') {
      return parseNpmLockfile(content, filePath);
    }
    if (ext === 'lock' && !autoDetect) {
      return parseYarnLockfile(content, filePath);
    }
    if (ext === 'yaml' || ext === 'yml') {
      return parsePnpmLockfile(content, filePath);
    }

    if (autoDetect) {
      if (content.trimStart().startsWith('{')) {
        return parseNpmLockfile(content, filePath);
      }
      if (content.includes('__metadata')) {
        return parsePnpmLockfile(content, filePath);
      }
      if (content.includes('@npm:') || /^\s*"?[\w@/-]+['"]?\s*,\s*$/m.test(content)) {
        return parseYarnLockfile(content, filePath);
      }
    }

    return parseNpmLockfile(content, filePath);
  } catch (e) {
    throw new Error(`Failed to parse lockfile: ${e.message}`);
  }
}

function parseNpmLockfile(content, filePath) {
  const lockfile = JSON.parse(content);
  const packages = [];

  if (lockfile.packages) {
    for (const [key, pkg] of Object.entries(lockfile.packages)) {
      if (key === '') continue;
      const name = pkg.name || key.replace(/^node_modules\//, '').replace(/^[^/]+\//, '');
      packages.push({
        name,
        version: pkg.version || 'unknown',
        resolved: pkg.resolved || '',
        integrity: pkg.integrity || '',
        path: key,
        peerDeps: pkg.peerDependencies || {},
        dev: pkg.dev || false,
        optional: pkg.optional || false,
        scripts: pkg.scripts || {},
        dependencies: pkg.dependencies || {}
      });
    }
  }

  const rootDeps = lockfile.packages?.['node_modules/'] || {};
  return {
    version: lockfile.lockfileVersion,
    packages,
    root: {
      name: rootDeps.name || 'unknown',
      version: rootDeps.version || 'unknown',
      dependencies: rootDeps.dependencies || {},
      devDependencies: rootDeps.devDependencies || {},
      peerDependencies: rootDeps.peerDependencies || {}
    }
  };
}

function parseYarnLockfile(content, filePath) {
  const packages = [];
  const lines = content.split('\n');
  let i = 0;
  const n = lines.length;

  const MULTI_ENTRY_RE = /^"?([\w@./-]+)@(\^?[\w.+\-~]+)"?\s*,\s*"?([\w@./-]+)@(\^?[\w.+\-~]+)"?\s*:\s*$/;
  const SINGLE_ENTRY_RE = /^"?([\w@./-]+)@(\^?[\w.+\-~]+)"?\s*:\s*$/;

  while (i < n) {
    let line = lines[i].trimEnd();
    let specs = [];

    const multiMatch = line.match(MULTI_ENTRY_RE);
    const singleMatch = line.match(SINGLE_ENTRY_RE);

    if (multiMatch) {
      specs = [
        { name: multiMatch[1], specVersion: multiMatch[2] },
        { name: multiMatch[3], specVersion: multiMatch[4] }
      ];
    } else if (singleMatch) {
      specs = [{ name: singleMatch[1], specVersion: singleMatch[2] }];
    }

    if (specs.length > 0) {
      let version = '';
      let resolved = '';
      let integrity = '';
      const dependencies = {};
      const optionalDependencies = {};
      const peerDependencies = {};
      let dev = false;
      let optional = false;

      i++;
      while (i < n) {
        const bodyLine = lines[i];
        const bodyTrim = bodyLine.trimEnd();

        if (bodyTrim === '' || bodyTrim.startsWith('#')) { i++; continue; }
        if (bodyTrim.endsWith(':') && !bodyLine.startsWith(' ')) { break; }

        if (bodyTrim.startsWith('version ')) {
          const vMatch = bodyTrim.match(/^version ['"]([^'"]+)['"]/);
          if (vMatch) version = vMatch[1];
        } else if (bodyTrim.match(/^\s*resolved\s+(.+)/)) {
          const rMatch = bodyTrim.match(/^\s*resolved\s+(.+)/);
          if (rMatch) {
            resolved = rMatch[1].trim().replace(/^['"]|['"]$/g, '');
            if (resolved.startsWith('https://registry.yarnpkg.com/')) {
              resolved = resolved.replace('https://registry.yarnpkg.com/', 'https://registry.npmjs.org/');
            }
          }
        } else if (bodyTrim.startsWith('integrity ')) {
          integrity = bodyTrim.replace('integrity ', '').trim();
        }
        // ... [dependencies, optionalDependencies, peerDependencies, dev, optional parsing]

        i++;
      }

      for (const { name, specVersion } of specs) {
        packages.push({
          name, version: version || specVersion, resolved, integrity,
          path: `node_modules/${name}`, peerDeps: peerDependencies,
          dev, optional, scripts: {}, dependencies, optionalDependencies
        });
      }
    } else { i++; }
  }

  return { version: 2, packages, root: { name: 'root', version: 'unknown', dependencies: {}, devDependencies: {}, peerDependencies: {} } };
}

function parsePnpmLockfile(content, filePath) {
  const lockfile = yaml.load(content);
  const packages = [];

  if (lockfile.packages) {
    for (const [key, pkg] of Object.entries(lockfile.packages)) {
      const nameMatch = key.match(/^\/(.+?)@([^@/]+)$/);
      if (!nameMatch) continue;
      const name = nameMatch[1];
      const version = nameMatch[2];

      packages.push({
        name, version,
        resolved: pkg.resolution?.url || '',
        integrity: pkg.resolution?.integrity || (pkg.resolution?.sha512 ? `sha512-${pkg.resolution.sha512}` : ''),
        path: `node_modules/${name}`,
        peerDeps: pkg.peerDependencies || {},
        dev: pkg.dev || false,
        optional: pkg.optional || false,
        scripts: pkg.hasBundledMedia ? { bundled: true } : {},
        dependencies: pkg.dependencies || {},
        optionalDependencies: pkg.optionalDependencies || {}
      });
    }
  }

  return { version: lockfile.version || (lockfile.lockfileVersion ?? 6), packages, root: { ... } };
}

export function checkMaliciousPatterns(pkg) {
  const findings = [];
  const name = pkg.name?.toLowerCase() || '';

  const typosquatPatterns = [
    /^(lodash|lodahs|lodash-js|lodashexe)$/,
    /^(axios|axio|ax10s|ax1os)$/,
    /^(react|reakt|reackt|r3act)$/,
    /^(express|expres|expresjs|exress)$/,
    /^(vue|vu3|vujs|vuejs)$/,
    /^(webpack|webpak|webpackjs)$/,
  ];

  for (const pattern of typosquatPatterns) {
    if (pattern.test(name)) {
      findings.push({
        id: 'ATK-007', severity: 'high',
        title: 'Typosquat detected',
        description: `Package name "${pkg.name}" is similar to popular packages`,
        evidence: `similar to ${pattern.source}`
      });
    }
  }

  return findings;
}

export function analyzeDependencyGraph(lockfileData) {
  const findings = [];
  const pkgMap = new Map();
  for (const pkg of lockfileData.packages) { pkgMap.set(pkg.name, pkg); }

  for (const pkg of lockfileData.packages) {
    // ATK-011: Worm propagation via peer deps containing 'plugin', 'hook', 'ext'
    if (pkg.peerDeps && Object.keys(pkg.peerDeps).length > 0) {
      for (const [peerName, peerVersion] of Object.entries(pkg.peerDeps)) {
        if (peerName.includes('plugin') || peerName.includes('hook') || peerName.includes('ext')) {
          findings.push({ id: 'ATK-011', severity: 'high', ... });
        }
      }
    }
    // ATK-011: Excessive transitive dependencies
    if (pkg.dependencies && Object.keys(pkg.dependencies).length > 5) {
      const transitiveCount = Object.keys(pkg.dependencies).filter(k => k.includes('/')).length;
      if (transitiveCount > 3) {
        findings.push({ id: 'ATK-011', severity: 'medium', ... });
      }
    }
    // ATK-011: Excessive optional dependencies (>10)
    if (pkg.optionalDependencies && Object.keys(pkg.optionalDependencies).length > 10) {
      findings.push({ id: 'ATK-011', severity: 'low', ... });
    }
  }
  return findings;
}

export function generateLockfileReport(lockfileData) {
  // Runs checkMaliciousPatterns on each package + analyzeDependencyGraph
  // Returns { scanId, package, version, totalDependencies, devDependencies, optionalDependencies, lockfileVersion, findings, riskScore }
}

function calculateRiskScore(findings) {
  // weights: critical=10, high=7, medium=4, low=2, info=0.5
  // maxSeverity + min(count * 0.3, 3), capped at 10
}
```
