package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Engine string
type VectorMode string
type WritePolicy string

const (
	EnginePostgres  Engine = "postgres"
	EngineSQLServer Engine = "sqlserver"

	VectorAuto VectorMode = "auto"
	VectorOn   VectorMode = "on"
	VectorOff  VectorMode = "off"

	WritePolicyNever    WritePolicy = "never"
	WritePolicyExplicit WritePolicy = "explicit"
)

type ConnectionConfig struct {
	Name              string      `yaml:"-" json:"name"`
	Engine            Engine      `yaml:"engine" json:"engine"`
	DSN               string      `yaml:"dsn,omitempty" json:"dsn,omitempty"`
	EnvRef            string      `yaml:"dsn_env,omitempty" json:"envRef,omitempty"`
	HostEnv           string      `yaml:"host_env,omitempty" json:"hostEnv,omitempty"`
	PortEnv           string      `yaml:"port_env,omitempty" json:"portEnv,omitempty"`
	DatabaseEnv       string      `yaml:"database_env,omitempty" json:"databaseEnv,omitempty"`
	UserEnv           string      `yaml:"user_env,omitempty" json:"userEnv,omitempty"`
	PasswordEnv       string      `yaml:"password_env,omitempty" json:"passwordEnv,omitempty"`
	Host              string      `yaml:"host,omitempty" json:"host,omitempty"`
	Port              string      `yaml:"port,omitempty" json:"port,omitempty"`
	Database          string      `yaml:"database,omitempty" json:"database,omitempty"`
	User              string      `yaml:"user,omitempty" json:"user,omitempty"`
	Password          string      `yaml:"password,omitempty" json:"password,omitempty"`
	ApplicationIntent string      `yaml:"application_intent,omitempty" json:"applicationIntent,omitempty"`
	DefaultDB         string      `yaml:"default_database,omitempty" json:"defaultDatabase,omitempty"`
	DefaultSchema     string      `yaml:"default_schema,omitempty" json:"defaultSchema,omitempty"`
	VectorMode        VectorMode  `yaml:"vector,omitempty" json:"vectorMode,omitempty"`
	WritePolicy       WritePolicy `yaml:"write_policy,omitempty" json:"writePolicy,omitempty"`
	Tags              []string    `yaml:"tags,omitempty" json:"tags,omitempty"`
	Notes             string      `yaml:"notes,omitempty" json:"notes,omitempty"`
}

type CatalogFile struct {
	Connections map[string]ConnectionConfig `yaml:"connections"`
}

type Catalog struct {
	SourceFiles []string
	Connections map[string]ConnectionConfig
}

func LoadCatalog() (*Catalog, error) {
	merged := map[string]ConnectionConfig{}
	sources := make([]string, 0)

	for _, path := range catalogCandidateFiles() {
		file, err := loadCatalogFile(path)
		if err != nil {
			return nil, err
		}
		if len(file.Connections) == 0 {
			continue
		}
		sources = append(sources, path)
		for name, cfg := range file.Connections {
			cfg.Name = name
			normalized, err := normalizeConnection(cfg)
			if err != nil {
				return nil, fmt.Errorf("connection %q in %s: %w", name, path, err)
			}
			merged[name] = normalized
		}
	}

	return &Catalog{SourceFiles: sources, Connections: merged}, nil
}

func loadCatalogFile(path string) (*CatalogFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &CatalogFile{Connections: map[string]ConnectionConfig{}}, nil
		}
		return nil, fmt.Errorf("read catalog %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &CatalogFile{Connections: map[string]ConnectionConfig{}}, nil
	}
	var file CatalogFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse catalog %s: %w", path, err)
	}
	if file.Connections == nil {
		file.Connections = map[string]ConnectionConfig{}
	}
	return &file, nil
}

func catalogCandidateFiles() []string {
	files := []string{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		files = append(files, filepath.Join(home, ".db-cli", "connections.yaml"))
	}
	if overlay := findProjectOverlay(); overlay != "" {
		files = append(files, overlay)
	}
	return files
}

func findProjectOverlay() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(wd, "infra", "db-cli.connections.yaml")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return ""
		}
		wd = parent
	}
}

