package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/Rambudhi/money-transfer/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

type SQLDB struct {
	Pool *pgxpool.Pool
}

type Options struct {
	Username           string
	Password           string
	Name               string
	Host               string
	Port               int
	Schema             string
	MinConnections     int32
	MaxOpenConnections int32
	MaxLifetime        time.Duration
}

func ProvidePostgresPool(db *SQLDB) *pgxpool.Pool {
	return db.Pool
}

func NewDB(
	lc fx.Lifecycle,
	cfg *config.Config,
) (*SQLDB, error) {
	opts := &Options{
		Username:           cfg.PostgreSQL.Username,
		Password:           cfg.PostgreSQL.Password,
		Name:               cfg.PostgreSQL.DB,
		Host:               cfg.PostgreSQL.Host,
		Port:               cfg.PostgreSQL.Port,
		Schema:             cfg.PostgreSQL.Schema,
		MinConnections:     int32(cfg.PostgreSQL.MinIdleConnections),
		MaxOpenConnections: int32(cfg.PostgreSQL.MaxOpenConnections),
		MaxLifetime:        time.Duration(cfg.PostgreSQL.MaxLifetime) * time.Minute,
	}

	db, err := Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := db.Pool.Ping(ctx); err != nil {
				return fmt.Errorf("ping postgres: %w", err)
			}

			return nil
		},

		OnStop: func(ctx context.Context) error {
			db.Pool.Close()
			return nil
		},
	})

	return db, nil
}

func Connect(opt *Options) (*SQLDB, error) {
	if opt == nil {
		return nil, fmt.Errorf("postgres options cannot be nil")
	}

	if opt.Host == "" || opt.Username == "" || opt.Name == "" {
		return nil, fmt.Errorf("postgres host, username, and database are required")
	}

	if opt.Port <= 0 {
		return nil, fmt.Errorf("postgres port must be greater than zero")
	}

	if opt.MinConnections < 0 ||
		opt.MaxOpenConnections <= 0 ||
		opt.MinConnections > opt.MaxOpenConnections {
		return nil, fmt.Errorf("invalid postgres connection pool configuration")
	}

	databaseURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			opt.Username,
			opt.Password,
		),
		Host: net.JoinHostPort(
			opt.Host,
			strconv.Itoa(opt.Port),
		),
		Path: "/" + opt.Name,
	}

	query := url.Values{}
	query.Set("sslmode", "disable")
	databaseURL.RawQuery = query.Encode()

	poolConfig, err := pgxpool.ParseConfig(databaseURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolConfig.MinConns = opt.MinConnections
	poolConfig.MaxConns = opt.MaxOpenConnections
	poolConfig.MaxConnLifetime = opt.MaxLifetime

	if opt.Schema != "" {
		poolConfig.ConnConfig.RuntimeParams["search_path"] = opt.Schema
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &SQLDB{
		Pool: pool,
	}, nil
}
