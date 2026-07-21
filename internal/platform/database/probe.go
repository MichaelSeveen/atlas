// Package database owns feature-free PostgreSQL connectivity and schema readiness.
package database

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MichaelSeveen/atlas/internal/platform/migration"
	"github.com/jackc/pgx/v5"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,62}$`)

type Config struct {
	Host     string
	Port     uint16
	Database string
	User     string
	Password string
}

type SchemaProbe struct {
	config *pgx.ConnConfig
}

func ConfigFromEnvironment() (Config, error) {
	port, err := strconv.ParseUint(os.Getenv("ATLAS_POSTGRES_PORT"), 10, 16)
	if err != nil {
		return Config{}, errors.New("database port is invalid")
	}
	config := Config{
		Host: os.Getenv("ATLAS_POSTGRES_HOST"), Port: uint16(port), Database: os.Getenv("ATLAS_POSTGRES_DB"),
		User: os.Getenv("ATLAS_POSTGRES_API_USER"), Password: os.Getenv("ATLAS_POSTGRES_API_PASSWORD"),
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" || strings.ContainsAny(c.Host, " /\\@") || c.Port == 0 {
		return errors.New("database endpoint is invalid")
	}
	if !identifierPattern.MatchString(c.Database) || !identifierPattern.MatchString(c.User) {
		return errors.New("database identity is invalid")
	}
	if len(c.Password) < 32 || len(c.Password) > 256 {
		return errors.New("database credential is invalid")
	}
	return nil
}

func NewSchemaProbe(config Config) (*SchemaProbe, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	endpoint := &url.URL{
		Scheme: "postgres", User: url.UserPassword(config.User, config.Password),
		Host: net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))), Path: config.Database,
	}
	query := endpoint.Query()
	query.Set("sslmode", "disable")
	endpoint.RawQuery = query.Encode()
	connectionConfig, err := pgx.ParseConfig(endpoint.String())
	if err != nil {
		return nil, errors.New("database readiness configuration is invalid")
	}
	connectionConfig.ConnectTimeout = 500 * time.Millisecond
	connectionConfig.RuntimeParams["application_name"] = "atlas-api-readiness"
	connectionConfig.RuntimeParams["statement_timeout"] = "500"
	connectionConfig.RuntimeParams["lock_timeout"] = "250"
	return &SchemaProbe{config: connectionConfig}, nil
}

func (p *SchemaProbe) Ready(ctx context.Context) bool {
	if p == nil || p.config == nil {
		return false
	}
	probeContext, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	connection, err := pgx.ConnectConfig(probeContext, p.config.Copy())
	if err != nil {
		return false
	}
	defer connection.Close(context.Background())
	var checksum string
	err = connection.QueryRow(probeContext,
		"SELECT checksum FROM atlas_foundation.schema_migrations WHERE version = $1",
		migration.CurrentVersion,
	).Scan(&checksum)
	return err == nil && checksum == migration.CurrentChecksum
}
