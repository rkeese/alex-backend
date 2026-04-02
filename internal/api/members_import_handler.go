package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
)

type ImportMembersResponse struct {
	SuccessCount int      `json:"success_count"`
	Errors       []string `json:"errors"`
}

func (s *Server) handleImportMembers(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	// limit file size to 10MB
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// German CSVs often use semicolon, but the sample used comma (based on standard CSV view).
	reader.Comma = ','
	// Relax the fields per record check to allow variable length rows if necessary,
	// though standard CSV should be consistent.
	reader.FieldsPerRecord = -1

	// Read header
	header, err := reader.Read()
	if err != nil {
		http.Error(w, "Error reading CSV header", http.StatusBadRequest)
		return
	}

	// Create a map of header name to index
	headerMap := make(map[string]int)
	for i, h := range header {
		// Trim BOM or spaces from header names if present
		cleanHeader := strings.TrimSpace(strings.ReplaceAll(h, "\ufeff", ""))
		headerMap[cleanHeader] = i
	}

	// Validate required headers
	requiredHeaders := []string{"Nachname", "Vorname", "Mitgliedsnr."}
	for _, h := range requiredHeaders {
		if _, ok := headerMap[h]; !ok {
			// Try fuzzy match or listing available headers for debugging
			http.Error(w, fmt.Sprintf("Missing required header: %s. Found: %v", h, header), http.StatusBadRequest)
			return
		}
	}

	ctx := r.Context()
	response := ImportMembersResponse{
		Errors: []string{},
	}

	line := 1 // Header is line 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Error reading record: %v", line, err))
			continue
		}

		// Helper to get value
		getVal := func(colName string) string {
			if idx, ok := headerMap[colName]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		// Parse basic fields
		memberNumber := getVal("Mitgliedsnr.")
		firstName := getVal("Vorname")
		lastName := getVal("Nachname")
		email := getVal("E-Mail") // Optional

		// Basic validation
		if firstName == "" || lastName == "" {
			response.Errors = append(response.Errors, fmt.Sprintf("Line %d: First/Last name missing", line))
			continue
		}

		// Create Member Params
		createMemberParams := database.CreateMemberParams{
			ClubID:            pgtype.UUID{Bytes: clubID, Valid: true},
			MemberNumber:      memberNumber,
			FirstName:         firstName,
			LastName:          lastName,
			Email:             pgtype.Text{String: email, Valid: email != ""},
			Salutation:        mapSalutation(getVal("Anrede")),
			Title:             toPgText(getVal("Titel")),
			Phone1:            toPgText(getVal("Telefon")),
			Mobile:            toPgText(getVal("Mobil")),
			StreetHouseNumber: toPgText(getVal("Strasse & Hausnr.")),
			PostalCode:        toPgText(getVal("PLZ")),
			City:              toPgText(getVal("Ort")),
			Nation:            toPgText(getVal("Land")),
			Status:            mapStatus(getVal("Status")),
			Gender:            mapGender(getVal("Geschlecht")),
			MaritalStatus:     mapMaritalStatus(getVal("Familienstand")),
			Note:              toPgText(getVal("Notizen")),
		}

		// Parse Dates
		birthDatePG, err := parseGermanDate(getVal("Geburtstag"))
		if err != nil && getVal("Geburtstag") != "" {
			response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Invalid BirthDate '%s': %v", line, getVal("Geburtstag"), err))
		} else {
			createMemberParams.BirthDate = birthDatePG
		}

		joinedAtPG, err := parseGermanDate(getVal("Mitglied seit"))
		if err == nil {
			createMemberParams.JoinedAt = joinedAtPG
		}

		memberUntilPG, err := parseGermanDate(getVal("Mitglied bis"))
		if err == nil {
			createMemberParams.MemberUntil = memberUntilPG
		}

		// Honorary (Ehrenmitglied)
		honoraryStr := getVal("Ehrenmitglied")
		createMemberParams.Honorary = strings.EqualFold(honoraryStr, "Ja")

		// Insert Member
		member, err := s.Queries.CreateMember(ctx, createMemberParams)
		if err != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Database error creating member: %v", line, err))
			continue
		}

		// --- Bank Account ---
		iban := getVal("IBAN")
		if iban != "" {
			accountHolder := getVal("Kontoinhaber")
			if accountHolder == "" {
				accountHolder = fmt.Sprintf("%s %s", firstName, lastName)
			}

			// mandateDate is pgtype.Date
			mandateDate, err := parseGermanDate(getVal("Mandat erteilt am"))
			if err != nil || !mandateDate.Valid {
				// Fallback: Check if "SEPA-Mandat erteilt" contains the date
				altDate, err2 := parseGermanDate(getVal("SEPA-Mandat erteilt"))
				if err2 == nil && altDate.Valid {
					mandateDate = altDate
				} else {
					// Fallback: Default to now if required
					mandateDate = pgtype.Date{Time: time.Now(), Valid: true}
				}
			}

			// lastUsedDate is pgtype.Date
			lastUsedDate, _ := parseGermanDate(getVal("Letzte Verwendung"))

			// Mandate Valid Until
			mandateValidUntil := pgtype.Date{
				Time:  time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC),
				Valid: true,
			}

			mandateRef := getVal("Mandatsreferenz")
			if mandateRef == "" {
				mandateRef = fmt.Sprintf("MANDATE-%s-%s", memberNumber, time.Now().Format("20060102"))
			}

			sepaMsg := getVal("SEPA-Mandat erteilt")
			hasSepa := strings.EqualFold(sepaMsg, "Einzug") || strings.EqualFold(sepaMsg, "Ja")
			if !hasSepa {
				_, errDate := parseGermanDate(sepaMsg)
				if errDate == nil {
					hasSepa = true
				}
			}

			_, err = s.Queries.CreateMemberBankAccount(ctx, database.CreateMemberBankAccountParams{
				MemberID:             member.ID,
				AccountHolder:        accountHolder,
				Iban:                 iban,
				Bic:                  pgtype.Text{Valid: false},
				SepaMandateAvailable: hasSepa,
				MandateReference:     toPgText(mandateRef),
				MandateType:          "basic",
				NextDirectDebitType:  mapDirectDebitType(getVal("Art der nächsten Lastschrift")),
				MandateIssuedAt:      mandateDate,
				LastUsedAt:           lastUsedDate,
				MandateKind:          mapMandateKind(getVal("Art des Mandats")),
				MandateValidUntil:    mandateValidUntil,
			})
			if err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Error creating bank account: %v", line, err))
			}
		}

		// --- Fee ---
		feeLabel := getVal("Beitrag (Bezeichnung)")
		if feeLabel != "" {
			amountStr := getVal("Beitrag (Betrag)")
			amountStr = strings.ReplaceAll(amountStr, ",", ".")
			// Remove currency symbol if present
			amountStr = strings.TrimSuffix(amountStr, " €")
			amountStr = strings.TrimSpace(amountStr)

			var amount pgtype.Numeric
			if err := amount.Scan(amountStr); err != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Invalid Amount: %v", line, err))
			} else {
				maturityPG, _ := parseGermanDate(getVal("Beitrag (Fälligkeit)"))

				// Period defaults?
				period := getVal("Beitrag (Zeitraum)")

				_, err = s.Queries.CreateMembershipFee(ctx, database.CreateMembershipFeeParams{
					MemberID:      member.ID,
					FeeLabel:      feeLabel,
					FeeType:       getVal("Beitrag (Typ)"),
					Amount:        amount,
					Period:        mapContributionPeriod(period),
					MaturityDate:  maturityPG,
					PaymentMethod: mapPaymentMethod(getVal("Zahlungsart")),
					Assignment:    "auto",                    // Default assignment? Required field.
					StartsAt:      pgtype.Date{Valid: false}, // Not in CSV? Maybe "Mitglied seit"?
					EndsAt:        pgtype.Date{Valid: false},
				})
				if err != nil {
					response.Errors = append(response.Errors, fmt.Sprintf("Line %d: Error creating fee: %v", line, err))
				}
			}
		}

		response.SuccessCount++
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func parseGermanDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{Valid: false}, nil
	}
	// Try formats: d.m.yyyy, dd.mm.yyyy
	t, err := time.Parse("2.1.2006", s)
	if err == nil {
		return pgtype.Date{Time: t, Valid: true}, nil
	}
	t, err = time.Parse("02.01.2006", s)
	if err == nil {
		return pgtype.Date{Time: t, Valid: true}, nil
	}
	return pgtype.Date{Valid: false}, err
}

func toPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func mapGender(s string) pgtype.Text {
	s = strings.ToLower(strings.TrimSpace(s))
	// German matches
	if strings.Contains(s, "männlich") || s == "m" || s == "maennlich" || s == "male" {
		return pgtype.Text{String: "m", Valid: true}
	}
	if strings.Contains(s, "weiblich") || s == "w" || s == "f" || s == "female" {
		return pgtype.Text{String: "f", Valid: true}
	}
	if strings.Contains(s, "divers") || s == "d" || s == "diverse" {
		return pgtype.Text{String: "d", Valid: true}
	}
	// Default to unknown or let the DB fail?
	// The DB constraint is (gender IN ('unknown', 'm', 'f', 'd'))
	// Let's default to unknown if we can't map it, to avoid import failure.
	return pgtype.Text{String: "unknown", Valid: true}
}

func mapStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "active" || s == "aktiv" {
		return "active"
	}
	if s == "passive" || s == "passiv" {
		return "passive"
	}
	if s == "honorary" || s == "ehrenmitglied" {
		return "honorary"
	}
	if s == "inactive" || s == "inaktiv" {
		return "inactive"
	}
	// Default to passive? Or cancelled?
	// DB likely has a check/enum. Let's assume 'passive' is safe if not active.
	return "passive"
}

