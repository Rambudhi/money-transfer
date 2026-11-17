package database

import (
	"context"

	"github.com/Rambudhi/money-transfer/internal/config"
	"github.com/Rambudhi/money-transfer/internal/infrastructure/postgres"
	"github.com/Rambudhi/money-transfer/internal/pkg/log"
	"github.com/Rambudhi/money-transfer/internal/pkg/migrations"
	"github.com/Rambudhi/money-transfer/internal/pkg/seeds"
)

func Initialize(cfg *config.Config, db *postgres.SQLDB) error {
	ctx := context.Background()

	log.Info("Starting database initialization")

	// Run migrations
	if err := migrations.Run(cfg); err != nil {
		log.Error("Failed to run database migrations", "error", err)
		return err
	}

	log.Info("Database migrations completed successfully")

	// Run seeds
	if err := seeds.Run(ctx, db.Pool); err != nil {
		log.Error("Failed to run database seeds", "error", err)
		return err
	}

	log.Info("Database seeds completed successfully")
	log.Info("Database initialization completed")

	return nil
}
