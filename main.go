package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		writeJSON(map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "db-cli",
		Short:         "Database operations CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newCatalogCommand())
	root.AddCommand(newDoctorCommand())
	root.AddCommand(newProbeCommand())
	root.AddCommand(newCapabilitiesCommand())
	root.AddCommand(newQueryCommand())
	root.AddCommand(newQueryManyCommand())
	root.AddCommand(newWriteCommand())
	root.AddCommand(newMigrateCommand())
	root.AddCommand(newDiscoverCommand())
	return root
}

func newCatalogCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "catalog",
		Short:         "Inspect configured connections",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(&cobra.Command{
		Use:           "list",
		Short:         "List connections",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			items := make([]ConnSummary, 0, len(catalog.Connections))
			for _, name := range catalog.Names() {
				cfg, _ := catalog.Get(name)
				items = append(items, summarizeConnection(cfg))
			}
			writeJSON(map[string]any{
				"success":     true,
				"operation":   "catalog list",
				"connections": items,
				"sourceFiles": catalog.SourceFiles,
			})
			return nil
		},
	})

	var conn string
	show := &cobra.Command{
		Use:           "show",
		Short:         "Show a connection",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if conn == "" && len(args) > 0 {
				conn = args[0]
			}
			if conn == "" {
				return errors.New("--conn is required")
			}
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			writeJSON(map[string]any{
				"success":     true,
				"operation":   "catalog show",
				"connection":  summarizeConnection(cfg),
				"sourceFiles": catalog.SourceFiles,
			})
			return nil
		},
	}
	show.Flags().StringVar(&conn, "conn", "", "connection alias")
	cmd.AddCommand(show)
	return cmd
}

