package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/microsoft/go-mssqldb"
)

type OperationError struct {
	Success   bool   `json:"success"`
	Operation string `json:"operation"`
	Conn      string `json:"conn,omitempty"`
	Error     string `json:"error"`
}

type QueryResult struct {
	Success    bool             `json:"success"`
	Operation  string           `json:"operation"`
	Engine     string           `json:"engine"`
	Conn       string           `json:"conn"`
	Columns    []string         `json:"columns"`
	Rows       []map[string]any `json:"rows"`
	RowCount   int64            `json:"rowCount"`
	CommandTag string           `json:"commandTag,omitempty"`
	DurationMs int64            `json:"durationMs"`
	Error      string           `json:"error,omitempty"`
}

type ProbeResult struct {
	Success       bool   `json:"success"`
	Operation     string `json:"operation"`
	Engine        string `json:"engine"`
	Conn          string `json:"conn"`
	Reachable     bool   `json:"reachable"`
	Authenticated bool   `json:"authenticated"`
	Database      string `json:"database,omitempty"`
	User          string `json:"user,omitempty"`
	ServerVersion string `json:"serverVersion,omitempty"`
	Vector        bool   `json:"vector"`
	DurationMs    int64  `json:"durationMs"`
	Error         string `json:"error,omitempty"`
}

type CapabilityResult struct {
	Success       bool   `json:"success"`
	Operation     string `json:"operation"`
	Engine        string `json:"engine"`
	Conn          string `json:"conn"`
	ServerVersion string `json:"serverVersion,omitempty"`
	ServerMajor   int    `json:"serverMajor,omitempty"`
	ServerMinor   int    `json:"serverMinor,omitempty"`
	Vector        bool   `json:"vector"`
	DefaultSchema string `json:"defaultSchema,omitempty"`
	WritePolicy   string `json:"writePolicy,omitempty"`
	VectorMode    string `json:"vectorMode,omitempty"`
	DurationMs    int64  `json:"durationMs"`
	Error         string `json:"error,omitempty"`
}

type QueryManyItem struct {
	Conn   string      `json:"conn"`
	Result QueryResult `json:"result"`
}

type QueryManyResult struct {
	Success    bool            `json:"success"`
	Operation  string          `json:"operation"`
	Results    []QueryManyItem `json:"results"`
	DurationMs int64           `json:"durationMs"`
}

type WriteResult struct {
	Success      bool   `json:"success"`
	Operation    string `json:"operation"`
	Engine       string `json:"engine"`
	Conn         string `json:"conn"`
	DryRun       bool   `json:"dryRun"`
	Reason       string `json:"reason,omitempty"`
	ExpectedRows *int64 `json:"expectedRows,omitempty"`
	Applied      bool   `json:"applied"`
	RowCount     int64  `json:"rowCount,omitempty"`
	CommandTag   string `json:"commandTag,omitempty"`
	DurationMs   int64  `json:"durationMs"`
	Error        string `json:"error,omitempty"`
}