func normalizeConnection(cfg ConnectionConfig) (ConnectionConfig, error) {
	if cfg.Engine == "" {
		return ConnectionConfig{}, fmt.Errorf("engine is required")
	}
	switch cfg.Engine {
	case EnginePostgres:
		if cfg.DefaultSchema == "" {
			cfg.DefaultSchema = "public"
		}
		if cfg.VectorMode == "" {
			cfg.VectorMode = VectorAuto
		}
	case EngineSQLServer:
		if cfg.DefaultSchema == "" {
			cfg.DefaultSchema = "dbo"
		}
		if cfg.VectorMode != "" && cfg.VectorMode != VectorOff {
			return ConnectionConfig{}, fmt.Errorf("vector is only supported for postgres connections")
		}
		cfg.VectorMode = ""
	default:
		return ConnectionConfig{}, fmt.Errorf("unsupported engine %q", cfg.Engine)
	}

	if cfg.WritePolicy == "" {
		cfg.WritePolicy = WritePolicyNever
	}
	switch cfg.WritePolicy {
	case WritePolicyNever, WritePolicyExplicit:
	default:
		return ConnectionConfig{}, fmt.Errorf("unsupported write policy %q", cfg.WritePolicy)
	}

	if cfg.DSN == "" && cfg.EnvRef == "" && !cfg.HasStructuredEnvConfig() && !cfg.HasInlineConfig() {
		return ConnectionConfig{}, fmt.Errorf("connection has no dsn, dsn_env, env-based builder fields, or inline host/database/user/password")
	}

	return cfg, nil
}

func (c ConnectionConfig) HasStructuredEnvConfig() bool {
	return c.HostEnv != "" && c.DatabaseEnv != "" && c.UserEnv != "" && c.PasswordEnv != ""
}

func (c ConnectionConfig) HasInlineConfig() bool {
	return c.Host != "" && c.Database != "" && c.User != "" && c.Password != ""
}

func (c *Catalog) Names() []string {
	names := make([]string, 0, len(c.Connections))
	for name := range c.Connections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Catalog) Get(name string) (ConnectionConfig, bool) {
	cfg, ok := c.Connections[name]
	return cfg, ok
}

func resolveDSN(cfg ConnectionConfig) (string, error) {
	if cfg.DSN != "" {
		return cfg.DSN, nil
	}
	if cfg.EnvRef != "" {
		if value := os.Getenv(cfg.EnvRef); value != "" {
			return value, nil
		}
		return "", fmt.Errorf("environment variable %s is not set", cfg.EnvRef)
	}
	if cfg.HasStructuredEnvConfig() {
		return buildDSNFromEnv(cfg)
	}
	if cfg.HasInlineConfig() {
		return buildDSNFromInline(cfg)
	}
	return "", fmt.Errorf("connection %s has no dsn or env_ref", cfg.Name)
}

func (e Engine) String() string {
	return string(e)
}

func (e Engine) IsVectorCapable() bool {
	return e == EnginePostgres
}

