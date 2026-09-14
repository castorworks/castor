# Database migrations

Castor applies schema changes only from the explicit `database.Migrate` runner
during `init-db`. Applied versions are recorded in the PostgreSQL
`schema_migrations` table and protected by the transaction-scoped advisory lock
in `bootstrap.Initialize`.

The initial version adopts the existing GORM model schema once. Existing
databases without `schema_migrations` are reconciled once and recorded as a
legacy baseline; subsequent deployments never run a recurring `AutoMigrate`.

## Adding a schema change

1. Add a new entry to `apps/api/internal/infrastructure/database/migration.go`.
2. Give it the next monotonically increasing version and a descriptive name.
3. Implement the change explicitly with SQL or GORM operations inside the
   migration function. Do not modify the initial migration.
4. Test the migration against a disposable PostgreSQL database, including a
   fresh database and the previous schema version.
5. Deploy the migration with `init-db` before rolling out the application image.

Every migration must be backward-compatible with the currently running image
when a rolling deployment is used. Destructive changes require a maintenance
window, a verified backup, and a separately reviewed data migration. Rollback
means restoring a database-compatible application image; schema downgrades
are intentionally not attempted automatically.

The database initializer is deliberately separate from normal API startup.
Normal startup only connects to the already initialized schema.
