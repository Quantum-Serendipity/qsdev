# Database Migration

Create safe, reversible database schema migrations.

## Planning

1. Understand the desired schema change: new table, add/drop column, add index,
   alter type, rename, etc.
2. Check the current schema state by reading existing migration files or
   querying the database schema if accessible.
3. Determine the migration framework in use:
   - Go: `goose`, `golang-migrate`, `atlas`
   - Python: `alembic`, `Django migrations`
   - JavaScript/TypeScript: `knex`, `prisma migrate`, `typeorm`
4. Create a new migration file following the framework's naming convention
   (typically timestamp-prefixed or sequential).

## Writing the UP migration

- Write the forward migration that applies the schema change.
- Use `IF NOT EXISTS` / `IF EXISTS` guards where the dialect supports them
  to make the migration idempotent.
- Add indexes for every new foreign key column. Missing indexes on foreign
  keys cause full table scans on joins and deletes.
- For column additions with `NOT NULL`, provide a `DEFAULT` value or
  backfill existing rows in the same migration.
- For column type changes, consider whether a data conversion is needed.
  If so, add the conversion logic to the migration, not application code.
- For table renames, update all foreign key references in the same migration.

## Writing the DOWN migration

- Write the reverse migration that undoes the UP step exactly.
- For `CREATE TABLE`, the down is `DROP TABLE`.
- For `ADD COLUMN`, the down is `DROP COLUMN` (note: this loses data).
- For data-destructive operations, add a comment explaining that down
  migration will cause data loss and may not be suitable for production
  rollback.
- If a migration is truly irreversible (e.g., dropping a column with
  important data), document this explicitly and raise a warning.

## Data preservation

- Never drop columns or tables that contain production data without an
  explicit backup step or a multi-phase migration plan:
  1. Phase 1: Stop writing to the old column. Deploy application changes.
  2. Phase 2: Migrate data to the new location.
  3. Phase 3: Drop the old column in a subsequent migration.
- For large tables, consider online schema change tools (`pt-online-schema-change`,
  `gh-ost`, `pg_repack`) to avoid locking.

## Verification

1. Apply the migration to a test database: run the UP migration.
2. Verify the schema matches expectations.
3. Run the DOWN migration and verify the schema reverts cleanly.
4. Run the UP migration again to confirm idempotency.
5. Run the application's test suite against the migrated schema.

## Output format

Provide the complete migration file(s) with:
- Appropriate file naming for the framework
- Both UP and DOWN sections
- Comments explaining non-obvious decisions
- Any required data backfill queries
