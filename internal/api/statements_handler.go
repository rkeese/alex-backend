package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/pdf"
)

// Helper to convert pgtype.Numeric to float64
func numericToFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, _ := n.Float64Value()
	return f.Float64
}

// Convert float64 to pgtype.Numeric
func floatToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	n.Scan(fmt.Sprintf("%f", f))
	return n
}

type CreateFinanceStatementRequest struct {
	Year            int                `json:"year"`
	InitialBalances map[string]float64 `json:"initial_balances"`
}

type FinanceStatementReport struct {
	ClubName         string                  `json:"clubName"`
	Year             int                     `json:"year"`
	GeneratedAt      time.Time               `json:"generatedAt"`
	BankBalances     []BankBalanceRow        `json:"bankBalances"`
	TotalBankBalance BankBalanceRow          `json:"totalBankBalance"`
	Overview         []OverviewRow           `json:"overview"`
	TotalOverview    OverviewRow             `json:"totalOverview"`
	Details          map[string][]DetailItem `json:"details"`
}

type BankBalanceRow struct {
	Name         string  `json:"name"`
	StartBalance float64 `json:"startBalance"`
	Income       float64 `json:"income"`
	Expense      float64 `json:"expense"`
	EndBalance   float64 `json:"endBalance"`
}

func (r BankBalanceRow) ToPDF() pdf.BankBalanceRow {
	return pdf.BankBalanceRow{
		Name:         r.Name,
		StartBalance: r.StartBalance,
		Income:       r.Income,
		Expense:      r.Expense,
		EndBalance:   r.EndBalance,
	}
}

