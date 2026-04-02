package api

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/pdf"
	"github.com/rkeese/alex-backend/internal/sepa"
)

// --- Booking Accounts ---

var majorityListDescriptions = map[string]string{
	"1_ideel":        "Ideeller Bereich",
	"2_vermoegen":    "Vermögens-Bereich",
	"3_zweckbetrieb": "Zweckbetrieb",
	"4_wirtschaft":   "Wirtschaftlicher Geschäftsbetrieb",
	"9_sammelposten": "Sammelposten",
}

type CreateBookingAccountRequest struct {
	MajorityList string `json:"majority_list"`
	MinorityList string `json:"minority_list"`
}

type BookingAccountResponse struct {
	ID                      pgtype.UUID        `json:"id"`
	ClubID                  pgtype.UUID        `json:"club_id"`
	MajorityList            string             `json:"majority_list"`
	MajorityListDescription string             `json:"majority_list_description"`
	MinorityList            string             `json:"minority_list"`
	CreatedAt               pgtype.Timestamptz `json:"created_at"`
	UpdatedAt               pgtype.Timestamptz `json:"updated_at"`
}

func mapBookingAccountToResponse(account database.BookingAccount) BookingAccountResponse {
	desc := majorityListDescriptions[account.MajorityList]
	return BookingAccountResponse{
		ID:                      account.ID,
		ClubID:                  account.ClubID,
		MajorityList:            account.MajorityList,
		MajorityListDescription: desc,
		MinorityList:            account.MinorityList,
		CreatedAt:               account.CreatedAt,
		UpdatedAt:               account.UpdatedAt,
	}
}

func (s *Server) handleCreateBookingAccount(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateBookingAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	arg := database.CreateBookingAccountParams{
		ClubID:       uuidToPgtype(clubID),
		MajorityList: req.MajorityList,
		MinorityList: req.MinorityList,
	}

	account, err := s.Queries.CreateBookingAccount(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create booking account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapBookingAccountToResponse(account))
}

