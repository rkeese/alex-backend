package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rkeese/alex-backend/internal/config"
)

func main() {
	configPath := "./config/config.json"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Used DB URL: %s... Failed to connect: %v", cfg.DatabaseURL[:15], err)
	}
	defer pool.Close()

	clubID := "0f8c2738-d96a-4baa-bf2a-ff91ad64da5f"

	fmt.Println("--- DEBUG: LIST ALL BOOKINGS FOR CLUB ---")
	rows, err := pool.Query(ctx, `
		SELECT booking_date, valuta_date, booking_text, amount, assigned_booking_account_id, club_bank_account_id 
		FROM bookings 
		WHERE club_id = $1
		ORDER BY valuta_date ASC
	`, clubID)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var bDate, vDate time.Time
		var text string
		var amount float64
		var assignID, bankID *string

		err := rows.Scan(&bDate, &vDate, &text, &amount, &assignID, &bankID)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		aID := "NULL"
		if assignID != nil {
			aID = *assignID
		}

		bkID := "NULL"
		if bankID != nil {
			bkID = *bankID
		}

		fmt.Printf("[%d] %s | Text: '%s' | Amount: %.2f | Assign: %s | Bank: %s\n", count, vDate.Format("2006-01-02"), text, amount, aID, bkID)
		count++
	}
	fmt.Printf("Total bookings found: %d\n", count)
}
