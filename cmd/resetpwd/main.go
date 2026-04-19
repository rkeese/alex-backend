package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/config"
)

func main() {
	configPath := flag.String("config", "./config/config.json", "Path to config file")
	email := flag.String("email", "", "Email of the user whose password should be changed")
	flag.Parse()

	if *email == "" {
		fmt.Println("Usage: resetpwd -email <user-email> [-config <config-path>]")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Verify user exists
	var userID string
	err = conn.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", *email).Scan(&userID)
	if err != nil {
		log.Fatalf("User with email %q not found: %v", *email, err)
	}

	fmt.Printf("User found: %s (ID: %s)\n", *email, userID)

	// Read password from stdin
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter new password: ")
	password, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read password: %v", err)
	}
	password = strings.TrimSpace(password)

	if err := validatePassword(password); err != nil {
		log.Fatalf("Password validation failed: %v", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	_, err = conn.Exec(ctx,
		"UPDATE users SET password_hash = $1, must_change_password = false, failed_login_attempts = 0, locked_until = NULL, updated_at = NOW() WHERE email = $2",
		hash, *email)
	if err != nil {
		log.Fatalf("Failed to update password: %v", err)
	}

	fmt.Println("Password updated successfully.")
}

func validatePassword(password string) error {
	if len(password) < 10 {
		return fmt.Errorf("password must be at least 10 characters long")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}
