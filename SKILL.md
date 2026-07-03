---
name: db-cli
description: Use when working with configured database connections from Codex, especially when you need to query several PostgreSQL or SQL Server databases, detect pgvector support, inspect connection capabilities, perform explicit write operations, or execute DDL migrations through the db-cli CLI in Windows or Linux workflows.
---

# DB CLI

## Overview

`db-cli` is a multiplatform CLI skill for inspecting, querying, writing, and migrating PostgreSQL and SQL Server databases. Use it when one agent needs a fast local workflow across many aliases without an MCP hop.

## Project Connection Workflow

When a repository exposes architectural and environment context, use this order before selecting an alias:

1. Run `discover` first to auto-detect connections from `appsettings*.json` and `.env` files.
2. If no connections found, review the project's architecture document to identify the target database.
3. Inspect the nearest `infra/.env` for credentials or use the global catalog.
4. Prefer project overlay aliases in `infra/db-cli.connections.yaml` that reference env vars.
5. If credentials are missing, reconcile with the source of truth before using `probe`, `query`, `write`, or `migrate`.

## Entry Points

- Windows PowerShell: `scripts/db-cli.ps1`
- Windows CMD: `scripts/db-cli.cmd`
- Linux shell: `scripts/db-cli.sh`
- Binaries: `bin/windows-amd64/`, `bin/windows-arm64/`, `bin/linux-amd64/`, `bin/linux-arm64/`

## Commands

### Discovery

- `discover` - Auto-detect database connections from project files (appsettings*.json, .env, infra/.env)

### Read-only

- `catalog list`
- `catalog show --conn <alias>`
- `doctor [--conn <alias>]`
- `probe --conn <alias>`
- `capabilities --conn <alias>`
- `query --conn <alias> --sql "<sql>"` or `--file <path>`
- `query-many --conns a,b,c --sql "<sql>" --concurrency 4`

### Explicit writes (DML only)

- `write --conn <alias> --sql "<sql>" --reason "<why>" --confirm <alias>`
- `write` rejects aliases with `write_policy=never`
- `write` is for simple DML only; no DDL, no `COPY`, no multi-statement

### Migrations (DDL + DML)

- `migrate --conn <alias> --file <path> --reason "<why>" --confirm <alias> [--dry-run]`
- `migrate` allows DDL (CREATE TABLE, ALTER TABLE, DROP, etc.) and multi-statement SQL
- `migrate` requires `write_policy=explicit` on the connection
- Use `--dry-run` to validate without executing

## Configuration

- Global catalog: `%USERPROFILE%\.db-cli\connections.yaml`
- Project overlay: `infra\db-cli.connections.yaml`
- Use `engine: postgres|sqlserver`
- Connection credentials can be specified three ways:
  1. **Inline fields**: `host`, `port`, `database`, `user`, `password` (simplest, for local/dev use)
  2. **Env var references**: `host_env`, `port_env`, `database_env`, `user_env`, `password_env` (preferred for secrets)
  3. **DSN**: `dsn` (full connection string) or `dsn_env` (env var with full DSN)
- Use `vector: auto|on|off` only on PostgreSQL connections
- `write_policy`: `never` (default, read-only) or `explicit` (allows write and migrate)
- Wrappers auto-load the nearest `infra\.env` before invoking the binary

See [catalog format](references/catalog.md) and [safe usage](references/safe-usage.md).

## Publish

- Build native Windows binaries with `scripts\build.ps1`
- Build native Linux binaries with `scripts/build.sh`
- Build every supported target with `scripts\build.ps1 -AllTargets` or `scripts/build.sh --all`
- Publish with `scripts\publish.ps1`
- `publish.ps1` validates the skill, rebuilds all targets, and syncs to global install and mirror at `C:\repos\buho\assets\skills\db-cli`

## Common mistakes

- Treating `query` as a generic SQL runner instead of a read-only command.
- Using `write` without `--reason` or without matching `--confirm`.
- Using `write` for DDL operations - use `migrate` instead.
- Putting secrets directly in the repo catalog instead of env references.
- Setting `vector` on SQL Server aliases.
- Forgetting to run `discover` first when entering a new project.
