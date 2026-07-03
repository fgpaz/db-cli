# Safe Usage

## Read First

- Start by reviewing the project's architecture document so you pick the correct service/database pair before touching data.
- Then inspect the nearest `infra/.env` and prefer env-based aliases over inline DSNs.
- If the credentials in `infra/.env` are missing or stale, reconcile them first and update the file before running `probe`, `query`, or `write`.
- Use `query` for inspections and checks.
- Use `write` only when the change is intentional and explicit.
- Prefer `query` with `EXPLAIN` before a write if you need to understand the impact.

## Write Guardrails

- Require `--reason` on every write.
- Require `--confirm <alias>` to match the target connection.
- Keep v1 writes to simple single-statement DML.
- Reject DDL, `COPY`, and multi-statement bodies.
- Respect `write_policy=never` as a hard block.

## Parallel Use

- `query-many` is for read-only fan-out.
- Keep concurrency modest unless the target databases are known to tolerate more load.
- Do not point many agents at the same write target unless the change is coordinated.

## pgvector

- Use `capabilities` to detect whether the `vector` extension is present.
- Treat `vector=auto` as a capability check, not a promise.
- Do not assume SQL Server aliases support `vector`.