func (s *Server) handleListBookingAccounts(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	accounts, err := s.Queries.ListBookingAccounts(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list booking accounts", http.StatusInternalServerError)
		return
	}

	response := make([]BookingAccountResponse, len(accounts))
	for i, acc := range accounts {
		response[i] = mapBookingAccountToResponse(acc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// --- Receipts ---

type CreateReceiptRequest struct {
	Type               string           `json:"type"`
	Recipient          string           `json:"recipient"`
	Number             string           `json:"number"`
	Date               string           `json:"date"` // YYYY-MM-DD
	PositionAssignment *string          `json:"position_assignment"`
	Amount             float64          `json:"amount"`
	IsBooked           bool             `json:"is_booked"`
	Note               *string          `json:"note"`
	PositionTaxAccount *string          `json:"position_tax_account"`
	PositionPercentage *string          `json:"position_percentage"`
	DonorID            *string          `json:"donor_id"`
	SellerName         *string          `json:"seller_name"`
	SellerAddress      *string          `json:"seller_address"`
	BuyerName          *string          `json:"buyer_name"`
	BuyerAddress       *string          `json:"buyer_address"`
	SellerTaxID        *string          `json:"seller_tax_id"`
	SellerVatID        *string          `json:"seller_vat_id"`
	DeliveryDate       *string          `json:"delivery_date"`
	TotalVatAmount     float64          `json:"total_vat_amount"`
	InvoiceItems       []map[string]any `json:"invoice_items"`
}

func (s *Server) handleCreateReceipt(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	var deliveryDate pgtype.Date
	if req.DeliveryDate != nil {
		dd, err := time.Parse("2006-01-02", *req.DeliveryDate)
		if err == nil {
			deliveryDate = pgtype.Date{Time: dd, Valid: true}
		}
	}

	var donorID pgtype.UUID
	if req.DonorID != nil {
		id, err := uuid.Parse(*req.DonorID)
		if err == nil {
			donorID = uuidToPgtype(id)
		}
	}

	amountNumeric := floatToPgNumeric(req.Amount)
	totalVatNumeric := floatToPgNumeric(req.TotalVatAmount)

	var percentageNumeric pgtype.Numeric
	if req.PositionPercentage != nil && *req.PositionPercentage != "" {
		// Basic parsing attempt if needed, otherwise leave invalid/null
		// In a real app we'd parse the string to numeric
		percentageNumeric = pgtype.Numeric{Valid: false}
	}

	var invoiceItemsJSON []byte
	if req.InvoiceItems != nil {
		invoiceItemsJSON, _ = json.Marshal(req.InvoiceItems)
	} else {
		invoiceItemsJSON = []byte("[]")
	}

	var positionAssignment pgtype.Text
	if req.PositionAssignment != nil && *req.PositionAssignment != "" {
		positionAssignment = pgtype.Text{String: *req.PositionAssignment, Valid: true}
	} else {
		positionAssignment = pgtype.Text{Valid: false}
	}

	arg := database.CreateReceiptParams{
		ClubID:             uuidToPgtype(clubID),
		Type:               req.Type,
		Recipient:          req.Recipient,
		Number:             req.Number,
		Date:               pgtype.Date{Time: date, Valid: true},
		PositionAssignment: positionAssignment,
		Amount:             amountNumeric,
		IsBooked:           req.IsBooked,
		Note:               stringToPgText(req.Note),
		PositionTaxAccount: stringToPgText(req.PositionTaxAccount),
		PositionPercentage: percentageNumeric,
		DonorID:            donorID,
		SellerName:         stringToPgText(req.SellerName),
		SellerAddress:      stringToPgText(req.SellerAddress),
		BuyerName:          stringToPgText(req.BuyerName),
		BuyerAddress:       stringToPgText(req.BuyerAddress),
		SellerTaxID:        stringToPgText(req.SellerTaxID),
		SellerVatID:        stringToPgText(req.SellerVatID),
		DeliveryDate:       deliveryDate,
		TotalVatAmount:     totalVatNumeric,
		InvoiceItems:       json.RawMessage(invoiceItemsJSON),
	}

	receipt, err := s.Queries.CreateReceipt(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create receipt: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipt)
}

func (s *Server) handleUpdateReceipt(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid receipt ID", http.StatusBadRequest)
		return
	}

	var req CreateReceiptRequest // Reuse create struct for update
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	var deliveryDate pgtype.Date
	if req.DeliveryDate != nil {
		dd, err := time.Parse("2006-01-02", *req.DeliveryDate)
		if err == nil {
			deliveryDate = pgtype.Date{Time: dd, Valid: true}
		}
	}

	var donorID pgtype.UUID
	if req.DonorID != nil {
		did, err := uuid.Parse(*req.DonorID)
		if err == nil {
			donorID = uuidToPgtype(did)
		}
	}

	amountNumeric := floatToPgNumeric(req.Amount)
	totalVatNumeric := floatToPgNumeric(req.TotalVatAmount)

	var percentageNumeric pgtype.Numeric
	if req.PositionPercentage != nil && *req.PositionPercentage != "" {
		percentageNumeric = pgtype.Numeric{Valid: false}
	}

	var invoiceItemsJSON []byte
	if req.InvoiceItems != nil {
		invoiceItemsJSON, _ = json.Marshal(req.InvoiceItems)
	} else {
		invoiceItemsJSON = []byte("[]")
	}

	var positionAssignment pgtype.Text
	if req.PositionAssignment != nil && *req.PositionAssignment != "" {
		positionAssignment = pgtype.Text{String: *req.PositionAssignment, Valid: true}
	} else {
		positionAssignment = pgtype.Text{Valid: false}
	}

	arg := database.UpdateReceiptParams{
		ID:                 uuidToPgtype(id),
		ClubID:             uuidToPgtype(clubID),
		Type:               req.Type,
		Recipient:          req.Recipient,
		Number:             req.Number,
		Date:               pgtype.Date{Time: date, Valid: true},
		PositionAssignment: positionAssignment,
		Amount:             amountNumeric,
		IsBooked:           req.IsBooked,
		Note:               stringToPgText(req.Note),
		PositionTaxAccount: stringToPgText(req.PositionTaxAccount),
		PositionPercentage: percentageNumeric,
		DonorID:            donorID,
		SellerName:         stringToPgText(req.SellerName),
		SellerAddress:      stringToPgText(req.SellerAddress),
		BuyerName:          stringToPgText(req.BuyerName),
		BuyerAddress:       stringToPgText(req.BuyerAddress),
		SellerTaxID:        stringToPgText(req.SellerTaxID),
		SellerVatID:        stringToPgText(req.SellerVatID),
		DeliveryDate:       deliveryDate,
		TotalVatAmount:     totalVatNumeric,
		InvoiceItems:       json.RawMessage(invoiceItemsJSON),
	}

	receipt, err := s.Queries.UpdateReceipt(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update receipt: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipt)
}

func (s *Server) handleDeleteReceipt(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid receipt ID", http.StatusBadRequest)
		return
	}

	arg := database.DeleteReceiptParams{
		ID:     uuidToPgtype(id),
		ClubID: uuidToPgtype(clubID),
	}

	if err := s.Queries.DeleteReceipt(r.Context(), arg); err != nil {
		http.Error(w, "Failed to delete receipt: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListReceipts(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	receipts, err := s.Queries.ListReceipts(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list receipts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}

type ListBookingsResponse struct {
	Bookings     []database.Booking `json:"bookings"`
	StartBalance float64            `json:"start_amount"`
	EndBalance   float64            `json:"end_amount"`
}

type UpdateBookingRequest struct {
	AssignedBookingAccountID *string `json:"assigned_booking_account_id"`
}

func (s *Server) handleListBookings(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()
	bankAccountIDStr := query.Get("bank_account_id")
	if bankAccountIDStr == "" {
		bankAccountIDStr = query.Get("bankAccountId") // Fallback for camelCase
	}
	startDateStr := query.Get("start_date")
	if startDateStr == "" {
		startDateStr = query.Get("startDate") // Fallback for camelCase
	}
	endDateStr := query.Get("end_date")
	if endDateStr == "" {
		endDateStr = query.Get("endDate") // Fallback for camelCase
	}

	var bankAccountID pgtype.UUID
	if bankAccountIDStr != "" && bankAccountIDStr != "null" {
		id, err := uuid.Parse(bankAccountIDStr)
		if err == nil {
			bankAccountID = uuidToPgtype(id)
		}
	} else {
		bankAccountID.Valid = false
	}

	var startDate, endDate pgtype.Date
	startDate.Valid = false
	endDate.Valid = false

	if startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			startDate = pgtype.Date{Time: t, Valid: true}
		}
	}
	if endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			endDate = pgtype.Date{Time: t, Valid: true}
		}
	}

	arg := database.ListBookingsParams{
		ClubID:            uuidToPgtype(clubID),
		ClubBankAccountID: bankAccountID,
		StartDate:         startDate,
		EndDate:           endDate,
	}

	bookings, err := s.Queries.ListBookings(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to list bookings: "+err.Error(), http.StatusInternalServerError)
		return
	}

	startBalance := 0.0
	// If a start date is available AND a specific bank account is selected, get balance before that date
	// using the new logic (initial balance + transactions)
	if startDate.Valid && bankAccountID.Valid {
		balArg := database.GetBookingStartBalanceParams{
			ClubID:            uuidToPgtype(clubID),
			ClubBankAccountID: bankAccountID,
			BeforeDate:        startDate,
		}
		balNumeric, err := s.Queries.GetBookingStartBalance(r.Context(), balArg)
		if err == nil {
			val, _ := balNumeric.Float64Value()
			startBalance = val.Float64
		} else {
			// If error, it might mean the bank account doesn't exist or no initial balance set?
			// Actually GetBookingStartBalance groups by ID so if no rows returned, it's 0 or empty.
		}
	}

	currentSum := 0.0
	for _, b := range bookings {
		val, _ := b.Amount.Float64Value()
		currentSum += val.Float64
	}
	endBalance := startBalance + currentSum

	// If bookings is nil, make it empty array
	if bookings == nil {
		bookings = []database.Booking{}
	}

	resp := ListBookingsResponse{
		Bookings:     bookings,
		StartBalance: startBalance,
		EndBalance:   endBalance,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleUpdateBooking(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	bookingID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid booking ID", http.StatusBadRequest)
		return
	}

	var req UpdateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var assignedID pgtype.UUID
	if req.AssignedBookingAccountID != nil {
		id, err := uuid.Parse(*req.AssignedBookingAccountID)
		if err == nil {
			assignedID = uuidToPgtype(id)
		}
	} else {
		assignedID.Valid = false
	}

	arg := database.UpdateBookingParams{
		ID:                       uuidToPgtype(bookingID),
		ClubID:                   uuidToPgtype(clubID),
		AssignedBookingAccountID: assignedID,
	}

	updatedBooking, err := s.Queries.UpdateBooking(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update booking: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedBooking)
}

// --- Club Bank Accounts ---

type CreateClubBankAccountRequest struct {
	Name               string  `json:"name"`
	AccountHolder      string  `json:"account_holder"`
	CreditorID         string  `json:"creditor_id"`
	IBAN               string  `json:"iban"`
	BIC                *string `json:"bic"`
	IsDefault          bool    `json:"is_default"`
	InitialBalance     float64 `json:"initial_balance"`
	InitialBalanceDate *string `json:"initial_balance_date"` // YYYY-MM-DD
}

func (s *Server) handleCreateClubBankAccount(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateClubBankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	arg := database.CreateClubBankAccountParams{
		ClubID:        uuidToPgtype(clubID),
		Name:          req.Name,
		AccountHolder: req.AccountHolder,
		CreditorID:    req.CreditorID,
		Iban:          req.IBAN,
		Bic:           stringToPgText(req.BIC),
		IsDefault:     req.IsDefault,
	}

	account, err := s.Queries.CreateClubBankAccount(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create club bank account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (s *Server) handleListClubBankAccounts(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	accounts, err := s.Queries.ListClubBankAccounts(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list club bank accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (s *Server) handleGetClubBankAccount(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	accountID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid bank account ID", http.StatusBadRequest)
		return
	}

	account, err := s.Queries.GetClubBankAccountByID(r.Context(), database.GetClubBankAccountByIDParams{
		ID:     uuidToPgtype(accountID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Bank account not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (s *Server) handleUpdateClubBankAccount(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	accountID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid bank account ID", http.StatusBadRequest)
		return
	}

	var req CreateClubBankAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	arg := database.UpdateClubBankAccountParams{
		ID:            uuidToPgtype(accountID),
		ClubID:        uuidToPgtype(clubID),
		Name:          req.Name,
		AccountHolder: req.AccountHolder,
		CreditorID:    req.CreditorID,
		Iban:          req.IBAN,
		Bic:           stringToPgText(req.BIC),
		IsDefault:     req.IsDefault,
	}

	account, err := s.Queries.UpdateClubBankAccount(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update club bank account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)
}

func (s *Server) handleDeleteClubBankAccount(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	accountID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid bank account ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteClubBankAccount(r.Context(), database.DeleteClubBankAccountParams{
		ID:     uuidToPgtype(accountID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete club bank account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- SEPA ---

type SEPAMemberResponse struct {
	MemberID            string  `json:"member_id"`
	FirstName           string  `json:"first_name"`
	LastName            string  `json:"last_name"`
	Amount              float64 `json:"amount"`
	FeeLabel            string  `json:"fee_label"`
	MemberIBAN          string  `json:"member_iban"`
	MemberBIC           string  `json:"member_bic,omitempty"`
	MandateReference    string  `json:"mandate_reference"`
	MandateIssuedAt     string  `json:"mandate_issued_at"`
	SequenceType        string  `json:"sequence_type"`
	TargetAccountHolder string  `json:"target_account_holder"`
	TargetIBAN          string  `json:"target_iban"`
	TargetBankName      string  `json:"target_bank_name"`
}

func (s *Server) handleGetSEPAMembers(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	executionDateStr := r.URL.Query().Get("execution_date")
	if executionDateStr == "" {
		http.Error(w, "execution_date query parameter is required", http.StatusBadRequest)
		return
	}

	executionDate, err := time.Parse("2006-01-02", executionDateStr)
	if err != nil {
		http.Error(w, "Invalid execution date format", http.StatusBadRequest)
		return
	}

	fees, err := s.Queries.GetDueMembershipFees(r.Context(), database.GetDueMembershipFeesParams{
		ClubID:       uuidToPgtype(clubID),
		MaturityDate: pgtype.Date{Time: executionDate, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to fetch due fees: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare GroupMap to replicate grouping logic for consistent view,
	// OR just return list. User asked for "complete list of members".
	// A flat list with sequence type info is probably best.

	response := []SEPAMemberResponse{}
	for _, fee := range fees {
		amount, _ := fee.Amount.Float64Value()

		seqType := "RCUR"
		if fee.NextDirectDebitType != "" {
			seqType = fee.NextDirectDebitType
		}

		memberID, _ := uuid.FromBytes(fee.MemberID.Bytes[:])

		resp := SEPAMemberResponse{
			MemberID:            memberID.String(),
			FirstName:           fee.FirstName,
			LastName:            fee.LastName,
			Amount:              amount.Float64,
			FeeLabel:            fee.FeeLabel,
			MemberIBAN:          fee.MemberIban,
			MemberBIC:           fee.MemberBic.String,
			MandateReference:    fee.MandateReference.String,
			MandateIssuedAt:     "",
			SequenceType:        seqType,
			TargetAccountHolder: fee.TargetAccountHolder,
			TargetIBAN:          fee.TargetIban,
			TargetBankName:      fee.TargetBankName,
		}

		if fee.MandateIssuedAt.Valid {
			resp.MandateIssuedAt = fee.MandateIssuedAt.Time.Format("2006-01-02")
		}

		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type GenerateSEPARequest struct {
	ExecutionDate string `json:"execution_date"` // YYYY-MM-DD
}

func (s *Server) handleGenerateSEPA(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req GenerateSEPARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	executionDate, err := time.Parse("2006-01-02", req.ExecutionDate)
	if err != nil {
		http.Error(w, "Invalid execution date format", http.StatusBadRequest)
		return
	}

	// 1. Get Club (for Initiator Name)
	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Club not found", http.StatusInternalServerError)
		return
	}

	// 2. Get Due Fees
	fees, err := s.Queries.GetDueMembershipFees(r.Context(), database.GetDueMembershipFeesParams{
		ClubID:       uuidToPgtype(clubID),
		MaturityDate: pgtype.Date{Time: executionDate, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to fetch due fees: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(fees) == 0 {
		http.Error(w, "No due fees found", http.StatusNotFound)
		return
	}

	// 2a. Validate Data
	// Since we relaxed constraints for import, we must ensure data is valid before generating SEPA.
	validationErrors := []string{}
	for _, fee := range fees {
		memberIdentifier := fmt.Sprintf("%s %s (ID: %s)", fee.FirstName, fee.LastName, uuidToString(fee.MemberID))

		amount, _ := fee.Amount.Float64Value()
		if amount.Float64 <= 0 {
			// Skip fees with 0 or negative amount for SEPA generation
			continue
		}

		if !fee.MandateReference.Valid || fee.MandateReference.String == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("Missing Mandate Reference for %s", memberIdentifier))
		}
		if !fee.MandateIssuedAt.Valid {
			validationErrors = append(validationErrors, fmt.Sprintf("Missing Mandate Date for %s", memberIdentifier))
		}
	}

	if len(validationErrors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Validation failed",
			"details": validationErrors,
		})
		return
	}

	// 3. Group Transactions by Creditor (Target Account) and Sequence Type
	// Map: TargetIBAN -> SequenceType -> PaymentGroup
	bankGroups := make(map[string]map[string]*sepa.PaymentGroup)
	bankNames := make(map[string]string)

	for _, fee := range fees {
		amount, _ := fee.Amount.Float64Value()

		seqType := "RCUR"
		if fee.NextDirectDebitType != "" {
			seqType = fee.NextDirectDebitType
		}

		targetIBAN := strings.ReplaceAll(fee.TargetIban, " ", "")
		if _, ok := bankGroups[targetIBAN]; !ok {
			bankGroups[targetIBAN] = make(map[string]*sepa.PaymentGroup)
			bankNames[targetIBAN] = fee.TargetBankName
		}

		if _, exists := bankGroups[targetIBAN][seqType]; !exists {
			bankGroups[targetIBAN][seqType] = &sepa.PaymentGroup{
				CreditorName: fee.TargetAccountHolder,
				CreditorIBAN: targetIBAN,
				CreditorBIC:  fee.TargetBic,
				CreditorID:   fee.TargetCreditorID,
				SequenceType: seqType,
				Transactions: []sepa.Transaction{},
			}
		}

		tx := sepa.Transaction{
			EndToEndID:     uuid.New().String(),
			Amount:         amount.Float64,
			DebtorName:     fee.FirstName + " " + fee.LastName,
			DebtorIBAN:     strings.ReplaceAll(fee.MemberIban, " ", ""),
			DebtorBIC:      fee.MemberBic.String,
			MandateID:      fee.MandateReference.String,
			MandateDate:    fee.MandateIssuedAt.Time,
			RemittanceInfo: fee.FeeLabel,
		}
		bankGroups[targetIBAN][seqType].Transactions = append(bankGroups[targetIBAN][seqType].Transactions, tx)
	}

	// 4. Generate XMLs and Zip
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for iban, groups := range bankGroups {
		var paymentGroups []sepa.PaymentGroup
		for _, g := range groups {
			paymentGroups = append(paymentGroups, *g)
		}

		bankName := "SEPA_Export"
		if name, ok := bankNames[iban]; ok && name != "" {
			bankName = name
		}
		// Sanitize filename
		filename := fmt.Sprintf("%s_%s.xml", bankName, executionDate.Format("2006-01-02"))

		safeBankName := bankName
		if len(safeBankName) > 10 {
			safeBankName = safeBankName[:10]
		}

		config := sepa.Config{
			MessageID:      "MSG-" + time.Now().Format("20060102150405") + "-" + safeBankName, // Ensure unique MsgId?
			InitiatorName:  club.Name,
			CollectionDate: executionDate,
		}

		xmlBytes, err := sepa.GenerateSEPA(config, paymentGroups)
		if err != nil {
			http.Error(w, "Failed to generate SEPA XML: "+err.Error(), http.StatusInternalServerError)
			return
		}

		f, err := zipWriter.Create(filename)
		if err != nil {
			http.Error(w, "Failed to create zip entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = f.Write(xmlBytes)
		if err != nil {
			http.Error(w, "Failed to write zip entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := zipWriter.Close(); err != nil {
		http.Error(w, "Failed to close zip archive: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 5. Return File
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=sepa_files.zip")
	w.Write(buf.Bytes())
}

// --- PDF ---

func (s *Server) handleGenerateDonationReceipt(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	receiptID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid receipt ID", http.StatusBadRequest)
		return
	}

	// 1. Get Receipt & Donor
	receipt, err := s.Queries.GetReceiptWithDonor(r.Context(), database.GetReceiptWithDonorParams{
		ID:     uuidToPgtype(receiptID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// 2. Get Club
	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Club not found", http.StatusInternalServerError)
		return
	}

	// 3. Generate PDF
	amount, _ := receipt.Amount.Float64Value()

	donorName := "Unbekannt"
	donorAddress := ""
	if receipt.FirstName.Valid || receipt.LastName.Valid {
		donorName = receipt.FirstName.String + " " + receipt.LastName.String
		donorAddress = receipt.DonorStreet.String + ", " + receipt.DonorZip.String + " " + receipt.DonorCity.String
	}

	pdfBytes, err := pdf.GenerateDonationReceipt(pdf.ReceiptData{
		ClubName:      club.Name,
		ClubAddress:   club.StreetHouseNumber.String + ", " + club.PostalCode.String + " " + club.City.String,
		DonorName:     donorName,
		DonorAddress:  donorAddress,
		Amount:        amount.Float64,
		Date:          receipt.Date.Time,
		ReceiptNumber: receipt.Number,
	})
	if err != nil {
		http.Error(w, "Failed to generate PDF: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Return File
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=spendenquittung.pdf")
	w.Write(pdfBytes)
}

func (s *Server) handleBookReceipt(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	receiptUUID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid receipt ID", http.StatusBadRequest)
		return
	}

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	q := s.Queries.WithTx(tx)

	receipt, err := q.GetReceipt(r.Context(), database.GetReceiptParams{
		ID:     uuidToPgtype(receiptUUID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	if receipt.IsBooked {
		http.Error(w, "Receipt is already booked", http.StatusBadRequest)
		return
	}

	purpose := receipt.Type
	if receipt.Note.Valid && receipt.Note.String != "" {
		purpose = receipt.Note.String
	}

	var assignedAccountID pgtype.UUID
	if receipt.PositionAssignment.Valid {
		if uuidVal, err := uuid.Parse(receipt.PositionAssignment.String); err == nil {
			assignedAccountID = uuidToPgtype(uuidVal)
		} else {
			assignedAccountID = pgtype.UUID{Valid: false}
		}
	} else {
		assignedAccountID = pgtype.UUID{Valid: false}
	}

	bookingText := fmt.Sprintf("Beleg %s", receipt.Number)

	arg := database.CreateBookingParams{
		ClubID:                   uuidToPgtype(clubID),
		BookingDate:              receipt.Date,
		ValutaDate:               receipt.Date,
		ClientRecipient:          receipt.Recipient,
		BookingText:              bookingText,
		Purpose:                  purpose,
		Amount:                   receipt.Amount,
		Currency:                 "EUR",
		ReceiptID:                receipt.ID,
		AssignedBookingAccountID: assignedAccountID,
		ClubBankAccountID:        pgtype.UUID{Valid: false},
		PaymentParticipantIban:   pgtype.Text{Valid: false},
		PaymentParticipantBic:    pgtype.Text{Valid: false},
	}

	_, err = q.CreateBooking(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create booking: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = q.SetReceiptBooked(r.Context(), database.SetReceiptBookedParams{
		ID:       receipt.ID,
		ClubID:   receipt.ClubID,
		IsBooked: true,
	})
	if err != nil {
		http.Error(w, "Failed to update receipt status", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Receipt transferred to booking successfully"}`))
}

// Helper for float to pgtype.Numeric (Simplified, might need proper implementation)
func floatToPgNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	s := fmt.Sprintf("%.2f", f)
	if err := n.Scan(s); err != nil {
		fmt.Printf("Error scanning float to numeric: %v\n", err)
	}
	return n
}
