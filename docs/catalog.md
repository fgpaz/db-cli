# Catalog Format

Use one catalog for aliases and keep secrets outside the repo.

## Locations

- Global: `%USERPROFILE%\.db-cli\connections.yaml`
- Project overlay: `infra\db-cli.connections.yaml`

## Schema

```yaml
version: 1
connections:
  app-read:
    engine: postgres
    host_env: BS_POSTGRES_HOST
    port_env: BS_POSTGRES_PORT
    database_env: BS_POSTGRES_DB
    user_env: BS_POSTGRES_USER
    password_env: BS_POSTGRES_PASSWORD
    default_database: app
    default_schema: public
    vector: auto
    write_policy: never
    tags: [prod, read-only]

  report-write:
    engine: sqlserver
    dsn_env: DB_CLI_REPORT_SQLSERVER_RW_DSN
    default_database: reports
    default_schema: dbo
    write_policy: explicit
    tags: [prod, writable]
```

## Fields

- `engine`: `postgres` or `sqlserver`.
- alias key: stable alias used by the CLI.
- `dsn_env`: environment variable that contains the connection string.
- `dsn`: optional inline connection string for throwaway local use only.
- `host_env`, `port_env`, `database_env`, `user_env`, `password_env`: optional env-based DSN builder fields when the project exposes split credentials in `infra/.env`.
- `application_intent`: optional SQL Server hint such as `ReadOnly` when the alias is built from split env vars.
- `default_database`: optional default database name for prompts and validation.
- `default_schema`: optional schema hint.
- `vector`: `auto`, `on`, or `off` for PostgreSQL only.
- `write_policy`: `never` or `explicit`.
- `tags`: optional labels for filtering and grouping.

## Overlay Rule

The project overlay can add aliases or override non-secret defaults. The global catalog remains the primary home for reusable connections.

For repositories like `buho/salud`, prefer this order before you add or use aliases:

1. Review `.docs/wiki/02_arquitectura.md` to identify which service owns the target database.
2. Inspect `infra/.env` for the live connection variables or DSN source.
3. Add or update `infra/db-cli.connections.yaml` with env-based aliases instead of copying secrets into the repo.
4. If the expected credentials are missing or stale, reconcile them with the source of truth and update `infra/.env` before using the alias.
