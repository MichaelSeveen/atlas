package database

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewApplicationPool creates the bounded non-migration pool used by synchronous
// product repositories. It grants no authority beyond the configured database role.
func NewApplicationPool(ctx context.Context, config Config, maxConnections int32) (*pgxpool.Pool, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if maxConnections < 1 || maxConnections > 4 {
		return nil, errors.New("application database pool size is invalid")
	}
	endpoint := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Path:   config.Database,
	}
	query := endpoint.Query()
	query.Set("sslmode", "disable")
	endpoint.RawQuery = query.Encode()
	poolConfig, err := pgxpool.ParseConfig(endpoint.String())
	if err != nil {
		return nil, errors.New("application database configuration is invalid")
	}
	poolConfig.ConnConfig.ConnectTimeout = time.Second
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "atlas-api-identity"
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = "3000"
	poolConfig.ConnConfig.RuntimeParams["lock_timeout"] = "500"
	poolConfig.MinConns = 0
	poolConfig.MaxConns = maxConnections
	poolConfig.MaxConnIdleTime = 2 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, errors.New("create application database pool")
	}
	return pool, nil
}
