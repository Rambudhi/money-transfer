package migrations

import (
	"embed"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/Rambudhi/money-transfer/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

func Run(cfg *config.Config) error {
	dsn := url.URL{
		Scheme: "pgx5",
		User: url.UserPassword(
			cfg.PostgreSQL.Username,
			cfg.PostgreSQL.Password,
		),
		Host: net.JoinHostPort(
			cfg.PostgreSQL.Host,
			strconv.Itoa(cfg.PostgreSQL.Port),
		),
		Path: "/" + cfg.PostgreSQL.DB,
	}

	query := dsn.Query()
	query.Set("sslmode", "disable")
	dsn.RawQuery = query.Encode()

	source, err := iofs.New(migrationFiles, "sql")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		source,
		dsn.String(),
	)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil &&
		!errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
