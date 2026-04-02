package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
)

// handleImportBookings handles the upload and processing of CSV bank statements into the import staging table
func (s *Server) handleImportBookings(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	// 1. Parse Multipart Form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Setup CSV Reader
	// The file might be Windows-1252 or UTF-8 (with or without BOM).
	// Since we can't easily detect, we try to read with a decoder that falls back or just use a strategy.
	// For now, let's read the file content first to check for BOM.

	// Problem: If we blindly apply Windows1252 to a UTF-8 BOM, we get garbage at the start.
	// Simple fix: We can try to detect UTF-8 BOM manually.

	// Read full content to buffer to handle BOM check
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)
	fileBytes := buf.Bytes()

	var reader *csv.Reader

	// Check for UTF-8 BOM (0xEF, 0xBB, 0xBF)
	if len(fileBytes) >= 3 && fileBytes[0] == 0xEF && fileBytes[1] == 0xBB && fileBytes[2] == 0xBF {
		// It IS a UTF-8 file with BOM. Skip BOM and read as UTF-8.
		reader = csv.NewReader(bytes.NewReader(fileBytes[3:]))
	} else {
		// Assume Windows-1252
		decoder := charmap.Windows1252.NewDecoder()
		utf8Reader := transform.NewReader(bytes.NewReader(fileBytes), decoder)
		reader = csv.NewReader(utf8Reader)
	}

	reader.Comma = ';'          // German standard
	reader.LazyQuotes = true    // Allow sloppy quotes
	reader.FieldsPerRecord = -1 // Variable fields allowed

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		http.Error(w, "Failed to read CSV: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(records) < 2 {
		http.Error(w, "CSV file is empty or missing header", http.StatusBadRequest)
		return
	}

	// 3. Identification & Parsing
	clubAccounts, err := s.Queries.ListClubBankAccounts(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to load club bank accounts", http.StatusInternalServerError)
		return
	}

	header := records[0]

	profile := DetectProfile(header)
	if profile == nil {
		http.Error(w, "Unknown CSV format. Could not detect known headers.", http.StatusBadRequest)
		return
	}

	// 4. Process Rows - Insert into bank_bookings_import
	successCount := 0
	errorCount := 0

	type ImportError struct {
		Row int    `json:"row"`
		Err string `json:"error"`
	}
	var failures []ImportError

	for i, record := range records {
		if i == 0 {
			continue // Skip header
		}
		if len(record) == 0 {
			continue
		}

		bookingData, accountIBAN, err := profile.ParseRow(record)
		if err != nil {
			errorCount++
			failures = append(failures, ImportError{Row: i + 1, Err: fmt.Sprintf("Parse error: %v", err)})
			continue
		}

		// Try to map to a Club Bank Account
		bankAccountID := pgtype.UUID{Valid: false}
		for _, acc := range clubAccounts {
			if normalizeIBAN(acc.Iban) == normalizeIBAN(accountIBAN) {
				bankAccountID = acc.ID
				break
			}
		}

		// Helper to safely deref string pointers
		safeStr := func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		}

		arg := database.CreateBankBookingImportParams{
			ClubID:                 uuidToPgtype(clubID),
			ClubBankAccountID:      bankAccountID,
			BookingDate:            pgtype.Date{Time: bookingData.BookingDate, Valid: true},
			ValutaDate:             pgtype.Date{Time: bookingData.ValutaDate, Valid: true},
			Amount:                 floatToPgNumeric(bookingData.Amount),
			Currency:               bookingData.Currency,
			Purpose:                pgtype.Text{String: bookingData.Purpose, Valid: bookingData.Purpose != ""},
			PaymentParticipantName: pgtype.Text{String: bookingData.ClientRecipient, Valid: bookingData.ClientRecipient != ""},
			PaymentParticipantIban: pgtype.Text{String: safeStr(bookingData.ClientIBAN), Valid: safeStr(bookingData.ClientIBAN) != ""},
			PaymentParticipantBic:  pgtype.Text{String: safeStr(bookingData.ClientBIC), Valid: safeStr(bookingData.ClientBIC) != ""},
			Status:                 "pending",
		}

		_, err = s.Queries.CreateBankBookingImport(r.Context(), arg)
		if err == nil {
			successCount++
		} else {
			errorCount++
			failures = append(failures, ImportError{Row: i + 1, Err: fmt.Sprintf("Database error: %v", err)})
		}
	}

	response := map[string]interface{}{
		"message": fmt.Sprintf("Imported %d bookings to staging area using profile '%s'. Errors: %d", successCount, profile.Name(), errorCount),
		"count":   successCount,
		"errors":  failures,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListImportBookings returns the pending imports
func (s *Server) handleListImportBookings(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}

	imports, err := s.Queries.ListBankBookingImports(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list imports", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(imports)
}

// handleUpdateImportBooking updates a single import record (e.g. user correcting data)
func (s *Server) handleUpdateImportBooking(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}

	importIDStr := r.PathValue("id")
	importID, err := uuid.Parse(importIDStr)
	if err != nil {
		http.Error(w, "Invalid import ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ClubBankAccountID      string  `json:"club_bank_account_id"`
		BookingDate            string  `json:"booking_date"`
		ValutaDate             string  `json:"valuta_date"`
		Amount                 float64 `json:"amount"`
		Purpose                string  `json:"purpose"`
		PaymentParticipantName string  `json:"payment_participant_name"`
		PaymentParticipantIban string  `json:"payment_participant_iban"`
		PaymentParticipantBic  string  `json:"payment_participant_bic"`
		Status                 string  `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple date parsing assuming YYYY-MM-DD
	parseDateStr := func(ds string) pgtype.Date {
		if ds == "" {
			return pgtype.Date{Valid: false}
		}
		// Assuming generic ParseDate helper or time.Parse
		// database.ParseDate is likely available if generated or custom, but let's be safe
		// Try minimal impl:
		var d pgtype.Date
		d.Scan(ds) // pgtype.Date.Scan handles standard formats
		return d
	}

	bd := parseDateStr(req.BookingDate)
	vd := parseDateStr(req.ValutaDate)

	accID := pgtype.UUID{Valid: false}
	if req.ClubBankAccountID != "" {
		if u, err := uuid.Parse(req.ClubBankAccountID); err == nil {
			accID = pgtype.UUID{Bytes: u, Valid: true}
		}
	}

	arg := database.UpdateBankBookingImportParams{
		ID:                     pgtype.UUID{Bytes: importID, Valid: true},
		ClubID:                 uuidToPgtype(clubID),
		ClubBankAccountID:      accID,
		BookingDate:            bd,
		ValutaDate:             vd,
		Amount:                 floatToPgNumeric(req.Amount),
		Purpose:                pgtype.Text{String: req.Purpose, Valid: true},
		PaymentParticipantName: pgtype.Text{String: req.PaymentParticipantName, Valid: true},
		PaymentParticipantIban: pgtype.Text{String: req.PaymentParticipantIban, Valid: true},
		PaymentParticipantBic:  pgtype.Text{String: req.PaymentParticipantBic, Valid: true},
		Status:                 req.Status,
	}

	updated, err := s.Queries.UpdateBankBookingImport(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update import record: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updated)
}

// handleDeleteImportBooking removes an import record
func (s *Server) handleDeleteImportBooking(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}

	importIDStr := r.PathValue("id")
	importID, err := uuid.Parse(importIDStr)
	if err != nil {
		http.Error(w, "Invalid import ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteBankBookingImport(r.Context(), database.DeleteBankBookingImportParams{
		ID:     pgtype.UUID{Bytes: importID, Valid: true},
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete import record", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleCommitImportBooking converts an import record into a real booking
func (s *Server) handleCommitImportBooking(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}

	importIDStr := r.PathValue("id")
	importID, err := uuid.Parse(importIDStr)
	if err != nil {
		http.Error(w, "Invalid import ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ClubBankAccountID      string  `json:"club_bank_account_id"`
		BookingDate            string  `json:"booking_date"`
		ValutaDate             string  `json:"valuta_date"`
		Amount                 float64 `json:"amount"`
		Purpose                string  `json:"purpose"`
		PaymentParticipantName string  `json:"payment_participant_name"`
		PaymentParticipantIban string  `json:"payment_participant_iban"`
		PaymentParticipantBic  string  `json:"payment_participant_bic"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClubBankAccountID == "" {
		http.Error(w, "Bank Account ID is required", http.StatusBadRequest)
		return
	}
	accID, err := uuid.Parse(req.ClubBankAccountID)
	if err != nil {
		http.Error(w, "Invalid Bank Account ID", http.StatusBadRequest)
		return
	}

	bd := pgtype.Date{}
	bd.Scan(req.BookingDate)
	vd := pgtype.Date{}
	vd.Scan(req.ValutaDate)

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())
	qtx := s.Queries.WithTx(tx)

	// Create Booking
	bookingArg := database.CreateBookingParams{
		ClubID:                   uuidToPgtype(clubID),
		BookingDate:              bd,
		ValutaDate:               vd,
		ClientRecipient:          req.PaymentParticipantName,
		BookingText:              "", // Optional, maybe map Purpose here too?
		Purpose:                  req.Purpose,
		Amount:                   floatToPgNumeric(req.Amount),
		Currency:                 "EUR",
		AssignedBookingAccountID: pgtype.UUID{Valid: false}, // User must assign later
		ClubBankAccountID:        pgtype.UUID{Bytes: accID, Valid: true},
		PaymentParticipantIban:   pgtype.Text{String: req.PaymentParticipantIban, Valid: req.PaymentParticipantIban != ""},
		PaymentParticipantBic:    pgtype.Text{String: req.PaymentParticipantBic, Valid: req.PaymentParticipantBic != ""},
	}

	newBooking, err := qtx.CreateBooking(r.Context(), bookingArg)
	if err != nil {
		http.Error(w, "Failed to create booking: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete Import Record
	err = qtx.DeleteBankBookingImport(r.Context(), database.DeleteBankBookingImportParams{
		ID:     pgtype.UUID{Bytes: importID, Valid: true},
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to remove import record", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Transaction commit failed", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(newBooking)
}

// --- Helpers ---

func normalizeIBAN(iban string) string {
	return strings.ToUpper(strings.ReplaceAll(iban, " ", ""))
}

// isOurAccount checks if the IBAN belongs to the club
func isOurAccount(iban string, accounts []database.ClubBankAccount) bool {
	norm := normalizeIBAN(iban)
	for _, acc := range accounts {
		if normalizeIBAN(acc.Iban) == norm {
			return true
		}
	}
	return false
}

func (s *Server) getFallbackBookingAccount(ctx context.Context, clubID uuid.UUID) uuid.UUID {
	accts, err := s.Queries.ListBookingAccounts(ctx, uuidToPgtype(clubID))
	if err != nil || len(accts) == 0 {
		return uuid.Nil
	}
	return pgtypeToUuid(accts[0].ID)
}

// Helper needed: context is not defined in this file snippet, add imports
