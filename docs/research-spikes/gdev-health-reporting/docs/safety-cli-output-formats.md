<!-- Source: https://docs.safetycli.com/safety-2/safety-cli-2-scanner/output-formats -->
<!-- Retrieved: 2026-05-12 -->

# Safety CLI Output Formats

## Supported Formats

1. **Screen** (default): prints the results to the screen with command-line formatting
2. **Text**: Same as screen output without any command line formatting, for file saving
3. **JSON**: Structured data format requiring an API key
4. **Bare**: Very basic (bare) output is a simplified version of the JSON output
5. **HTML5**: Available in versions >2.3.5, generates HTML5 formatted reports

## JSON Output Structure

The JSON response contains these key sections:

- `report_meta` -- scan metadata (timestamps, scan targets, counts)
- `scanned_packages` -- array of detected packages and versions
- `affected_packages` -- packages with relevant vulnerabilities
- `vulnerabilities` -- vulnerability array related to scanned packages
- `ignored_vulnerabilities` -- vulnerabilities excluded via policy or CLI arguments
- `remediations` -- fix recommendations per vulnerable package
- `announcements` -- Safety team messages and version announcements

## Command Examples

```
safety check --output json --key <YOUR-API-KEY>
safety check --output html --key <YOUR-API-KEY> --save-html output.html
```

## Notable Features

The documentation does not mention SBOM or SARIF output formats. The `SAFETY_COLOR` environment variable controls styling across all outputs when set to `False` or `0`.