func newDoctorCommand() *cobra.Command {
	var conn string
	cmd := &cobra.Command{
		Use:           "doctor",
		Short:         "Check environment and catalog",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			res := DoctorResult{
				Success:    true,
				Operation:  "doctor",
				Catalog:    make([]ConnSummary, 0, len(catalog.Connections)),
				DurationMs: time.Since(start).Milliseconds(),
			}
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				res.HomeConfig = filepath.Join(home, ".db-cli", "connections.yaml")
			}
			for _, src := range catalog.SourceFiles {
				if strings.Contains(src, string(os.PathSeparator)+"infra"+string(os.PathSeparator)) {
					res.Overlay = src
				}
			}
			for _, name := range catalog.Names() {
				cfg, _ := catalog.Get(name)
				res.Catalog = append(res.Catalog, summarizeConnection(cfg))
			}
			if conn != "" {
				cfg, ok := catalog.Get(conn)
				if !ok {
					return fmt.Errorf("connection %q not found", conn)
				}
				probe, err := probeConnection(context.Background(), cfg)
				if err != nil {
					return err
				}
				res.Target = probe
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	return cmd
}

func newProbeCommand() *cobra.Command {
	var conn string
	cmd := &cobra.Command{
		Use:           "probe",
		Short:         "Test connectivity and authentication",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			res, err := probeConnection(context.Background(), cfg)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	_ = cmd.MarkFlagRequired("conn")
	return cmd
}

func newCapabilitiesCommand() *cobra.Command {
	var conn string
	cmd := &cobra.Command{
		Use:           "capabilities",
		Short:         "Inspect server capabilities",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			res, err := detectCapabilities(context.Background(), cfg)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	_ = cmd.MarkFlagRequired("conn")
	return cmd
}

func newQueryCommand() *cobra.Command {
	var conn, sqlText, file string
	cmd := &cobra.Command{
		Use:           "query",
		Short:         "Run read-only SQL",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			sql, err := loadSQL(sqlText, file)
			if err != nil {
				return err
			}
			if err := ValidateSQL(sql, SQLModeQuery); err != nil {
				return err
			}
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			res, err := executeQuery(context.Background(), cfg, sql)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	cmd.Flags().StringVar(&sqlText, "sql", "", "sql text")
	cmd.Flags().StringVar(&file, "file", "", "sql file")
	_ = cmd.MarkFlagRequired("conn")
	return cmd
}

func newQueryManyCommand() *cobra.Command {
	var conns, sqlText, file string
	var concurrency int
	cmd := &cobra.Command{
		Use:           "query-many",
		Short:         "Run read-only SQL across many connections",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if conns == "" {
					conns = strings.Join(args, " ")
				} else {
					conns = strings.TrimSpace(conns + " " + strings.Join(args, " "))
				}
			}
			sql, err := loadSQL(sqlText, file)
			if err != nil {
				return err
			}
			if err := ValidateSQL(sql, SQLModeQuery); err != nil {
				return err
			}
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			aliases := splitAliases(conns)
			res, err := queryMany(context.Background(), catalog, aliases, sql, concurrency)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conns, "conns", "", "comma separated aliases")
	cmd.Flags().StringVar(&sqlText, "sql", "", "sql text")
	cmd.Flags().StringVar(&file, "file", "", "sql file")
	cmd.Flags().IntVar(&concurrency, "concurrency", 4, "max concurrent connections")
	_ = cmd.MarkFlagRequired("conns")
	return cmd
}

func newWriteCommand() *cobra.Command {
	var conn, sqlText, file, reason, confirm string
	var expectRows int64
	var dryRun bool
	cmd := &cobra.Command{
		Use:           "write",
		Short:         "Run explicit SQL writes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			sql, err := loadSQL(sqlText, file)
			if err != nil {
				return err
			}
			if conn != confirm {
				return errors.New("--confirm must match --conn")
			}
			if reason == "" {
				return errors.New("--reason is required")
			}
			if err := ValidateSQL(sql, SQLModeWrite); err != nil {
				return err
			}
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			res, err := executeWrite(context.Background(), cfg, sql, dryRun, reason, expectRows)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	cmd.Flags().StringVar(&sqlText, "sql", "", "sql text")
	cmd.Flags().StringVar(&file, "file", "", "sql file")
	cmd.Flags().StringVar(&reason, "reason", "", "write reason")
	cmd.Flags().StringVar(&confirm, "confirm", "", "confirmation alias")
	cmd.Flags().Int64Var(&expectRows, "expect-rows", 0, "expected rows affected")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate without applying")
	_ = cmd.MarkFlagRequired("conn")
	_ = cmd.MarkFlagRequired("confirm")
	_ = cmd.MarkFlagRequired("reason")
	return cmd
}

func loadSQL(sqlText, file string) (string, error) {
	switch {
	case sqlText != "" && file != "":
		return "", errors.New("use either --sql or --file, not both")
	case sqlText != "":
		return sqlText, nil
	case file != "":
		data, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		return string(data), nil
	default:
		return "", errors.New("either --sql or --file is required")
	}
}

func newMigrateCommand() *cobra.Command {
	var conn, sqlText, file, reason, confirm string
	var dryRun bool
	cmd := &cobra.Command{
		Use:           "migrate",
		Short:         "Execute DDL and DML migrations",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			sql, err := loadSQL(sqlText, file)
			if err != nil {
				return err
			}
			if conn != confirm {
				return errors.New("--confirm must match --conn")
			}
			if reason == "" {
				return errors.New("--reason is required")
			}
			if err := ValidateSQL(sql, SQLModeMigrate); err != nil {
				return err
			}
			catalog, err := LoadCatalog()
			if err != nil {
				return err
			}
			cfg, ok := catalog.Get(conn)
			if !ok {
				return fmt.Errorf("connection %q not found", conn)
			}
			res, err := executeMigrate(context.Background(), cfg, sql, dryRun, reason)
			if err != nil {
				return err
			}
			writeJSON(res)
			return nil
		},
	}
	cmd.Flags().StringVar(&conn, "conn", "", "connection alias")
	cmd.Flags().StringVar(&sqlText, "sql", "", "sql text")
	cmd.Flags().StringVar(&file, "file", "", "sql file")
	cmd.Flags().StringVar(&reason, "reason", "", "migration reason")
	cmd.Flags().StringVar(&confirm, "confirm", "", "confirmation alias")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate without applying")
	_ = cmd.MarkFlagRequired("conn")
	_ = cmd.MarkFlagRequired("confirm")
	_ = cmd.MarkFlagRequired("reason")
	return cmd
}

func newDiscoverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "discover",
		Short:         "Auto-discover database connections in the current project",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			found := DiscoverProjectConnections()
			writeJSON(map[string]any{
				"success":     true,
				"operation":   "discover",
				"connections": found,
				"count":       len(found),
			})
			return nil
		},
	}
	return cmd
}

func splitAliases(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || unicode.IsSpace(r)
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}
