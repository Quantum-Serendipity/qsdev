<!-- Source: https://offlinetools.org/a/json-formatter/schema-versioning-for-json-configuration-files -->
<!-- Retrieved: 2026-05-12 -->

# JSON Configuration Schema Versioning: Best Practices

## Version Field Placement

The recommended approach is straightforward: "put a `schemaVersion` field in every config, validate each version explicitly, and migrate older files into one latest in-memory representation." This field should be:

- **Root-level and document-bound**: Embed the version inside the JSON itself rather than relying solely on file names like `config.v2.json`
- **Separate from application version**: Keep your binary release number distinct from the config schema version, since one application release may support multiple schema versions

## Breaking vs. Non-Breaking Changes

Bump the schema version when:
- Removing, renaming, or relocating required fields
- Changing field types or meanings
- Tightening validation rules enough to reject previously valid files
- Modifying defaults that materially affect runtime behavior

Do *not* bump for:
- Adding optional fields (if older readers safely ignore unknown properties)
- Non-structural fixes

Use simple integer versioning (1, 2, 3) for most configuration files rather than semantic versioning.

## Migration Strategy

Implement incremental, step-by-step migrations:
- Design `v1 -> v2 -> v3` migration functions rather than special-case conversions from every old version to the newest
- Normalize to a single internal representation that the rest of your application consumes
- Establish an explicit support window defining how many previous versions your loader accepts

## Version-Aware Loading Pattern

A typical loader follows this pipeline:

1. Parse JSON
2. Detect `schemaVersion`
3. Validate against that specific version's schema
4. Apply sequential migrations to reach the latest format
5. Hand the normalized object to application code

## Forward/Backward Compatibility

**Rollout order prevents breakage**: Deploy readers capable of handling the new schema *before* writers begin emitting it.

Additional safeguards include logging deprecation warnings well in advance, testing round-trip migrations with real configurations, and ensuring error messages are human-friendly for manual edits.

## Schema Validation

Use JSON Schema per version, keeping one schema file per configuration version. Modern JSON Schema draft 2020-12 is recommended unless your validator targets an older standard. Note that `$schema` (which specifies the JSON Schema dialect) differs from `schemaVersion` (your document format identifier).