func mapSalutation(s string) pgtype.Text {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "herr" || s == "mr" || s == "mr." {
		return pgtype.Text{String: "mr", Valid: true}
	}
	if s == "frau" || s == "ms" || s == "ms." || s == "mrs" || s == "mrs." {
		return pgtype.Text{String: "ms", Valid: true}
	}
	if s == "divers" || s == "div" || s == "mx" {
		return pgtype.Text{String: "div", Valid: true}
	}
	if s == "firma" || s == "company" {
		return pgtype.Text{String: "company", Valid: true}
	}
	return pgtype.Text{Valid: false}
}

func mapMaritalStatus(s string) pgtype.Text {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "ledig" || s == "single" {
		return pgtype.Text{String: "single", Valid: true}
	}
	if s == "verheiratet" || s == "married" {
		return pgtype.Text{String: "married", Valid: true}
	}
	if s == "geschieden" || s == "divorced" {
		return pgtype.Text{String: "divorced", Valid: true}
	}
	if s == "verwitwet" || s == "widowed" {
		return pgtype.Text{String: "widowed", Valid: true}
	}
	return pgtype.Text{Valid: false}
}

func mapContributionPeriod(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(s, "monat") || s == "monthly" {
		return "monthly"
	}
	if strings.Contains(s, "viertel") || s == "quarterly" {
		return "quarterly"
	}
	if strings.Contains(s, "halb") || s == "half_yearly" || s == "semiannually" {
		return "half_yearly"
	}
	if strings.Contains(s, "jahr") || s == "jährlich" || s == "jaehrlich" || s == "yearly" || s == "annually" {
		return "yearly"
	}
	// Default
	return "yearly"
}

func mapPaymentMethod(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(s, "sepa") || strings.Contains(s, "lastschrift") || s == "sepa" || s == "sepa_direct_debit" {
		return "sepa"
	}
	if strings.Contains(s, "rechnung") || s == "invoice" {
		return "invoice"
	}
	if strings.Contains(s, "überweisung") || strings.Contains(s, "ueberweisung") || s == "transfer" {
		return "transfer"
	}
	if strings.Contains(s, "bar") || s == "cash" {
		return "cash"
	}
	return "transfer"
}

func mapMandateKind(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// CSV: "Einmalig" -> one_time
	// DB: open_ended, one_time, last
	if strings.Contains(s, "einmalig") || s == "one_time" || s == "one-time" {
		return "one_time"
	}
	if strings.Contains(s, "letzte") || s == "last" {
		return "last"
	}
	// Default to open_ended (Recurrent)
	return "open_ended"
}

func mapDirectDebitType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// CSV: "Erste Lastschrift" -> first
	// DB: first, followup, last
	if strings.Contains(s, "erste") || s == "first" {
		return "first"
	}
	if strings.Contains(s, "folge") || s == "followup" {
		return "followup"
	}
	if strings.Contains(s, "letzte") || s == "last" {
		return "last"
	}
	// Default? 'first' is safe for new entries.
	return "first"
}
