<!-- Source: https://raw.githubusercontent.com/lateos-ai/npm-scan/main/backend/detectors/ (atk-004, atk-010) -->
<!-- Retrieved: 2026-05-15 -->

# @lateos/npm-scan — Additional Detector Source Code

## backend/detectors/atk-004-persist.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const code = files.map(f => f.content).join('\n');
  if (/mkdir.*(\.vscode|\.claude|\.cursor)/.test(code)) {
    findings.push({
      id: 'ATK-004',
      severity: 'high',
      title: 'Persistence via editor configs',
      description: 'Creates .vscode/.claude/.cursor dirs',
      evidence: 'mkdir pattern match'
    });
  }
  return findings;
}
```

## backend/detectors/atk-010-sandbox-evasion.js

```javascript
export async function scan(pkgJson, files = []) {
  const findings = [];
  const code = files.map(f => f.content).join('\n');

  // High-severity patterns
  const highPatterns = [
    { pattern: /\bdebugger\s*;?(\s*\/\/|\s*$|\)|\])/m, label: 'debugger statement' },
    { pattern: /process\.argv.*['"]--inspect['"]|process\.argv.*\binspect\b(?!.*argv)/, label: 'inspect/debug flag detection' },
    { pattern: /hostname.*(?:docker|sandbox|container|vmware|vbox)/i, label: 'anti-sandbox hostname check' },
    { pattern: /detect.*(?:sandbox|debugger|analysis|virtual)/i, label: 'explicit evasion probe' },
    { pattern: /e\.stack\b.*(?:sandbox|docker|container|vmware)/i, label: 'stack trace sandbox probe' },
  ];

  for (const { pattern, label } of highPatterns) {
    if (pattern.test(code)) {
      findings.push({
        id: 'ATK-010', severity: 'high',
        title: 'Sandbox evasion / anti-analysis',
        description: `Package performs anti-analysis behavior: ${label}`,
        evidence: 'evasion pattern detected'
      });
      break;
    }
  }

  // Medium-severity: multiple system fingerprinting APIs
  if (findings.length === 0) {
    const multiApi = ['process.pid', 'process.ppid', 'os.hostname', 'os.cpus', 'process.arch']
      .filter(api => code.includes(api));
    if (multiApi.length >= 3) {
      findings.push({
        id: 'ATK-010', severity: 'medium',
        title: 'Sandbox evasion / anti-analysis',
        description: 'Multiple system fingerprinting APIs detected',
        evidence: `${multiApi.length} fingerprinting APIs: ${multiApi.join(', ')}`
      });
    }
  }

  // Medium-severity: stack trace + code execution
  const multiStack = ['Error().stack', 'new Error().stack'].filter(s => code.includes(s));
  if (multiStack.length > 0 && /atob|eval|execSync|spawn|child_process/.test(code)) {
    findings.push({
      id: 'ATK-010', severity: 'medium',
      title: 'Sandbox evasion / anti-analysis',
      description: 'Stack trace capture combined with code execution',
      evidence: 'stack trace + execution'
    });
  }

  return findings;
}
```
