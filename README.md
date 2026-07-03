# db-cli

> Conectate a PostgreSQL y SQL Server directamente desde la terminal. Sin servidores MCP, sin capas extra, sin magia.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](LICENSE)
[![Windows](https://img.shields.io/badge/-Windows-0078D6?style=for-the-badge&logo=windows)](#)
[![Linux](https://img.shields.io/badge/-Linux-333?style=for-the-badge&logo=linux)](#)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql)](#)
[![SQL Server](https://img.shields.io/badge/SQL%20Server-CC2927?style=for-the-badge&logo=microsoft-sql-server)](#)

---

## Por que CLI y no MCP?

Los servidores MCP (Model Context Protocol) agregan una capa de complejidad innecesaria para acceso a bases de datos:

- Un proceso extra que levantar y mantener
- Configuracion adicional de transporte (stdio o HTTP)
- Serializacion/deserializacion entre el cliente y el servidor
- Latencia anadida en cada consulta

**db-cli** hace lo mismo, pero directo: ejecutas el comando, te conectas, obtienes resultados JSON para integracion con agentes, tablas legibles para humanos.

Sin servidores. Sin capas. Sin excusas.

## Caracteristicas

- **Catalogo YAML**: definis todas tus conexiones en un solo archivo, con aliases
- **Auto-descubrimiento**: detecta conexiones de `appsettings*.json` y `.env` automaticamente
- **PostgreSQL + SQL Server**: los dos motores mas usados, en un solo binario
- **pgvector**: detecta automaticamente si tu base Postgres tiene extensiones vectoriales
- **Consultas paralelas**: ejecuta la misma query en multiples bases a la vez (`query-many`)
- **Escritura protegida**: confirmacion explicita antes de cualquier write
- **SQL Guard**: bloquea operaciones peligrosas en modos query/write
- **DDL migrations**: ejecuta scripts de migracion con dry-run
- **JSON output**: salida estructurada para integracion con agentes y pipelines

## Instalacion

### Un comando (skill)

```bash
codex skill install --repo fgpaz/db-cli
```

### Binario directo (sin Go)

Descarga el zip de tu plataforma desde la [release v1.0.0](https://github.com/fgpaz/db-cli/releases/tag/v1.0.0), descomprime y listo:

```bash
# Linux
unzip db-cli_linux_amd64.zip
chmod +x db-cli
sudo mv db-cli /usr/local/bin/

# Windows: descomprime y usa directamente
.\db-cli.exe catalog list
```

### Desde fuente (requiere Go 1.23+)

```bash
git clone https://github.com/fgpaz/db-cli.git
cd db-cli
go build -o db-cli.exe .
```


### Usar los wrappers (sin compilar)

Si ya tenes los binarios en `bin/`, usa los wrappers:

```powershell
# Windows PowerShell
.\scripts\db-cli.ps1 catalog list

# Linux
./scripts/db-cli.sh catalog list

# Windows CMD
.\scripts\db-cli.cmd catalog list
```

Los wrappers buscan automaticamente un archivo `infra/.env` cercano y cargan las variables de entorno.

## Configuracion

### Catalogo de conexiones

Crea un archivo `~/.db-cli/connections.yaml` (catalogo global) o `infra/db-cli.connections.yaml` (overlay del proyecto):

```yaml
version: 1
connections:
  my-app-read:
    engine: postgres
    host_env: PGHOST
    port_env: PGPORT
    database_env: PGDATABASE
    user_env: PGUSER
    password_env: PGPASSWORD
    default_database: myapp
    default_schema: public
    vector: auto
    write_policy: never
    tags:
      - local
      - read-only

  my-app-write:
    engine: postgres
    host_env: PGHOST
    port_env: PGPORT
    database_env: PGDATABASE
    user_env: PGUSER
    password_env: PGPASSWORD
    default_database: myapp
    write_policy: explicit
    tags:
      - local
      - writable

  legacy-reporting:
    engine: sqlserver
    dsn_env: REPORTING_DSN
    default_database: reports
    default_schema: dbo
    application_intent: ReadOnly
    write_policy: never
```

**Nunca pongas credenciales inline en el catalogo.** Usa variables de entorno (`_env` suffix) o DSN desde env vars (`dsn_env`).

### Auto-descubrimiento

```bash
# Detecta conexiones en appsettings*.json y .env del proyecto actual
db-cli discover
```

## Uso

### Listar conexiones del catalogo

```bash
db-cli catalog list
```

### Ver detalle de una conexion

```bash
db-cli catalog show my-app-read
```

### Probar conectividad

```bash
db-cli probe --conn my-app-read
```

### Diagnosticar entorno

```bash
db-cli doctor
db-cli doctor --conn my-app-read  # incluye probe de la conexion
```

### Inspeccionar capacidades del servidor

```bash
db-cli capabilities --conn my-app-read
```

### Ejecutar consulta (solo lectura)

```bash
db-cli query --conn my-app-read --sql "SELECT COUNT(*) FROM users WHERE created_at > '2025-01-01'"

# Desde un archivo SQL
db-cli query --conn my-app-read --file ./queries/active_users.sql
```

### Escribir datos (con confirmacion explicita)

```bash
db-cli write \
  --conn my-app-write \
  --sql "INSERT INTO logs (message, level) VALUES ('deploy ok', 'info')" \
  --reason "Adding deployment log entry" \
  --confirm my-app-write
```

El flag `--confirm` debe coincidir exactamente con `--conn`. Esto evita escrituras accidentales.

### Ejecutar migraciones DDL

```bash
# Dry-run primero (valida sin ejecutar)
db-cli migrate \
  --conn my-app-write \
  --file ./migrations/001_create_users.sql \
  --reason "Create users table" \
  --confirm my-app-write \
  --dry-run

# Aplicar
db-cli migrate \
  --conn my-app-write \
  --file ./migrations/001_create_users.sql \
  --reason "Create users table" \
  --confirm my-app-write
```

### Consultas paralelas a multiples bases

```bash
db-cli query-many \
  --conns "my-app-read,my-app-write" \
  --sql "SELECT version()" \
  --concurrency 4
```

## Seguridad

### Guardrail de escritura

- `write` requiere `--reason` y `--confirm <alias>` (debe coincidir con `--conn`)
- `write` solo permite INSERT, UPDATE, DELETE simples (una sola sentencia)
- Conexiones con `write_policy: never` bloquean escritura absolutamente
- `migrate` solo funciona con `write_policy: explicit`

### SQL Guard

En modos `query` y `write`, el SQL guard bloquea automaticamente:

- COPY, ALTER, CREATE, DROP, TRUNCATE
- GRANT, REVOKE
- VACUUM, LISTEN, NOTIFY
- CLUSTER, REFRESH, MERGE
- EXECUTE, CALL
- Multi-sentencias (v1)
- SELECT INTO en modo query

En modo `migrate` se permiten todas las operaciones (DDL + DML).

### Variables de entorno para credenciales

Usa `host_env`, `port_env`, `database_env`, `user_env`, `password_env` para construir el DSN desde env vars, o `dsn_env` para un connection string completo:

```yaml
connections:
  prod-db:
    engine: postgres
    host_env: DB_HOST
    port_env: DB_PORT
    database_env: DB_NAME
    user_env: DB_USER
    password_env: DB_PASS
```

## Estructura del proyecto

```
db-cli/
├── main.go              # Entry point, comandos cobra
├── config.go            # Catalogo YAML, resolucion de DSN, discovery
├── db.go                # Core de DB: probe, query, write, migrate, query-many
├── json.go              # Helper de salida JSON
├── sqlguard.go          # Validador SQL por modo
├── *_test.go            # Tests unitarios
├── go.mod / go.sum      # Dependencias Go
├── scripts/
│   ├── db-cli.ps1       # Wrapper Windows PowerShell
│   ├── db-cli.sh        # Wrapper Linux
│   ├── db-cli.cmd       # Wrapper Windows CMD
│   ├── build.ps1        # Build multiplataforma (Windows)
│   └── build.sh         # Build multiplataforma (Linux)
├── examples/
│   └── connections.example.yaml
├── docs/
│   ├── catalog.md       # Formato del catalogo
│   └── safe-usage.md    # Guia de uso seguro
├── .github/
│   └── workflows/ci.yml # CI: build + test
├── Makefile             # Targets: build, test, lint
├── README.md
├── CONTRIBUTING.md
└── LICENSE
```

## Desarrollo

```bash
# Compilar
make build

# Test
make test

# Build multiplataforma (windows/linux x amd64/arm64)
make build-all

# Build con Go directo
go build -o db-cli.exe .
go test ./...
```

## Contribuir

Las contribuciones son bienvenidas. Lee [CONTRIBUTING.md](CONTRIBUTING.md) para comenzar.

## Licencia

MIT — ver [LICENSE](LICENSE) para detalles.

---

**Hecho con ❤️ en Argentina** — [fg paz](https://github.com/fgpaz)




