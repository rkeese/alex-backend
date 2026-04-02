package database

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rkeese/alex-backend/internal/config"
)

var testQueries *Queries
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// Load configuration
	// Assuming the test is run from the package directory, we need to go up two levels
	cfg, err := config.LoadConfig("../../config/config.json")
	if err != nil {
		log.Printf("Failed to load config: %v. Skipping integration tests.", err)
		// If config fails (e.g. in CI without file), we might want to skip or fail.
		// For now, let's try to proceed or exit.
		// If we can't connect to DB, we can't run integration tests.
		os.Exit(0)
	}

	ctx := context.Background()
	testPool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Unable to connect to database: %v. Skipping integration tests.", err)
		os.Exit(0)
	}
	defer testPool.Close()

	testQueries = New(testPool)

	os.Exit(m.Run())
}