type MigrateResult struct {
	Success    bool   `json:"success"`
	Operation  string `json:"operation"`
	Engine     string `json:"engine"`
	Conn       string `json:"conn"`
	DryRun     bool   `json:"dryRun"`
	Reason     string `json:"reason,omitempty"`
	Applied    bool   `json:"applied"`
	Statements int    `json:"statements"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

type ConnSummary struct {
	Name           string   `json:"name"`
	Engine         string   `json:"engine"`
	DefaultDB      string   `json:"defaultDatabase,omitempty"`
	DefaultSchema  string   `json:"defaultSchema,omitempty"`
	VectorMode     string   `json:"vectorMode,omitempty"`
	SupportsVector bool     `json:"supportsVector"`
	WritePolicy    string   `json:"writePolicy"`
	Tags           []string `json:"tags,omitempty"`
	EnvRef         string   `json:"envRef,omitempty"`
	HasDSN         bool     `json:"hasDSN"`
}

type DoctorResult struct {
	Success    bool          `json:"success"`
	Operation  string        `json:"operation"`
	HomeConfig string        `json:"homeConfig,omitempty"`
	Overlay    string        `json:"overlay,omitempty"`
	Catalog    []ConnSummary `json:"catalog"`
	Target     *ProbeResult  `json:"target,omitempty"`
	DurationMs int64         `json:"durationMs"`
	Error      string        `json:"error,omitempty"`
}

func summarizeConnection(cfg ConnectionConfig) ConnSummary {
	return ConnSummary{
		Name:           cfg.Name,
		Engine:         cfg.Engine.String(),
		DefaultDB:      cfg.DefaultDB,
		DefaultSchema:  cfg.DefaultSchema,
		VectorMode:     string(cfg.VectorMode),
		SupportsVector: cfg.Engine.IsVectorCapable(),
		WritePolicy:    string(cfg.WritePolicy),
		Tags:           append([]string(nil), cfg.Tags...),
		EnvRef:         cfg.EnvRef,
		HasDSN:         cfg.DSN != "",
	}
}

func probeConnection(ctx context.Context, cfg ConnectionConfig) (*ProbeResult, error) {
	start := time.Now()
	res := &ProbeResult{Success: true, Operation: "probe", Engine: cfg.Engine.String(), Conn: cfg.Name}

	switch cfg.Engine {
	case EnginePostgres:
		pool, err := openPostgres(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()

		if err := pool.Ping(ctx); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		if err := pool.QueryRow(ctx, `select current_database(), current_user, current_setting('server_version'), exists(select 1 from pg_extension where extname = 'vector')`).Scan(&res.Database, &res.User, &res.ServerVersion, &res.Vector); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
	case EngineSQLServer:
		db, err := openSQLServer(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		var edition string
		if err := db.PingContext(ctx); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		if err := db.QueryRowContext(
			ctx,
			`select db_name(), system_user, cast(serverproperty('ProductVersion') as nvarchar(128)), cast(serverproperty('Edition') as nvarchar(128))`,
		).Scan(&res.Database, &res.User, &res.ServerVersion, &edition); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		res.Vector = false
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	res.Reachable = true
	res.Authenticated = true
	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func detectCapabilities(ctx context.Context, cfg ConnectionConfig) (*CapabilityResult, error) {
	start := time.Now()
	res := &CapabilityResult{
		Success:       true,
		Operation:     "capabilities",
		Engine:        cfg.Engine.String(),
		Conn:          cfg.Name,
		DefaultSchema: cfg.DefaultSchema,
		WritePolicy:   string(cfg.WritePolicy),
		VectorMode:    string(cfg.VectorMode),
	}

	switch cfg.Engine {
	case EnginePostgres:
		pool, err := openPostgres(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()

		var versionNum int
		if err := pool.QueryRow(ctx, `select current_setting('server_version_num')::int`).Scan(&versionNum); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		if err := pool.QueryRow(ctx, `select exists(select 1 from pg_extension where extname = 'vector')`).Scan(&res.Vector); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		res.ServerMajor = versionNum / 10000
		res.ServerMinor = (versionNum % 10000) / 100
		res.ServerVersion = fmt.Sprintf("%d.%d", res.ServerMajor, res.ServerMinor)
	case EngineSQLServer:
		db, err := openSQLServer(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		var version string
		var edition string
		if err := db.QueryRowContext(
			ctx,
			`select cast(serverproperty('ProductVersion') as nvarchar(128)), cast(serverproperty('Edition') as nvarchar(128))`,
		).Scan(&version, &edition); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		major, minor := parseSQLServerVersion(version)
		res.ServerVersion = version
		res.ServerMajor = major
		res.ServerMinor = minor
		res.Vector = false
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func executeQuery(ctx context.Context, cfg ConnectionConfig, sqlText string) (*QueryResult, error) {
	start := time.Now()
	switch cfg.Engine {
	case EnginePostgres:
		pool, err := openPostgres(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()

		tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		rows, err := tx.Query(ctx, sqlText)
		if err != nil {
			return &QueryResult{
				Success:    false,
				Operation:  "query",
				Engine:     cfg.Engine.String(),
				Conn:       cfg.Name,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      err.Error(),
			}, nil
		}
		defer rows.Close()

		descs := rows.FieldDescriptions()
		columns := make([]string, 0, len(descs))
		for _, desc := range descs {
			columns = append(columns, string(desc.Name))
		}

		out := make([]map[string]any, 0)
		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				return nil, err
			}
			row := make(map[string]any, len(columns))
			for i, col := range columns {
				row[col] = normalizeValue(values[i])
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		tag := rows.CommandTag()
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &QueryResult{
			Success:    true,
			Operation:  "query",
			Engine:     cfg.Engine.String(),
			Conn:       cfg.Name,
			Columns:    columns,
			Rows:       out,
			RowCount:   int64(len(out)),
			CommandTag: tag.String(),
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	case EngineSQLServer:
		db, err := openSQLServer(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		rows, err := db.QueryContext(ctx, sqlText)
		if err != nil {
			return &QueryResult{
				Success:    false,
				Operation:  "query",
				Engine:     cfg.Engine.String(),
				Conn:       cfg.Name,
				DurationMs: time.Since(start).Milliseconds(),
				Error:      err.Error(),
			}, nil
		}
		defer rows.Close()

		columns, out, err := scanSQLRows(rows)
		if err != nil {
			return nil, err
		}
		return &QueryResult{
			Success:    true,
			Operation:  "query",
			Engine:     cfg.Engine.String(),
			Conn:       cfg.Name,
			Columns:    columns,
			Rows:       out,
			RowCount:   int64(len(out)),
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

func executeWrite(ctx context.Context, cfg ConnectionConfig, sqlText string, dryRun bool, reason string, expectedRows int64) (*WriteResult, error) {
	if cfg.WritePolicy == WritePolicyNever {
		return nil, fmt.Errorf("connection %s does not allow writes", cfg.Name)
	}

	res := &WriteResult{
		Success:   true,
		Operation: "write",
		Engine:    cfg.Engine.String(),
		Conn:      cfg.Name,
		DryRun:    dryRun,
		Reason:    reason,
		Applied:   !dryRun,
	}

	start := time.Now()
	if dryRun {
		if expectedRows > 0 {
			res.ExpectedRows = &expectedRows
		}
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	switch cfg.Engine {
	case EnginePostgres:
		pool, err := openPostgres(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()

		tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite})
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		defer tx.Rollback(ctx)

		tag, err := tx.Exec(ctx, sqlText)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		if err := tx.Commit(ctx); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		res.RowCount = tag.RowsAffected()
		res.CommandTag = tag.String()
	case EngineSQLServer:
		db, err := openSQLServer(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		defer tx.Rollback()

		result, err := tx.ExecContext(ctx, sqlText)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		if err := tx.Commit(); err != nil {
			res.Success = false
			res.Error = err.Error()
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
		rowsAffected, err := result.RowsAffected()
		if err == nil {
			res.RowCount = rowsAffected
		}
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	res.DurationMs = time.Since(start).Milliseconds()
	if expectedRows > 0 {
		res.ExpectedRows = &expectedRows
	}
	return res, nil
}

func queryMany(ctx context.Context, catalog *Catalog, names []string, sqlText string, concurrency int) (*QueryManyResult, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("at least one alias is required")
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 16 {
		concurrency = 16
	}
	start := time.Now()
	type item struct {
		idx int
		res QueryResult
	}
	jobs := make(chan int)
	results := make(chan item, len(names))
	for i := 0; i < concurrency; i++ {
		go func() {
			for idx := range jobs {
				name := names[idx]
				cfg, ok := catalog.Get(name)
				if !ok {
					results <- item{idx: idx, res: QueryResult{Success: false, Operation: "query-many", Conn: name, Error: "connection not found"}}
					continue
				}
				out, err := executeQuery(ctx, cfg, sqlText)
				if err != nil {
					results <- item{idx: idx, res: QueryResult{Success: false, Operation: "query-many", Engine: cfg.Engine.String(), Conn: name, Error: err.Error()}}
					continue
				}
				out.Operation = "query-many"
				results <- item{idx: idx, res: *out}
			}
		}()
	}
	go func() {
		for i := range names {
			jobs <- i
		}
		close(jobs)
	}()
	out := make([]QueryManyItem, len(names))
	success := true
	for i := 0; i < len(names); i++ {
		item := <-results
		out[item.idx] = QueryManyItem{Conn: item.res.Conn, Result: item.res}
		if !item.res.Success {
			success = false
		}
	}
	return &QueryManyResult{
		Success:    success,
		Operation:  "query-many",
		Results:    out,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func openPostgres(ctx context.Context, cfg ConnectionConfig) (*pgxpool.Pool, error) {
	dsn, err := resolveDSN(cfg)
	if err != nil {
		return nil, err
	}
	pool, err := connectPostgres(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	return pgxpool.NewWithConfig(ctx, cfg)
}

func openSQLServer(ctx context.Context, cfg ConnectionConfig) (*sql.DB, error) {
	dsn, err := resolveDSN(cfg)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func scanSQLRows(rows *sql.Rows) ([]string, []map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	values := make([]any, len(columns))
	scanTargets := make([]any, len(columns))
	for i := range values {
		scanTargets[i] = &values[i]
	}

	out := make([]map[string]any, 0)
	for rows.Next() {
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, nil, err
		}
		row := make(map[string]any, len(columns))
		for i, col := range columns {
			row[col] = normalizeValue(values[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return columns, out, nil
}

func normalizeValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return v
	}
}

func parseSQLServerVersion(version string) (int, int) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	return major, minor
}

func executeMigrate(ctx context.Context, cfg ConnectionConfig, sqlText string, dryRun bool, reason string) (*MigrateResult, error) {
	if cfg.WritePolicy == WritePolicyNever {
		return nil, fmt.Errorf("connection %s does not allow writes (write_policy=never)", cfg.Name)
	}

	res := &MigrateResult{
		Success:   true,
		Operation: "migrate",
		Engine:    cfg.Engine.String(),
		Conn:      cfg.Name,
		DryRun:    dryRun,
		Reason:    reason,
		Applied:   !dryRun,
	}

	res.Statements = countStatements(sqlText)

	start := time.Now()
	if dryRun {
		res.DurationMs = time.Since(start).Milliseconds()
		return res, nil
	}

	switch cfg.Engine {
	case EnginePostgres:
		pool, err := openPostgres(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer pool.Close()

		_, err = pool.Exec(ctx, sqlText)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.Applied = false
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
	case EngineSQLServer:
		db, err := openSQLServer(ctx, cfg)
		if err != nil {
			return nil, err
		}
		defer db.Close()

		_, err = db.ExecContext(ctx, sqlText)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.Applied = false
			res.DurationMs = time.Since(start).Milliseconds()
			return res, nil
		}
	default:
		return nil, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	res.DurationMs = time.Since(start).Milliseconds()
	return res, nil
}

func countStatements(sql string) int {
	count := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == ';' && !inSingle && !inDouble {
			count++
		}
	}
	trimmed := strings.TrimSpace(sql)
	if trimmed != "" && (count == 0 || !strings.HasSuffix(trimmed, ";")) {
		count++
	}
	return count
}
