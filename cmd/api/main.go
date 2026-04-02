package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rkeese/alex-backend/internal/api"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/config"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/email"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config/config.json"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set JWT Secret
	if cfg.JWTSecret != "" {
		auth.SetJWTSecret(cfg.JWTSecret)
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Run database migrations
	migrationsDir := "./migrations"
	if err := database.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Initialize queries
	queries := database.New(pool)

	// Initialize email sender
	emailSender := email.NewSmtpSender(
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUsername,
		cfg.SMTPPassword,
		cfg.SMTPFrom,
	)

	// Setup server
	server := api.NewServer(queries, pool, emailSender)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      server.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Printf("Server starting on port %s\n", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
