<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/backend/detectors/ (multiple files) -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — Detector Source Code

## backend/detectors/index.js

```javascript
import * as atk001 from './atk-001-lifecycle.js';
import * as atk002 from './atk-002-obfusc.js';
import * as atk003 from './atk-003-creds.js';
import * as atk004 from './atk-004-persist.js';
import * as atk005 from './atk-005-exfil.js';
import * as atk006 from './atk-006-depconf.js';
import * as atk007 from './atk-007-typosquat.js';
import * as atk008 from './atk-008-tarball-tamper.js';
import * as atk009 from './atk-009-dormant-trigger.js';
import * as atk010 from './atk-010-sandbox-evasion.js';
import * as atk011 from './atk-011-transitive-prop.js';

export async function runAll(pkgJson, files = []) {
  const findings = [];
  findings.push(...await atk001.scan(pkgJson, files));
  findings.push(...await atk002.scan(pkgJson, files));
  findings.push(...await atk003.scan(pkgJson, files));
  findings.push(...await atk004.scan(pkgJson, files));
  findings.push(...await atk005.scan(pkgJson, files));
  findings.push(...await atk006.scan(pkgJson, files));
  findings.push(...await atk007.scan(pkgJson, files));
  findings.push(...await atk008.scan(pkgJson, files));
  findings.push(...await atk009.scan(pkgJson, files));
  findings.push(...await atk010.scan(pkgJson, files));
  findings.push(...await atk011.scan(pkgJson, files));
  return findings.sort((a, b) => b.severity.localeCompare(a.severity));
}
```

## backend/detectors/atk-001-lifecycle.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const scripts = pkgJson.scripts || {};
  const suspicious = Object.keys(scripts).filter(s => /pre|post|install/i.test(s));
  if (suspicious.length) {
    const content = suspicious.map(s => scripts[s]).join(' ');
    if (/curl|wget|sh |bash |\.sh|exfil|steal|pwn|c2|pastebin/i.test(content)) {
      findings.push({
        id: 'ATK-001',
        severity: 'high',
        title: 'Malicious lifecycle scripts',
        description: 'Suspicious install hooks',
        evidence: suspicious.join(', ')
      });
    }
  }
  return findings;
}
```

## backend/detectors/atk-002-obfusc.js (SUBSTANTIAL — ~200 lines)

Sophisticated multi-pass obfuscation detection:
- Maintains allowlists: DIST_BUILD_PATTERNS, TEST_FIXTURE_PATTERNS, KNOWN_SAFE_DOMAINS
- Helper functions: extractUrlDomain, isDistOrBuild, isTestOrFixture, isKnownSafeDomain, locateLine, decodePreview, detectEncodingType, isFileInLifecycleScript, isLikelyLifecycleFileName, createEvidence
- Detection layers:
  1. eval + hex/base64 decode detection
  2. Double-encoded nested payload detection
  3. Decode + network (fetch/curl/http) combined detection
  4. String.fromCharCode obfuscation + eval
  5. Shell-code patterns: env-eval, exec-buffer, function-eval, require-eval, strict-eval
- Returns rich evidence objects with file path, line number, encoding type, decoded preview, destination host, lifecycle hook context
- Context-aware: distinguishes dist/build files, test fixtures, known safe domains

## backend/detectors/atk-003-creds.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const code = files.map(f => f.content).join('\n');
  if (/process\.env\.(NPM_TOKEN|GIT_TOKEN|AWS_SECRET|AWS_ACCESS|SSH_KEY)|\.npmrc|\.ssh\/id_rsa|readFile.*\.ssh/.test(code)) {
    findings.push({
      id: 'ATK-003',
      severity: 'high',
      title: 'Credential harvesting',
      description: 'Env vars or .npmrc/SSH key access',
      evidence: 'credential pattern match'
    });
  }
  return findings;
}
```

## backend/detectors/atk-005-exfil.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const code = files.map(f => f.content).join('\n');
  if (/curl.*(-d|--data|--data-binary)|github\.com\/.*keys|pastebin|dns\.resolve.*\.com|exfil/.test(code.toLowerCase())) {
    findings.push({
      id: 'ATK-005',
      severity: 'critical',
      title: 'Network exfiltration',
      description: 'Suspicious network calls: curl data exfil, pastebin, dns tunneling',
      evidence: 'network exfil pattern'
    });
  }
  return findings;
}
```

## backend/detectors/atk-009-dormant-trigger.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const code = files.map(f => f.content).join('\n');

  // CI environment detection patterns
  const ciPatterns = [
    { pattern: /process\.env\.CI\b/, label: 'CI env check' },
    { pattern: /process\.env\.(TRAVIS|CIRCLECI|GITHUB_ACTIONS|JENKINS|GITLAB_CI|CODEBUILD)/, label: 'CI platform check' },
    { pattern: /\bisCI\b/, label: 'isCI utility check' },
  ];

  for (const { pattern, label } of ciPatterns) {
    if (pattern.test(code)) {
      findings.push({
        id: 'ATK-009',
        severity: 'high',
        title: 'Conditional trigger (CI/production env)',
        description: `Package checks for CI or production environment: ${label}`,
        evidence: 'conditional trigger detected'
      });
      break;
    }
  }

  // Time-based activation patterns (severity elevated if combined with suspicious behavior)
  const suspiciousCode = /\beval\(|atob\(|btoa\(|new Function\(|child_process\b|\.exec\(|spawn\(/;
  const suspiciousNetwork = /\.fetch\(|http\.request\(|https\.request\(|dns\.lookup\(/;
  const suspiciousEnv = /process\.env\.(?!NODE_ENV)[A-Z_]{4,}/;
  const hasSuspicious = suspiciousCode.test(code) || suspiciousNetwork.test(code) || suspiciousEnv.test(code);

  const timePatterns = [
    { pattern: /new Date\(\)\s*[><=!]+\s*new Date\(['"]\d{4}/, label: 'time-based activation' },
    { pattern: /\bDate\.now\(\)\s*[><=!]+.*(?:eval|fetch|exec|write|crypto|env\.CI)/i, label: 'timestamp check with suspicious behavior' },
    { pattern: /\bsetTimeout\s*\([^)]*,\s*(?!0\b)[1-9]\d{3,}/, label: 'long-delay execution (>1000ms)' },
    { pattern: /\bDate\(\)\b.*(?:exec|eval|fetch|write|crypto)/i, label: 'date check with suspicious behavior' },
  ];

  for (const { pattern, label } of timePatterns) {
    if (pattern.test(code)) {
      findings.push({
        id: 'ATK-009',
        severity: hasSuspicious ? 'high' : 'medium',
        title: 'Conditional trigger (time-based)',
        description: `Package uses ${label}`,
        evidence: `${label}${hasSuspicious ? ' — elevated (suspicious context)' : ''}`
      });
      break;
    }
  }

  return findings;
}
```
