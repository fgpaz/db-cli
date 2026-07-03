package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCatalogSupportsMixedEngines(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()

	mustWriteFile(t, filepath.Join(home, ".db-cli", "connections.yaml"), []byte(`version: 1
connections:
  home-postgres:
    engine: postgres
    dsn_env: HOME_POSTGRES_DSN
    default_database: app
    write_policy: never
  home-sqlserver:
    engine: sqlserver
    dsn_env: HOME_SQLSERVER_DSN
    default_database: reports
    write_policy: explicit
`))

	mustWriteFile(t, filepath.Join(repo, "infra", "db-cli.connections.yaml"), []byte(`version: 1
connections:
  repo-postgres:
    engine: postgres
    dsn_env: REPO_POSTGRES_DSN
    default_database: app
    vector: auto
    write_policy: explicit
`))

	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	catalog, err := LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}

	if got := len(catalog.SourceFiles); got != 2 {
		t.Fatalf("expected 2 source files, got %d", got)
	}

	repoCfg, ok := catalog.Get("repo-postgres")
	if !ok {
		t.Fatalf("repo-postgres not found")
	}
	if repoCfg.Engine != EnginePostgres {
		t.Fatalf("repo-postgres engine = %s, want postgres", repoCfg.Engine)
	}
	if repoCfg.VectorMode != VectorAuto {
		t.Fatalf("repo-postgres vector mode = %s, want auto", repoCfg.VectorMode)
	}
	if repoCfg.DefaultSchema != "public" {
		t.Fatalf("repo-postgres default schema = %s, want public", repoCfg.DefaultSchema)
	}

	sqlCfg, ok := catalog.Get("home-sqlserver")
	if !ok {
		t.Fatalf("home-sqlserver not found")
	}
	if sqlCfg.Engine != EngineSQLServer {
		t.Fatalf("home-sqlserver engine = %s, want sqlserver", sqlCfg.Engine)
	}
	if sqlCfg.DefaultSchema != "dbo" {
		t.Fatalf("home-sqlserver default schema = %s, want dbo", sqlCfg.DefaultSchema)
	}
	if sqlCfg.VectorMode != "" {
		t.Fatalf("home-sqlserver vector mode = %s, want empty", sqlCfg.VectorMode)
	}
}

func TestLoadCatalogRejectsVectorOnSqlServer(t *testing.T) {
	home := t.TempDir()
	mustWriteFile(t, filepath.Join(home, ".db-cli", "connections.yaml"), []byte(`version: 1
connections:
  bad-sqlserver:
    engine: sqlserver
    dsn_env: BAD_SQLSERVER_DSN
    vector: auto
`))

	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	if _, err := LoadCatalog(); err == nil {
		t.Fatal("LoadCatalog() error = nil, want error")
	}
}

func TestResolveDSNBuildsFromSplitEnv(t *testing.T) {
	t.Setenv("BS_POSTGRES_HOST", "localhost")
	t.Setenv("BS_POSTGRES_PORT", "5432")
	t.Setenv("BS_POSTGRES_DB", "buho_salud_dev")
	t.Setenv("BS_POSTGRES_USER", "postgres")
	t.Setenv("BS_POSTGRES_PASSWORD", "postgres")

	dsn, err := resolveDSN(ConnectionConfig{
		Name:        "core-net-api-read",
		Engine:      EnginePostgres,
		HostEnv:     "BS_POSTGRES_HOST",
		PortEnv:     "BS_POSTGRES_PORT",
		DatabaseEnv: "BS_POSTGRES_DB",
		UserEnv:     "BS_POSTGRES_USER",
		PasswordEnv: "BS_POSTGRES_PASSWORD",
	})
	if err != nil {
		t.Fatalf("resolveDSN() error = %v", err)
	}
	if want := "host=localhost port=5432 dbname=buho_salud_dev user=postgres password=postgres pool_max_conns=4 connect_timeout=15"; dsn != want {
		t.Fatalf("resolveDSN() = %q, want %q", dsn, want)
	}
}

func TestResolveDSNBuildsSQLServerURLFromSplitEnv(t *testing.T) {
	t.Setenv("REPORTING_SQLSERVER_HOST", "localhost")
	t.Setenv("REPORTING_SQLSERVER_PORT", "1433")
	t.Setenv("REPORTING_SQLSERVER_DB", "reports")
	t.Setenv("REPORTING_SQLSERVER_USER", "sa")
	t.Setenv("REPORTING_SQLSERVER_PASSWORD", "secret")

	dsn, err := resolveDSN(ConnectionConfig{
		Name:              "reporting-read",
		Engine:            EngineSQLServer,
		HostEnv:           "REPORTING_SQLSERVER_HOST",
		PortEnv:           "REPORTING_SQLSERVER_PORT",
		DatabaseEnv:       "REPORTING_SQLSERVER_DB",
		UserEnv:           "REPORTING_SQLSERVER_USER",
		PasswordEnv:       "REPORTING_SQLSERVER_PASSWORD",
		ApplicationIntent: "ReadOnly",
	})
	if err != nil {
		t.Fatalf("resolveDSN() error = %v", err)
	}

	for _, fragment := range []string{
		"sqlserver://sa:secret@localhost:1433",
		"database=reports",
		"ApplicationIntent=ReadOnly",
	} {
		if !strings.Contains(dsn, fragment) {
			t.Fatalf("resolveDSN() = %q, want fragment %q", dsn, fragment)
		}
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
