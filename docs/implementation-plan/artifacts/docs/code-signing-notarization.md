<!-- Sources: https://artyom.dev/notarizing-go-binaries-for-macos.md https://textslashplain.com/2025/03/12/authenticode-in-2025-azure-trusted-signing/ https://goreleaser.com/customization/sign/ -->
<!-- Retrieved: 2026-05-12 -->

# Code Signing and Notarization for Go Binaries

## macOS Notarization

### Requirements
- Apple Developer Account ($99/year)
- Xcode 11+ installed
- Developer ID Application certificate

### Process
1. Create Developer ID certificate via Xcode
2. Sign binary: `codesign -s <CERTIFICATE-ID> -o runtime -v <BINARY>`
3. Submit for notarization: `xcrun notarytool submit <FILE> --apple-id <EMAIL> --password <APP-PASSWORD> --team-id <TEAM-ID> --wait`
4. Apple reviews (minutes to hours)

### Tools
- `gon` by Mitchell Hashimoto - automates signing + notarization for Go binaries
- GoReleaser can integrate with gon via signing hooks

## Windows Authenticode (Azure Trusted Signing)

### Pricing
- $9.99/month via Azure Artifact Signing (formerly Trusted Signing)
- Available to individuals and enterprises (US, Canada, EU, UK)

### Critical: Timestamping
- Azure certificates expire in 3 days
- MUST timestamp signatures or they expire with the certificate
- With proper timestamp, signatures valid indefinitely

### Process
1. Create Azure account + Trusted Signing resource
2. Complete identity verification
3. Install latest SignTool + DLIB
4. Sign: `signtool sign /dlib <dlib-path> /dmdf <config-json> /td SHA256 /tr http://timestamp.acs.microsoft.com /v <BINARY>`

## GoReleaser Signing Config

```yaml
signs:
  - artifacts: checksum
    cmd: gpg
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sig"
      - "${artifact}"
```
