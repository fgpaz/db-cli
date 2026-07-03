# Instalar db-cli

Pega este prompt en tu agente (Codex, Claude Code, OpenCode, etc) para instalar y configurar db-cli automaticamente.

---

## Tu tarea

1. Detectar la plataforma del usuario (Linux/Mac/Windows, amd64/arm64)
2. Instalar Go 1.23+ si no esta
3. Clonar https://github.com/fgpaz/db-cli.git y compilar el binario
4. Mover el binario a un directorio en PATH
5. Crear ~/.db-cli/connections.yaml con la configuracion que el usuario indique
6. Setear variables de entorno
7. Probar la conexion con db-cli probe
8. Explicar los comandos basicos

## Pasos a ejecutar

### Paso 1: Detectar plataforma

```bash
uname -m 2>/dev/null || echo %PROCESSOR_ARCHITECTURE%
```

Mapeo:
- x86_64 / amd64 -> amd64
- aarch64 / arm64 / ARM64 -> arm64

### Paso 2: Instalar Go (si no existe)

```bash
go version 2>/dev/null || {
  echo "Instalando Go..."
  # Linux
  curl -sL https://go.dev/dl/go1.23.4.linux-amd64.tar.gz | sudo tar -C /usr/local -xz
  export PATH=$PATH:/usr/local/go/bin
  # Mac: brew install go
  # Windows: winget install GoLang.Go
}
```

### Paso 3: Compilar db-cli

```bash
git clone https://github.com/fgpaz/db-cli.git /tmp/db-cli-build
cd /tmp/db-cli-build
go build -o db-cli .
```

### Paso 4: Instalar binario

```bash
# Linux/Mac:
sudo mv /tmp/db-cli-build/db-cli /usr/local/bin/db-cli
chmod +x /usr/local/bin/db-cli

# Windows (PowerShell):
# New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\db-cli"
# Copy-Item db-cli.exe "$env:LOCALAPPDATA\db-cli\db-cli.exe"
```

### Paso 5: Verificar

```bash
db-cli doctor
```

### Paso 6: Configurar conexiones

Preguntar al usuario:
- Que motor usa? (postgres / sqlserver)
- Host, puerto, base de datos, usuario
- Es solo lectura o tambien escribe?

Crear ~/.db-cli/connections.yaml:

**PostgreSQL con env vars (recomendado):**
```yaml
version: 1
connections:
  mi-app:
    engine: postgres
    host_env: PGHOST
    port_env: PGPORT
    database_env: PGDATABASE
    user_env: PGUSER
    password_env: PGPASSWORD
    default_database: miapp
    default_schema: public
    vector: auto
    write_policy: never
```

**SQL Server con DSN:**
```yaml
version: 1
connections:
  mi-db:
    engine: sqlserver
    dsn_env: DB_DSN
    default_database: miapp
    default_schema: dbo
    write_policy: never
```

### Paso 7: Setear variables de entorno

Preguntar al usuario sus credenciales y ayudarlas a setear:

```bash
# PostgreSQL
export PGHOST=localhost
export PGPORT=5432
export PGDATABASE=miapp
export PGUSER=tu_usuario
export PGPASSWORD=tu_password

# SQL Server
export DB_DSN="sqlserver://user:pass@host:1433?database=miapp"
```

Explicar que puede guardarlas en ~/.bashrc, ~/.zshrc, o un archivo .env en su proyecto.

### Paso 8: Probar

```bash
db-cli probe --conn mi-app
db-cli query --conn mi-app --sql "SELECT 1 as test"
```

## Comandos para enseñar al usuario

```bash
db-cli doctor                          # Diagnostico completo
db-cli catalog list                    # Ver conexiones configuradas
db-cli probe --conn mi-app             # Testear conexion
db-cli capabilities --conn mi-app      # Ver version del servidor y pgvector
db-cli query --conn mi-app --sql "SELECT * FROM usuarios LIMIT 10"
db-cli discover                        # Auto-detectar conexiones en .env
```

## Links utiles

- Repo: https://github.com/fgpaz/db-cli
- Release con binarios: https://github.com/fgpaz/db-cli/releases
- Docs: https://github.com/fgpaz/db-cli/tree/main/docs