func buildDSNFromEnv(cfg ConnectionConfig) (string, error) {
	host, err := requiredEnv(cfg.HostEnv)
	if err != nil {
		return "", err
	}
	database, err := requiredEnv(cfg.DatabaseEnv)
	if err != nil {
		return "", err
	}
	user, err := requiredEnv(cfg.UserEnv)
	if err != nil {
		return "", err
	}
	password, err := requiredEnv(cfg.PasswordEnv)
	if err != nil {
		return "", err
	}

	port := defaultPortForEngine(cfg.Engine)
	if cfg.PortEnv != "" {
		value, err := requiredEnv(cfg.PortEnv)
		if err != nil {
			return "", err
		}
		port = value
	}

	switch cfg.Engine {
	case EnginePostgres:
		return fmt.Sprintf(
			"host=%s port=%s dbname=%s user=%s password=%s pool_max_conns=4 connect_timeout=15",
			host,
			port,
			database,
			user,
			password,
		), nil
	case EngineSQLServer:
		query := url.Values{}
		query.Set("database", database)
		query.Set("app name", "db-cli")
		if cfg.ApplicationIntent != "" {
			query.Set("ApplicationIntent", cfg.ApplicationIntent)
		}
		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(user, password),
			Host:     fmt.Sprintf("%s:%s", host, port),
			RawQuery: query.Encode(),
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

func requiredEnv(key string) (string, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", key)
	}
	return value, nil
}

func defaultPortForEngine(engine Engine) string {
	switch engine {
	case EngineSQLServer:
		return "1433"
	default:
		return "5432"
	}
}

func buildDSNFromInline(cfg ConnectionConfig) (string, error) {
	port := cfg.Port
	if port == "" {
		port = defaultPortForEngine(cfg.Engine)
	}
	switch cfg.Engine {
	case EnginePostgres:
		return fmt.Sprintf(
			"host=%s port=%s dbname=%s user=%s password=%s pool_max_conns=4 connect_timeout=15",
			cfg.Host, port, cfg.Database, cfg.User, cfg.Password,
		), nil
	case EngineSQLServer:
		query := url.Values{}
		query.Set("database", cfg.Database)
		query.Set("app name", "db-cli")
		if cfg.ApplicationIntent != "" {
			query.Set("ApplicationIntent", cfg.ApplicationIntent)
		}
		u := &url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     fmt.Sprintf("%s:%s", cfg.Host, port),
			RawQuery: query.Encode(),
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported engine %q", cfg.Engine)
	}
}

type DiscoveredConnection struct {
	Source string `json:"source"`
	Alias  string `json:"alias"`
	Engine string `json:"engine"`
	DSN    string `json:"dsn"`
	File   string `json:"file"`
}

func DiscoverProjectConnections() []DiscoveredConnection {
	var found []DiscoveredConnection
	wd, err := os.Getwd()
	if err != nil {
		return found
	}

	for dir := wd; ; {
		found = append(found, discoverDotNetConnections(dir)...)
		found = append(found, discoverEnvConnections(dir)...)

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return found
}

func discoverDotNetConnections(dir string) []DiscoveredConnection {
	var found []DiscoveredConnection
	patterns := []string{
		"appsettings.json",
		"appsettings.*.json",
		"**/appsettings.json",
		"**/appsettings.*.json",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if strings.Contains(match, string(os.PathSeparator)+"bin"+string(os.PathSeparator)) ||
				strings.Contains(match, string(os.PathSeparator)+"obj"+string(os.PathSeparator)) ||
				strings.Contains(match, "_tmp_publish") {
				continue
			}
			conns := parseAppSettingsConnections(match)
			found = append(found, conns...)
		}
	}
	return found
}

func parseAppSettingsConnections(path string) []DiscoveredConnection {
	var found []DiscoveredConnection
	data, err := os.ReadFile(path)
	if err != nil {
		return found
	}

	type AppSettings struct {
		ConnectionStrings map[string]string `json:"ConnectionStrings"`
	}

	var settings AppSettings
	if err := json.Unmarshal(data, &settings); err == nil && len(settings.ConnectionStrings) > 0 {
		for name, dsn := range settings.ConnectionStrings {
			if dsn == "" || strings.HasPrefix(dsn, "${") {
				continue
			}
			engine := "postgres"
			if strings.Contains(strings.ToLower(dsn), "server=") || strings.Contains(strings.ToLower(dsn), "sqlserver") {
				engine = "sqlserver"
			}

			base := filepath.Base(path)
			alias := strings.TrimSuffix(base, ".json")
			alias = strings.TrimPrefix(alias, "appsettings.")
			if alias == "appsettings" {
				alias = "default"
			}
			alias = fmt.Sprintf("%s-%s", alias, strings.ToLower(name))

			found = append(found, DiscoveredConnection{
				Source: "appsettings",
				Alias:  alias,
				Engine: engine,
				DSN:    dsn,
				File:   path,
			})
		}
	}

	return found
}

func discoverEnvConnections(dir string) []DiscoveredConnection {
	var found []DiscoveredConnection
	envFiles := []string{".env", "infra/.env", ".env.local"}

	for _, envFile := range envFiles {
		path := filepath.Join(dir, envFile)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

			keyUpper := strings.ToUpper(key)
			if strings.Contains(keyUpper, "DATABASE_URL") || strings.Contains(keyUpper, "DB_URL") ||
				strings.Contains(keyUpper, "POSTGRES_URL") || strings.Contains(keyUpper, "CONNECTION_STRING") {
				if value == "" || strings.HasPrefix(value, "${") {
					continue
				}
				engine := "postgres"
				if strings.Contains(strings.ToLower(value), "sqlserver") {
					engine = "sqlserver"
				}
				alias := fmt.Sprintf("env-%s", strings.ToLower(key))
				found = append(found, DiscoveredConnection{
					Source: "env",
					Alias:  alias,
					Engine: engine,
					DSN:    value,
					File:   path,
				})
			}
		}
	}
	return found
}
