# Contribuir a db-cli

Gracias por tu interés en contribuir. Cualquier aporte es bienvenido.

## Requisitos

- **Go 1.23** o superior
- **Git** para control de versiones
- Acceso a al menos una base PostgreSQL o SQL Server para pruebas locales

## Primeros pasos

1. Hacé un fork del repositorio
2. Cloná tu fork localmente:

```powershell
git clone https://github.com/<tu-usuario>/db-cli.git
cd db-cli
```

3. Creá una rama para tu cambio:

```powershell
git checkout -b feature/<descripcion-corta>
# o
git checkout -b fix/<descripcion-corta>
```

## Desarrollo

### Compilar

```powershell
go build -o db-cli.exe
```

### Ejecutar pruebas

```powershell
# Todas las pruebas
go test ./...

# Con coverage
go test -cover ./...

# Pruebas con race detector
go test -race ./...
```

### Pruebas con bases de datos

Si tenés PostgreSQL y/o SQL Server disponibles localmente, podés correr las pruebas de integración:

```powershell
# Setear las vars de entorno para las pruebas
$env:DB_CLI_TEST_PG = "host=localhost port=5432 user=test password=test dbname=testdb sslmode=disable"
$env:DB_CLI_TEST_SQLSERVER = "sqlserver://localhost:1433?database=testdb&user.id=test&user.password=test"

# Correr pruebas de integración
go test ./internal/engine/... -tags=integration
```

## Flujo de trabajo

1. **Hacé tu cambio** en tu rama
2. **Asegurate de que las pruebas pasen**: `go test ./...`
3. **Documentá** si agregaste una nueva funcionalidad (actualizá el README si corresponde)
4. **Hacé commit** con un mensaje descriptivo:

```powershell
git commit -m "feat: agregar soporte para conexión con SSL en PostgreSQL"
```

5. **Subí tu rama** y abrí un Pull Request

## Pautas para Pull Requests

- **Un cambio por PR**: mantené cada PR enfocado en una sola funcionalidad o corrección
- **Mensajes de commit** siguiendo [Conventional Commits](https://www.conventionalcommits.org/):
  - `feat:` nueva funcionalidad
  - `fix:` corrección de bug
  - `docs:` solo documentación
  - `refactor:` refactor sin cambio de comportamiento
  - `test:` agregar o corregir pruebas
- **Probá tu código** antes de enviarlo
- **Actualizá el README** si agregaste un nuevo comando o cambiaste el comportamiento existente
- **Referenciá issues** relacionados en la descripción del PR

## Estructura de carpetas

```
db-cli/
├── cmd/              # Subcomandos cobra
├── internal/         # Código interno del paquete
│   ├── catalog/      # Gestión del catálogo YAML
│   ├── discovery/    # Auto-descubrimiento de conexiones
│   ├── engine/       # Drivers de base de datos
│   └── guard/        # SQL guard y validación
├── pkg/              # APIs públicas
└── go.mod
```

## Código de Conducta

Este proyecto adopta el [Código de Conducta Contributor Covenant](https://www.contributor-covenant.org/).
Al participar, se espera que respetes este código. Reportá conductas inapropiadas a [fgpaz](https://github.com/fgpaz).

## ¿Necesitás ayuda?

- Abrí un [issue](https://github.com/fgpaz/db-cli/issues) con la etiqueta `question`
- Revisá los issues existentes antes de crear uno nuevo

¡Gracias por contribuir!