type OverviewRow struct {
	Name    string  `json:"name"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Result  float64 `json:"result"`
}

func (r OverviewRow) ToPDF() pdf.OverviewRow {
	return pdf.OverviewRow{
		Name:    r.Name,
		Income:  r.Income,
		Expense: r.Expense,
		Result:  r.Result,
	}
}

type DetailItem struct {
	Date        string  `json:"date"`
	BookingText string  `json:"bookingText"`
	Purpose     string  `json:"purpose"`
	Amount      float64 `json:"amount"`
	AccountName string  `json:"accountName"`
}

func (d DetailItem) ToPDF() pdf.DetailItem {
	return pdf.DetailItem{
		Date:        d.Date,
		BookingText: d.BookingText,
		Purpose:     d.Purpose,
		Amount:      d.Amount,
		AccountName: d.AccountName,
	}
}

var majorityListDescriptionsMap = map[string]string{
	"1_ideel":        "Ideeller Bereich",
	"2_vermoegen":    "Vermögens-Bereich",
	"3_zweckbetrieb": "Zweckbetrieb",
	"4_wirtschaft":   "Wirtschaftlicher Geschäftsbetrieb",
	"9_sammelposten": "Sammelposten",
}

func (s *Server) handleCreateFinanceStatement(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateFinanceStatementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Get Club Info
	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to get club info", http.StatusInternalServerError)
		return
	}

	// 2. Define Period
	startDate := time.Date(req.Year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(req.Year, 12, 31, 23, 59, 59, 0, time.UTC)

	pgStartDate := pgtype.Date{Time: startDate, Valid: true}
	pgEndDate := pgtype.Date{Time: endDate, Valid: true}

	// 3. Calculate Bank Balances
	bankAccounts, err := s.Queries.ListClubBankAccounts(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list bank accounts", http.StatusInternalServerError)
		return
	}

	bankRows := []BankBalanceRow{}
	totalBankRow := BankBalanceRow{Name: "Gesamt"}

	// Track bank rows by ID (string) to handle both UUIDs and "cash" key
	bankRowMap := make(map[string]*BankBalanceRow)

	// 3a. Initialize Bank Accounts with provided Initial Balances
	for _, ba := range bankAccounts {
		// ba.ID is pgtype.UUID. Convert to uuid.UUID then string.
		uid := pgtypeToUuid(ba.ID)
		baID := uid.String()

		startBal := 0.0
		if val, ok := req.InitialBalances[baID]; ok {
			startBal = val
		}

		row := BankBalanceRow{
			Name:         ba.Name,
			StartBalance: startBal,
			EndBalance:   startBal,
		}
		bankRows = append(bankRows, row)
	}

	// 3b. Initialize "Kasse" (Cash) - Append BEFORE creating pointers map to avoid invalidation
	{
		cashStart := 0.0
		if val, ok := req.InitialBalances["cash"]; ok {
			cashStart = val
		}
		cashRow := BankBalanceRow{
			Name:         "Kasse (Barbestand)",
			StartBalance: cashStart,
			EndBalance:   cashStart,
		}
		bankRows = append(bankRows, cashRow)
	}

	// Re-map after filling slice to get pointers to slice elements
	// Note: bankRows has len(bankAccounts) + 1 (Cash) elements.
	// The first len(bankAccounts) correspond to the bankAccounts slice.
	for i := range bankAccounts {
		uid := pgtypeToUuid(bankAccounts[i].ID)
		if uid != uuid.Nil {
			bankRowMap[uid.String()] = &bankRows[i]
		}
	}

	// Link "cash" key to the last row (Kasse)
	if len(bankRows) > 0 {
		bankRowMap["cash"] = &bankRows[len(bankRows)-1]
	}

	// 4. Fetch All Bookings for Period (Client-Side Filtering to avoid SQL Date issues)
	// We use DebugListAllBookings which fetches *everything* for the club, then we filter by year in Go.

	rawBookings, err := s.Queries.DebugListAllBookings(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list bookings", http.StatusInternalServerError)
		return
	}

	bookings := []database.DebugListAllBookingsRow{}
	for _, b := range rawBookings {
		// Filter by Year
		// Note: b.ValutaDate is pgtype.Date.
		// bDate := b.ValutaDate.Time // Time is set 00:00 UTC
		// We just check the Year of the ValutaDate

		dYear := b.ValutaDate.Time.Year()
		if dYear == req.Year {
			bookings = append(bookings, b)
		} else {
			// fmt.Printf("DEBUG: Skipping booking '%s' (Date=%s Year=%d) - Outside target year %d\n", b.BookingText, b.ValutaDate.Time.Format("2006-01-02"), dYear, req.Year)
		}
	}

	overviewMap := make(map[string]*OverviewRow) // key: majority_list
	detailsMap := make(map[string][]DetailItem)

	// Initialize Overview Rows
	categories := []string{"1_ideel", "2_vermoegen", "3_zweckbetrieb", "4_wirtschaft", "9_sammelposten"}
	for _, cat := range categories {
		overviewMap[cat] = &OverviewRow{Name: majorityListDescriptionsMap[cat]}
	}

	for _, b := range bookings {
		amt := numericToFloat(b.Amount)

		// Update Bank Balances
		// Determine target ID: UUID string or "cash"
		var targetID string
		if b.ClubBankAccountID.Valid {
			targetID = pgtypeToUuid(b.ClubBankAccountID).String()
		} else {
			targetID = "cash"
		}

		// Lookup row
		row, exists := bankRowMap[targetID]
		if !exists {
			// Fallback: If bank account not found (e.g. deleted), treat as Cash/Unknown
			// This ensures orphaned bookings like 'MasterDoor' are not dropped.
			row = bankRowMap["cash"]
			fmt.Printf("Warning: Booking '%s' (%.2f) has unknown BankAccountID %s. Mapping to Cash.\n", b.BookingText, amt, targetID)
		} else if targetID == "cash" {
			// fmt.Printf("Info: Booking '%s' (%.2f) is explicitly Cash.\n", b.BookingText, amt)
		}

		if row != nil {
			if amt >= 0 {
				row.Income += amt
			} else {
				row.Expense += math.Abs(amt)
			}
			row.EndBalance += amt
		}

		// Update Overview & Details
		// b.MajorityList, b.MinorityList come from the Join.
		// Since we handle LEFT JOIN now (planned), they might be invalid (NULL).

		cat := "9_sammelposten" // default for unknown or NULL
		if b.MajorityList.Valid && b.MajorityList.String != "" {
			cat = b.MajorityList.String
		}

		// Validate that category is one of our known types
		if _, known := majorityListDescriptionsMap[cat]; !known {
			cat = "9_sammelposten"
		}

		if _, ok := overviewMap[cat]; !ok {
			// Should not happen due to initialization, but safe fallback
			overviewMap[cat] = &OverviewRow{Name: majorityListDescriptionsMap[cat]}
		}

		// If map entry exists, update it
		if orow, ok := overviewMap[cat]; ok {
			if amt >= 0 {
				orow.Income += amt
			} else {
				orow.Expense += math.Abs(amt)
			}
			orow.Result += amt
		}

		// Detail
		accName := ""
		if b.MinorityList.Valid {
			accName = b.MinorityList.String
		}

		// Combine Recipient and BookingText for better visibility in PDF
		// Example: "MasterDoor: Invoice 123"
		displayText := b.BookingText
		if b.ClientRecipient != "" {
			if b.BookingText != "" && b.BookingText != b.ClientRecipient {
				displayText = fmt.Sprintf("%s: %s", b.ClientRecipient, b.BookingText)
			} else {
				displayText = b.ClientRecipient
			}
		}

		// Fallback if completely empty
		if displayText == "" {
			displayText = "(Kein Beschreibungstext)"
		}

		dItem := DetailItem{
			Date:        b.BookingDate.Time.Format("2006-01-02"),
			BookingText: displayText,
			Purpose:     b.Purpose,
			Amount:      amt,
			AccountName: accName,
		}

		// Determine description for grouping details
		desc := "Sonstiges"
		if val, ok := majorityListDescriptionsMap[cat]; ok {
			desc = val
		}

		detailsMap[desc] = append(detailsMap[desc], dItem)
	}

	// 5. Fetch Unbooked Receipts (Cash Barge)
	// These are valid financial records (paid from cash) that haven't been converted to bookings yet.
	receipts, err := s.Queries.ListUnbookedReceiptsInRange(r.Context(), database.ListUnbookedReceiptsInRangeParams{
		ClubID:  uuidToPgtype(clubID),
		Column2: pgStartDate,
		Column3: pgEndDate,
	})
	if err != nil {
		fmt.Printf("Error fetching receipts: %v\n", err)
	} else {
		// fmt.Printf("DEBUG: Found %d unbooked receipts (Cash Barge).\n", len(receipts))
		for _, rc := range receipts {
			amt := numericToFloat(rc.Amount) // Receipts are usually stored as positive values
			isExpense := rc.Type == "expense"

			signedAmt := amt
			if isExpense {
				signedAmt = -math.Abs(amt)
			} else {
				signedAmt = math.Abs(amt)
			}

			// 1. Update Cash Bank Row
			cashRow := bankRowMap["cash"]
			if cashRow == nil {
				// Safety fallback if cash row missing
				fmt.Println("Warning: Cash row missing for receipt aggregation")
			} else {
				if isExpense {
					cashRow.Expense += math.Abs(amt)
				} else {
					cashRow.Income += math.Abs(amt)
				}
				cashRow.EndBalance += signedAmt
			}

			// 2. Update Overview
			cat := "9_sammelposten"
			if rc.PositionAssignment.Valid && rc.PositionAssignment.String != "" {
				cat = rc.PositionAssignment.String
			}
			if _, known := majorityListDescriptionsMap[cat]; !known {
				cat = "9_sammelposten"
			}

			if _, ok := overviewMap[cat]; !ok {
				overviewMap[cat] = &OverviewRow{Name: majorityListDescriptionsMap[cat]}
			}

			if orow, ok := overviewMap[cat]; ok {
				if isExpense {
					orow.Expense += math.Abs(amt)
				} else {
					orow.Income += math.Abs(amt)
				}
				orow.Result += signedAmt
			}

			// 3. Details
			displayText := rc.Recipient
			if rc.Number != "" {
				displayText = fmt.Sprintf("%s (Beleg: %s)", displayText, rc.Number)
			}

			note := ""
			if rc.Note.Valid {
				note = rc.Note.String
			}

			rItem := DetailItem{
				Date:        rc.Date.Time.Format("2006-01-02"),
				BookingText: displayText,
				Purpose:     note,
				Amount:      signedAmt,
				AccountName: "Kasse",
			}

			desc := "Sonstiges"
			if val, ok := majorityListDescriptionsMap[cat]; ok {
				desc = val
			}
			detailsMap[desc] = append(detailsMap[desc], rItem)
		}
	}

	// Finalize structs
	// Bank Totals
	for _, r := range bankRows {
		totalBankRow.StartBalance += r.StartBalance
		totalBankRow.Income += r.Income
		totalBankRow.Expense += r.Expense
		totalBankRow.EndBalance += r.EndBalance
	}

	// Overview List
	overviewList := []OverviewRow{}
	totalOverview := OverviewRow{Name: "Gesamt"}

	// Ensure categories include Sammelposten for the final list
	outputCategories := []string{"1_ideel", "2_vermoegen", "3_zweckbetrieb", "4_wirtschaft", "9_sammelposten"}

	for _, cat := range outputCategories {
		if row, ok := overviewMap[cat]; ok {
			// Only include if non-zero? Or always?
			// Usually we include if it exists in map.
			// But specialized check for Sammelposten if empty?
			// Let's include everything initialized.
			overviewList = append(overviewList, *row)

			totalOverview.Income += row.Income
			totalOverview.Expense += row.Expense
			totalOverview.Result += row.Result
		}
	}

	report := FinanceStatementReport{
		ClubName:         club.Name,
		Year:             req.Year,
		GeneratedAt:      time.Now(),
		BankBalances:     bankRows,
		TotalBankBalance: totalBankRow,
		Overview:         overviewList,
		TotalOverview:    totalOverview,
		Details:          detailsMap,
	}

	reportJSON, err := json.Marshal(report)
	if err != nil {
		http.Error(w, "Failed to marshal report", http.StatusInternalServerError)
		return
	}

	// 6. Save to DB
	// Check if statement for this year already exists
	existingStmt, err := s.Queries.GetFinanceStatementByYear(r.Context(), database.GetFinanceStatementByYearParams{
		ClubID: uuidToPgtype(clubID),
		Year:   int32(req.Year),
	})

	if err == nil {
		// Found existing statement, delete it first to allow replacement
		// Note: database.GetFinanceStatementByYear returns a struct that has ID
		_ = s.Queries.DeleteFinanceStatement(r.Context(), database.DeleteFinanceStatementParams{
			ID:     existingStmt.ID,
			ClubID: uuidToPgtype(clubID),
		})
	}
	// If err != nil, it likely means not found (pgx.ErrNoRows), which is fine.

	// Create
	stmt, err := s.Queries.CreateFinanceStatement(r.Context(), database.CreateFinanceStatementParams{
		ClubID:    uuidToPgtype(clubID),
		Year:      int32(req.Year),
		StartDate: pgStartDate,
		EndDate:   pgEndDate,
		Data:      json.RawMessage(reportJSON),
	})
	if err != nil {
		// If conflict, return error or handle update?
		// Assuming conflict -> Delete and Re-insert logic might be better but I don't have DeleteByYear exposed?
		// Ah, I added GetFinanceStatementByYear.
		// Let's check context.
		// Actually, I can just return error "Statement already exists".
		http.Error(w, "Failed to save statement (already exists?): "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stmt)
}

func (s *Server) handleListFinanceStatements(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}

	stmts, err := s.Queries.ListFinanceStatements(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list statements", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stmts)
}

func (s *Server) handleGetFinanceStatement(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	stmt, err := s.Queries.GetFinanceStatement(r.Context(), database.GetFinanceStatementParams{
		ID:     uuidToPgtype(id),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}
	// Return the FULL stored JSON in "data" or just the record?
	// User might want the data. The struct has `Data json.RawMessage`.
	json.NewEncoder(w).Encode(stmt)
}

func (s *Server) handleDeleteFinanceStatement(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	err = s.Queries.DeleteFinanceStatement(r.Context(), database.DeleteFinanceStatementParams{
		ID:     uuidToPgtype(id),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetFinanceStatementPDF(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found", http.StatusInternalServerError)
		return
	}
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	stmt, err := s.Queries.GetFinanceStatement(r.Context(), database.GetFinanceStatementParams{
		ID:     uuidToPgtype(id),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Statement not found", http.StatusNotFound)
		return
	}

	// Unmarshal stored JSON data
	var report FinanceStatementReport
	if err := json.Unmarshal(stmt.Data, &report); err != nil {
		http.Error(w, "Failed to parse stored report data", http.StatusInternalServerError)
		return
	}

	// Convert to PDF Data Struct
	pdfData := pdf.FinanceStatementData{
		ClubName:    report.ClubName,
		Year:        report.Year,
		GeneratedAt: report.GeneratedAt,
		BankBalances: func() []pdf.BankBalanceRow {
			rows := make([]pdf.BankBalanceRow, len(report.BankBalances))
			for i, r := range report.BankBalances {
				rows[i] = r.ToPDF()
			}
			return rows
		}(),
		TotalBankBalance: report.TotalBankBalance.ToPDF(),
		Overview: func() []pdf.OverviewRow {
			rows := make([]pdf.OverviewRow, len(report.Overview))
			for i, r := range report.Overview {
				rows[i] = r.ToPDF()
			}
			return rows
		}(),
		TotalOverview: report.TotalOverview.ToPDF(),
		Details: func() map[string][]pdf.DetailItem {
			details := make(map[string][]pdf.DetailItem)
			for k, v := range report.Details {
				items := make([]pdf.DetailItem, len(v))
				for i, d := range v {
					items[i] = d.ToPDF()
				}
				details[k] = items
			}
			return details
		}(),
	}

	pdfBytes, err := pdf.GenerateFinanceStatement(pdfData)
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"Jahresabschluss_%d.pdf\"", report.Year))
	w.Write(pdfBytes)
}
