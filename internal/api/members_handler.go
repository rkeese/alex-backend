package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/pdf"
)

type CreateMemberRequest struct {
	MemberNumber       string  `json:"member_number"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	BirthDate          *string `json:"birth_date"` // YYYY-MM-DD
	Gender             string  `json:"gender"`
	StreetHouseNumber  *string `json:"street_house_number"`
	PostalCode         *string `json:"postal_code"`
	City               *string `json:"city"`
	Honorary           bool    `json:"honorary"`
	Status             string  `json:"status"`
	Salutation         *string `json:"salutation"`
	LetterSalutation   *string `json:"letter_salutation"`
	Phone1             *string `json:"phone1"`
	Phone1Note         *string `json:"phone1_note"`
	Phone2             *string `json:"phone2"`
	Phone2Note         *string `json:"phone2_note"`
	Mobile             *string `json:"mobile"`
	MobileNote         *string `json:"mobile_note"`
	Email              *string `json:"email"`
	EmailNote          *string `json:"email_note"`
	Nation             *string `json:"nation"`
	JoinedAt           *string `json:"joined_at"`    // YYYY-MM-DD
	MemberUntil        *string `json:"member_until"` // YYYY-MM-DD
	Note               *string `json:"note"`
	MaritalStatus      *string `json:"marital_status"`
	Title              *string `json:"title"`
	AssignedClubBankID *string `json:"assigned_club_bank_id"`

	// Bank Details
	BankAccountID      *string `json:"bank_account_id"`
	IBAN               *string `json:"iban"`
	AccountHolder      *string `json:"account_holder"`
	SepaMandateGranted *bool   `json:"sepa_mandate_granted"`
	MandateReference   *string `json:"mandate_reference"`
	MandateGrantedAt   *string `json:"mandate_granted_at"`
	MandateType        *string `json:"mandate_type"`
	MandateKind        *string `json:"mandate_kind"`

	// Fee Details
	PaymentMethod     *string  `json:"payment_method"`
	FeeLabel          *string  `json:"fee_label"`
	FeeType           *string  `json:"fee_type"`
	FeeAssignment     *string  `json:"fee_assignment"`
	FeeAmount         *float64 `json:"fee_amount"`
	FeePeriod         *string  `json:"fee_period"`
	FeeMaturity       *string  `json:"fee_maturity"`
	FeeStartsAt       *string  `json:"fee_starts_at"`
	FeeEndsAt         *string  `json:"fee_ends_at"`
	CreditorAccountID *string  `json:"creditor_account_id"`
}

func (s *Server) handleCreateMember(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BirthDate == nil || *req.BirthDate == "" {
		http.Error(w, "Birth date is required", http.StatusBadRequest)
		return
	}
	birthDate, err := time.Parse("2006-01-02", *req.BirthDate)
	if err != nil {
		http.Error(w, "Invalid birth date format", http.StatusBadRequest)
		return
	}

	if req.JoinedAt == nil || *req.JoinedAt == "" {
		http.Error(w, "Joined at date is required", http.StatusBadRequest)
		return
	}
	joinedAt, err := time.Parse("2006-01-02", *req.JoinedAt)
	if err != nil {
		http.Error(w, "Invalid joined at date format", http.StatusBadRequest)
		return
	}

	if req.StreetHouseNumber == nil || *req.StreetHouseNumber == "" {
		http.Error(w, "Street/House Number is required", http.StatusBadRequest)
		return
	}
	if req.PostalCode == nil || *req.PostalCode == "" {
		http.Error(w, "Postal code is required", http.StatusBadRequest)
		return
	}
	if req.City == nil || *req.City == "" {
		http.Error(w, "City is required", http.StatusBadRequest)
		return
	}

	var memberUntil pgtype.Date
	if req.MemberUntil != nil {
		t, err := time.Parse("2006-01-02", *req.MemberUntil)
		if err == nil {
			memberUntil = pgtype.Date{Time: t, Valid: true}
		}
	}

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	qtx := s.Queries.WithTx(tx)

	var assignedBankID pgtype.UUID
	if req.AssignedClubBankID != nil && *req.AssignedClubBankID != "" {
		if uid, err := uuid.Parse(*req.AssignedClubBankID); err == nil {
			assignedBankID = uuidToPgtype(uid)
		}
	}

	arg := database.CreateMemberParams{
		ClubID:             uuidToPgtype(clubID),
		MemberNumber:       req.MemberNumber,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		BirthDate:          pgtype.Date{Time: birthDate, Valid: true},
		Gender:             pgtype.Text{String: req.Gender, Valid: req.Gender != ""},
		StreetHouseNumber:  stringToPgText(req.StreetHouseNumber),
		PostalCode:         stringToPgText(req.PostalCode),
		City:               stringToPgText(req.City),
		Honorary:           req.Honorary,
		Status:             req.Status,
		Salutation:         stringToPgText(req.Salutation),
		LetterSalutation:   stringToPgText(req.LetterSalutation),
		Phone1:             stringToPgText(req.Phone1),
		Phone1Note:         stringToPgText(req.Phone1Note),
		Phone2:             stringToPgText(req.Phone2),
		Phone2Note:         stringToPgText(req.Phone2Note),
		Mobile:             stringToPgText(req.Mobile),
		MobileNote:         stringToPgText(req.MobileNote),
		Email:              stringToPgText(req.Email),
		EmailNote:          stringToPgText(req.EmailNote),
		Nation:             stringToPgText(req.Nation),
		JoinedAt:           pgtype.Date{Time: joinedAt, Valid: true},
		MemberUntil:        memberUntil,
		Note:               stringToPgText(req.Note),
		MaritalStatus:      stringToPgText(req.MaritalStatus),
		Title:              stringToPgText(req.Title),
		AssignedClubBankID: assignedBankID,
	}

	member, err := qtx.CreateMember(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create Bank Account
	if req.IBAN != nil && *req.IBAN != "" {
		mandateDate := pgtype.Date{Valid: false}
		if req.MandateGrantedAt != nil && *req.MandateGrantedAt != "" {
			t, err := time.Parse("2006-01-02", *req.MandateGrantedAt)
			if err == nil {
				mandateDate = pgtype.Date{Time: t, Valid: true}
			}
		} else if req.SepaMandateGranted != nil && *req.SepaMandateGranted {
			// If SEPA granted but no date, default into today (strict mode intention) or let it be NULL?
			// User said "constraints should take effect when creating a new data record."
			// So if creating, we might want to default to today if missing?
			// Let's default to today ONLY if it is a NEW member creation AND mandate is granted.
			mandateDate = pgtype.Date{Time: time.Now(), Valid: true}
		}

		validUntilDate := pgtype.Date{Valid: false}
		if mandateDate.Valid {
			validUntilTime := mandateDate.Time.AddDate(0, 36, 0)
			validUntilDate = pgtype.Date{Time: validUntilTime, Valid: true}
		}

		sepaGranted := false
		if req.SepaMandateGranted != nil {
			sepaGranted = *req.SepaMandateGranted
		}

		accountHolder := ""
		if req.AccountHolder != nil {
			accountHolder = *req.AccountHolder
		}

		mandateType := "basic"
		if req.MandateType != nil && *req.MandateType != "" {
			mandateType = *req.MandateType
		}

		mandateKind := "open_ended"
		if req.MandateKind != nil {
			switch *req.MandateKind {
			case "recurrent":
				mandateKind = "open_ended"
			case "one_off":
				mandateKind = "one_time"
			case "open_ended", "one_time", "last":
				mandateKind = *req.MandateKind
			}
		}

		_, err := qtx.CreateMemberBankAccount(r.Context(), database.CreateMemberBankAccountParams{
			MemberID:             member.ID,
			AccountHolder:        accountHolder,
			Iban:                 *req.IBAN,
			Bic:                  pgtype.Text{Valid: false}, // Optional
			SepaMandateAvailable: sepaGranted,
			MandateReference:     stringToPgText(req.MandateReference),
			MandateType:          mandateType,
			MandateIssuedAt:      mandateDate,
			MandateKind:          mandateKind,
			NextDirectDebitType:  "first", // Default
			LastUsedAt:           pgtype.Date{Valid: false},
			MandateValidUntil:    validUntilDate,
		})
		if err != nil {
			http.Error(w, "Failed to create bank account: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create Membership Fee
	hasFeeIntent := (req.PaymentMethod != nil && *req.PaymentMethod != "") || (req.FeeAmount != nil)
	if hasFeeIntent {
		// Only if we have fee details. If not, maybe we skip or use defaults?
		// Assuming frontend sends essential fee data if it wants to create one.
		// PaymentMethod is "sepa", "invoice" etc.

		feeLabel := "Standard"
		if req.FeeLabel != nil {
			feeLabel = *req.FeeLabel
		}

		feeType := "contribution"
		if req.FeeType != nil {
			feeType = *req.FeeType
		}

		assignment := "1_ideel"
		if req.FeeAssignment != nil {
			assignment = *req.FeeAssignment
		}

		amountNum := pgtype.Numeric{Valid: false}
		if req.FeeAmount != nil {
			s := fmt.Sprintf("%.2f", *req.FeeAmount)
			amountNum.Scan(s)
		}
		if !amountNum.Valid {
			// If we default, what amount? DB requires > 0.
			// Let's assume if they didn't provide amount, we can't create a fee.
			http.Error(w, "Membership Fee: Amount is required", http.StatusBadRequest)
			return
		}

		period := "yearly"
		if req.FeePeriod != nil {
			period = *req.FeePeriod
		}

		maturity := pgtype.Date{Valid: false}
		if req.FeeMaturity != nil {
			t, err := time.Parse("2006-01-02", *req.FeeMaturity)
			if err == nil {
				maturity = pgtype.Date{Time: t, Valid: true}
			}
		}

		paymentMethod := "transfer" // Default
		if req.PaymentMethod != nil && *req.PaymentMethod != "" {
			paymentMethod = *req.PaymentMethod
		} else if req.SepaMandateGranted != nil && *req.SepaMandateGranted {
			paymentMethod = "sepa"
		}

		startsAt := pgtype.Date{Time: time.Now(), Valid: true} // Default today
		if req.FeeStartsAt != nil {
			t, err := time.Parse("2006-01-02", *req.FeeStartsAt)
			if err == nil {
				startsAt = pgtype.Date{Time: t, Valid: true}
			}
		} else {
			// fallback to joinedAt
			startsAt = pgtype.Date{Time: joinedAt, Valid: true}
		}

		var creditorAccountID pgtype.UUID
		if req.CreditorAccountID != nil && *req.CreditorAccountID != "" {
			uid, err := uuid.Parse(*req.CreditorAccountID)
			if err == nil {
				creditorAccountID = uuidToPgtype(uid)
			}
		}

		_, err := qtx.CreateMembershipFee(r.Context(), database.CreateMembershipFeeParams{
			MemberID:          member.ID,
			FeeLabel:          feeLabel,
			FeeType:           feeType,
			Assignment:        assignment,
			Amount:            amountNum,
			Period:            period,
			MaturityDate:      maturity,
			PaymentMethod:     paymentMethod,
			StartsAt:          startsAt,
			EndsAt:            pgtype.Date{Valid: false},
			CreditorAccountID: creditorAccountID,
		})
		if err != nil {
			// If duplicate key (member_id, starts_at) -> likely not an issue for fresh member, but good to know
			http.Error(w, "Failed to create membership fee: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Fetch full details to return
	details, err := s.Queries.GetMemberDetails(r.Context(), database.GetMemberDetailsParams{
		ID:     member.ID,
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Member created but failed to fetch details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := mapMemberDetailToResponse(details)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	members, err := s.Queries.ListMembers(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(members)
}

func (s *Server) handleGetMember(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	memberID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	member, err := s.Queries.GetMemberDetails(r.Context(), database.GetMemberDetailsParams{
		ID:     uuidToPgtype(memberID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Member not found "+err.Error(), http.StatusNotFound)
		return
	}

	response := mapMemberDetailToResponse(member)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleUpdateMember(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	memberID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	var req CreateMemberRequest // Reusing Create request for now
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var birthDate pgtype.Date
	if req.BirthDate != nil && *req.BirthDate != "" {
		t, err := time.Parse("2006-01-02", *req.BirthDate)
		if err != nil {
			http.Error(w, "Invalid birth date format", http.StatusBadRequest)
			return
		}
		birthDate = pgtype.Date{Time: t, Valid: true}
	}

	var joinedAt pgtype.Date
	if req.JoinedAt != nil && *req.JoinedAt != "" {
		t, err := time.Parse("2006-01-02", *req.JoinedAt)
		if err != nil {
			http.Error(w, "Invalid joined at date format", http.StatusBadRequest)
			return
		}
		joinedAt = pgtype.Date{Time: t, Valid: true}
	}

	var memberUntil pgtype.Date
	if req.MemberUntil != nil {
		t, err := time.Parse("2006-01-02", *req.MemberUntil)
		if err == nil {
			memberUntil = pgtype.Date{Time: t, Valid: true}
		}
	}

	tx, err := s.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to start transaction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	qtx := s.Queries.WithTx(tx)

	var assignedBankID pgtype.UUID
	if req.AssignedClubBankID != nil && *req.AssignedClubBankID != "" {
		if uid, err := uuid.Parse(*req.AssignedClubBankID); err == nil {
			assignedBankID = uuidToPgtype(uid)
		}
	}

	arg := database.UpdateMemberParams{
		ID:                 uuidToPgtype(memberID),
		ClubID:             uuidToPgtype(clubID),
		MemberNumber:       req.MemberNumber,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		BirthDate:          birthDate,
		Gender:             pgtype.Text{String: req.Gender, Valid: req.Gender != ""},
		StreetHouseNumber:  stringToPgText(req.StreetHouseNumber),
		PostalCode:         stringToPgText(req.PostalCode),
		City:               stringToPgText(req.City),
		Honorary:           req.Honorary,
		Status:             req.Status,
		Salutation:         stringToPgText(req.Salutation),
		LetterSalutation:   stringToPgText(req.LetterSalutation),
		Phone1:             stringToPgText(req.Phone1),
		Phone1Note:         stringToPgText(req.Phone1Note),
		Phone2:             stringToPgText(req.Phone2),
		Phone2Note:         stringToPgText(req.Phone2Note),
		Mobile:             stringToPgText(req.Mobile),
		MobileNote:         stringToPgText(req.MobileNote),
		Email:              stringToPgText(req.Email),
		EmailNote:          stringToPgText(req.EmailNote),
		Nation:             stringToPgText(req.Nation),
		JoinedAt:           joinedAt,
		MemberUntil:        memberUntil,
		Note:               stringToPgText(req.Note),
		MaritalStatus:      stringToPgText(req.MaritalStatus),
		Title:              stringToPgText(req.Title),
		AssignedClubBankID: assignedBankID,
	}

	member, err := qtx.UpdateMember(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create/Update Bank Account (inserts new history record)
	if req.IBAN != nil && *req.IBAN != "" {
		mandateDate := pgtype.Date{Valid: false}
		if req.MandateGrantedAt != nil && *req.MandateGrantedAt != "" {
			t, err := time.Parse("2006-01-02", *req.MandateGrantedAt)
			if err == nil {
				mandateDate = pgtype.Date{Time: t, Valid: true}
			}
		}

		validUntilDate := pgtype.Date{Valid: false}
		if mandateDate.Valid {
			validUntilTime := mandateDate.Time.AddDate(0, 36, 0)
			validUntilDate = pgtype.Date{Time: validUntilTime, Valid: true}
		}

		sepaGranted := false
		if req.SepaMandateGranted != nil {
			sepaGranted = *req.SepaMandateGranted
		}

		accountHolder := ""
		if req.AccountHolder != nil {
			accountHolder = *req.AccountHolder
		}

		mandateType := "basic"
		if req.MandateType != nil && *req.MandateType != "" {
			mandateType = *req.MandateType
		}

		mandateKind := "open_ended"
		if req.MandateKind != nil {
			switch *req.MandateKind {
			case "recurrent":
				mandateKind = "open_ended"
			case "one_off":
				mandateKind = "one_time"
			case "open_ended", "one_time", "last":
				mandateKind = *req.MandateKind
			}
		}

		var bankAccountID pgtype.UUID
		shouldUpdate := false

		// Case 1: ID provided
		if req.BankAccountID != nil && *req.BankAccountID != "" {
			uid, err := uuid.Parse(*req.BankAccountID)
			if err == nil {
				bankAccountID = uuidToPgtype(uid)
				shouldUpdate = true
			}
		}

		// Case 2: No ID, check if IBAN exists
		if !shouldUpdate {
			existing, err := qtx.GetMemberBankAccountByIBAN(r.Context(), database.GetMemberBankAccountByIBANParams{
				MemberID: member.ID,
				Iban:     *req.IBAN,
			})
			if err == nil {
				bankAccountID = existing.ID
				shouldUpdate = true
			}
		}

		if shouldUpdate {
			_, err := qtx.UpdateMemberBankAccount(r.Context(), database.UpdateMemberBankAccountParams{
				ID:                   bankAccountID,
				MemberID:             member.ID,
				AccountHolder:        accountHolder,
				Iban:                 *req.IBAN,
				Bic:                  pgtype.Text{Valid: false},
				SepaMandateAvailable: sepaGranted,
				MandateReference:     stringToPgText(req.MandateReference),
				MandateType:          mandateType,
				MandateIssuedAt:      mandateDate,
				MandateKind:          mandateKind,
				NextDirectDebitType:  "first",
				LastUsedAt:           pgtype.Date{Valid: false},
				MandateValidUntil:    validUntilDate,
			})
			if err != nil {
				http.Error(w, "Failed to update bank account: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			_, err := qtx.CreateMemberBankAccount(r.Context(), database.CreateMemberBankAccountParams{
				MemberID:             member.ID,
				AccountHolder:        accountHolder,
				Iban:                 *req.IBAN,
				Bic:                  pgtype.Text{Valid: false}, // Optional
				SepaMandateAvailable: sepaGranted,
				MandateReference:     stringToPgText(req.MandateReference),
				MandateType:          mandateType,
				MandateIssuedAt:      mandateDate,
				MandateKind:          mandateKind,
				NextDirectDebitType:  "first", // Default
				LastUsedAt:           pgtype.Date{Valid: false},
				MandateValidUntil:    validUntilDate,
			})
			if err != nil {
				http.Error(w, "Failed to create bank account: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Create/Update Membership Fee (inserts new history record)
	hasFeeChanges := (req.PaymentMethod != nil && *req.PaymentMethod != "") ||
		(req.FeeAmount != nil) ||
		(req.FeeLabel != nil) ||
		(req.FeeType != nil) ||
		(req.CreditorAccountID != nil)

	if hasFeeChanges {
		// Fetch latest fee for defaults (history)
		latest, err := qtx.GetLatestMembershipFee(r.Context(), member.ID)
		foundLatest := err == nil

		feeLabel := "Standard"
		if foundLatest {
			feeLabel = latest.FeeLabel
		}
		if req.FeeLabel != nil {
			feeLabel = *req.FeeLabel
		}

		feeType := "contribution"
		if foundLatest {
			feeType = latest.FeeType
		}
		if req.FeeType != nil {
			feeType = *req.FeeType
		}

		assignment := "1_ideel"
		if foundLatest {
			assignment = latest.Assignment
		}
		if req.FeeAssignment != nil {
			assignment = *req.FeeAssignment
		}

		amountNum := pgtype.Numeric{Valid: false}
		if foundLatest {
			amountNum = latest.Amount
		}
		if req.FeeAmount != nil {
			s := fmt.Sprintf("%.2f", *req.FeeAmount)
			amountNum.Scan(s)
		}
		if !amountNum.Valid {
			http.Error(w, "Fee amount is required", http.StatusBadRequest)
			return
		}

		period := "yearly"
		if foundLatest {
			period = latest.Period
		}
		if req.FeePeriod != nil {
			period = *req.FeePeriod
		}

		maturity := pgtype.Date{Valid: false}
		if foundLatest {
			maturity = latest.MaturityDate
		}
		if req.FeeMaturity != nil {
			t, err := time.Parse("2006-01-02", *req.FeeMaturity)
			if err == nil {
				maturity = pgtype.Date{Time: t, Valid: true}
			}
		}

		paymentMethod := ""
		if foundLatest {
			paymentMethod = latest.PaymentMethod
		}
		if req.PaymentMethod != nil && *req.PaymentMethod != "" {
			paymentMethod = *req.PaymentMethod
		}
		if paymentMethod == "" {
			http.Error(w, "Payment method is required", http.StatusBadRequest)
			return
		}

		startsAt := pgtype.Date{Time: time.Now(), Valid: true} // Default today for updates
		if req.FeeStartsAt != nil {
			t, err := time.Parse("2006-01-02", *req.FeeStartsAt)
			if err == nil {
				startsAt = pgtype.Date{Time: t, Valid: true}
			}
		}

		var creditorAccountID pgtype.UUID
		if foundLatest {
			creditorAccountID = latest.CreditorAccountID
		}
		if req.CreditorAccountID != nil {
			if *req.CreditorAccountID == "" {
				creditorAccountID = pgtype.UUID{Valid: false}
			} else {
				uid, err := uuid.Parse(*req.CreditorAccountID)
				if err == nil {
					creditorAccountID = uuidToPgtype(uid)
				}
			}
		}

		_, err = qtx.CreateMembershipFee(r.Context(), database.CreateMembershipFeeParams{
			MemberID:          member.ID,
			FeeLabel:          feeLabel,
			FeeType:           feeType,
			Assignment:        assignment,
			Amount:            amountNum,
			Period:            period,
			MaturityDate:      maturity,
			PaymentMethod:     paymentMethod,
			StartsAt:          startsAt,
			EndsAt:            pgtype.Date{Valid: false},
			CreditorAccountID: creditorAccountID,
		})
		if err != nil {
			// If duplicate key (member_id, starts_at), it means a fee change already happened today or for this start date.
			// We might want to swallow this error or return it.
			// For now, returning it ensures we don't silently fail to save if something is wrong.
			http.Error(w, "Failed to update membership fee: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Fetch full details
	details, err := s.Queries.GetMemberDetails(r.Context(), database.GetMemberDetailsParams{
		ID:     member.ID,
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Member updated but failed to fetch details: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := mapMemberDetailToResponse(details)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleDeleteMember(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	memberID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid member ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteMember(r.Context(), database.DeleteMemberParams{
		ID:     uuidToPgtype(memberID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete member", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func stringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func (s *Server) handleGetMemberStatistics(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	yearStr := r.URL.Query().Get("year")
	if yearStr == "" {
		http.Error(w, "year parameter is required", http.StatusBadRequest)
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		http.Error(w, "invalid year parameter", http.StatusBadRequest)
		return
	}

	stats, err := s.Queries.GetMemberStatistics(r.Context(), database.GetMemberStatisticsParams{
		ClubID:  pgtype.UUID{Bytes: clubID, Valid: true},
		Column2: int32(year),
	})
	if err != nil {
		http.Error(w, "internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "failed to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

type BirthdayMember struct {
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	BirthDate string    `json:"birth_date"`
	Date      string    `json:"date"`
	NewAge    int       `json:"new_age"`
	rawDate   time.Time `json:"-"`
}

func (s *Server) getBirthdayList(r *http.Request) (int, []BirthdayMember, error) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		return 0, nil, fmt.Errorf("club ID not found in context")
	}

	yearStr := r.URL.Query().Get("year")
	year := time.Now().Year()
	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid year parameter")
		}
	}

	milestonesStr := r.URL.Query().Get("milestones")
	milestones := []int{50, 60, 70, 80, 90, 100}
	if milestonesStr != "" {
		parts := strings.Split(milestonesStr, ",")
		var customMilestones []int
		for _, p := range parts {
			m, err := strconv.Atoi(strings.TrimSpace(p))
			if err == nil {
				customMilestones = append(customMilestones, m)
			}
		}
		if len(customMilestones) > 0 {
			milestones = customMilestones
		}
	}

	members, err := s.Queries.ListMembers(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list members: %w", err)
	}

	var result []BirthdayMember
	for _, m := range members {
		if !m.BirthDate.Valid {
			continue
		}
		bd := m.BirthDate.Time
		age := year - bd.Year()

		isMilestone := false
		for _, ms := range milestones {
			if age == ms {
				isMilestone = true
				break
			}
		}

		if isMilestone {
			// Handle leap years properly if born on Feb 29
			month := bd.Month()
			day := bd.Day()
			targetDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

			// If invalid date (e.g. Feb 29 in non-leap year), time.Date normalizes it to March 1
			// This is usually acceptable for birthday lists, or we could handle strictly.
			// Go's time.Date: "February 29 in a non-leap year... Normalizes to March 1"

			result = append(result, BirthdayMember{
				FirstName: m.FirstName,
				LastName:  m.LastName,
				BirthDate: bd.Format("2006-01-02"),
				Date:      targetDate.Format("2006-01-02"),
				NewAge:    age,
				rawDate:   targetDate,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].rawDate.Before(result[j].rawDate)
	})

	return year, result, nil
}

func (s *Server) handleGetMemberBirthdays(w http.ResponseWriter, r *http.Request) {
	_, members, err := s.getBirthdayList(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(members)
}

func (s *Server) handleGetMemberBirthdaysPDF(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	year, members, err := s.getBirthdayList(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to fetch club details", http.StatusInternalServerError)
		return
	}

	var pdfEntries []pdf.BirthdayEntry
	for _, m := range members {
		pdfEntries = append(pdfEntries, pdf.BirthdayEntry{
			Date:      m.rawDate,
			FirstName: m.FirstName,
			LastName:  m.LastName,
			NewAge:    m.NewAge,
		})
	}

	pdfBytes, err := pdf.GenerateBirthdayList(club.Name, year, pdfEntries)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"birthdays_%d.pdf\"", year))
	w.Write(pdfBytes)
}

type AnniversaryMember struct {
	FirstName       string
	LastName        string
	JoinedAt        string
	AnniversaryDate string
	MembershipYears int
	rawDate         time.Time
}

func (s *Server) getAnniversaryList(r *http.Request) (int, []AnniversaryMember, error) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		return 0, nil, fmt.Errorf("club ID not found in context")
	}

	yearStr := r.URL.Query().Get("year")
	year := time.Now().Year()
	if yearStr != "" {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			return 0, nil, fmt.Errorf("invalid year parameter")
		}
	}

	yearsStr := r.URL.Query().Get("years")
	targetYears := []int{25, 30, 40, 50, 60}
	if yearsStr != "" {
		parts := strings.Split(yearsStr, ",")
		var customYears []int
		for _, p := range parts {
			y, err := strconv.Atoi(strings.TrimSpace(p))
			if err == nil {
				customYears = append(customYears, y)
			}
		}
		if len(customYears) > 0 {
			targetYears = customYears
		}
	}

	members, err := s.Queries.ListMembers(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list members: %w", err)
	}

	var result []AnniversaryMember
	for _, m := range members {
		if !m.JoinedAt.Valid {
			continue
		}
		joined := m.JoinedAt.Time
		membershipYears := year - joined.Year()

		isAnniversary := false
		for _, target := range targetYears {
			if membershipYears == target {
				isAnniversary = true
				break
			}
		}

		if isAnniversary {
			month := joined.Month()
			day := joined.Day()
			targetDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

			result = append(result, AnniversaryMember{
				FirstName:       m.FirstName,
				LastName:        m.LastName,
				JoinedAt:        joined.Format("2006-01-02"),
				AnniversaryDate: targetDate.Format("2006-01-02"),
				MembershipYears: membershipYears,
				rawDate:         targetDate,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].rawDate.Before(result[j].rawDate)
	})

	return year, result, nil
}

func (s *Server) handleGetMemberAnniversaries(w http.ResponseWriter, r *http.Request) {
	_, members, err := s.getAnniversaryList(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(members)
}

func (s *Server) handleGetMemberAnniversariesPDF(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	year, members, err := s.getAnniversaryList(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to fetch club details", http.StatusInternalServerError)
		return
	}

	var pdfEntries []pdf.AnniversaryEntry
	for _, m := range members {
		pdfEntries = append(pdfEntries, pdf.AnniversaryEntry{
			Date:            m.rawDate,
			FirstName:       m.FirstName,
			LastName:        m.LastName,
			MembershipYears: m.MembershipYears,
		})
	}

	pdfBytes, err := pdf.GenerateAnniversaryList(club.Name, year, pdfEntries)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"anniversaries_%d.pdf\"", year))
	w.Write(pdfBytes)
}

// --------------------------------------------------------------------------------
// Response DTOs and Helpers to ensure API contract stability
// --------------------------------------------------------------------------------

type MemberResponse struct {
	ID                 string  `json:"id"`
	ClubID             string  `json:"club_id"`
	MemberNumber       string  `json:"member_number"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	BirthDate          *string `json:"birth_date"`
	Gender             *string `json:"gender"`
	StreetHouseNumber  *string `json:"street_house_number"`
	PostalCode         *string `json:"postal_code"`
	City               *string `json:"city"`
	Honorary           bool    `json:"honorary"`
	Status             string  `json:"status"`
	Salutation         *string `json:"salutation"`
	LetterSalutation   *string `json:"letter_salutation"`
	Phone1             *string `json:"phone1"`
	Phone1Note         *string `json:"phone1_note"`
	Phone2             *string `json:"phone2"`
	Phone2Note         *string `json:"phone2_note"`
	Mobile             *string `json:"mobile"`
	MobileNote         *string `json:"mobile_note"`
	Email              *string `json:"email"`
	EmailNote          *string `json:"email_note"`
	Nation             *string `json:"nation"`
	JoinedAt           *string `json:"joined_at"`
	MemberUntil        *string `json:"member_until"`
	Note               *string `json:"note"`
	MaritalStatus      *string `json:"marital_status"`
	Title              *string `json:"title"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	UserID             string  `json:"user_id"`
	BankAccountID      string  `json:"bank_account_id"`
	Iban               string  `json:"iban"`
	AccountHolder      string  `json:"account_holder"`
	SepaMandateGranted bool    `json:"sepa_mandate_granted"`
	MandateReference   *string `json:"mandate_reference"`
	MandateGrantedAt   *string `json:"mandate_granted_at"`
	PaymentMethod      string  `json:"payment_method"`
	FeeLabel           string  `json:"fee_label"`
	FeeAmount          float64 `json:"fee_amount"`
	FeePeriod          string  `json:"fee_period"`
	FeeStartsAt        *string `json:"fee_starts_at"`
	FeeMaturity        *string `json:"fee_maturity"`
	CreditorAccountID  string  `json:"creditor_account_id"`
	AssignedClubBankID string  `json:"assigned_club_bank_id"`
}

func mapMemberDetailToResponse(m database.GetMemberDetailsRow) MemberResponse {
	return MemberResponse{
		ID:                 uuidToString(m.ID),
		ClubID:             uuidToString(m.ClubID),
		MemberNumber:       m.MemberNumber,
		FirstName:          m.FirstName,
		LastName:           m.LastName,
		BirthDate:          dateToString(m.BirthDate),
		Gender:             textToString(m.Gender),
		StreetHouseNumber:  textToString(m.StreetHouseNumber),
		PostalCode:         textToString(m.PostalCode),
		City:               textToString(m.City),
		Honorary:           m.Honorary,
		Status:             m.Status,
		Salutation:         textToString(m.Salutation),
		LetterSalutation:   textToString(m.LetterSalutation),
		Phone1:             textToString(m.Phone1),
		Phone1Note:         textToString(m.Phone1Note),
		Phone2:             textToString(m.Phone2),
		Phone2Note:         textToString(m.Phone2Note),
		Mobile:             textToString(m.Mobile),
		MobileNote:         textToString(m.MobileNote),
		Email:              textToString(m.Email),
		EmailNote:          textToString(m.EmailNote),
		Nation:             textToString(m.Nation),
		JoinedAt:           dateToString(m.JoinedAt),
		MemberUntil:        dateToString(m.MemberUntil),
		Note:               textToString(m.Note),
		MaritalStatus:      textToString(m.MaritalStatus),
		Title:              textToString(m.Title),
		CreatedAt:          timestampToString(m.CreatedAt),
		UpdatedAt:          timestampToString(m.UpdatedAt),
		UserID:             uuidToString(m.UserID),
		AssignedClubBankID: uuidToString(m.AssignedClubBankID),
		BankAccountID:      uuidToString(m.BankAccountID),

		Iban:               m.Iban,
		AccountHolder:      m.AccountHolder,
		SepaMandateGranted: m.SepaMandateGranted,
		MandateReference:   &m.MandateReference,
		MandateGrantedAt:   dateToString(m.MandateGrantedAt),
		PaymentMethod:      m.PaymentMethod,
		FeeLabel:           m.FeeLabel,
		FeeAmount:          m.FeeAmount,
		FeePeriod:          m.FeePeriod,
		FeeStartsAt:        dateToString(m.FeeStartsAt),
		FeeMaturity:        dateToString(m.FeeMaturity),
		CreditorAccountID:  uuidToString(m.CreditorAccountID),
	}
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

func dateToString(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func textToString(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func timestampToString(t pgtype.Timestamptz) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}

func dateToGermanString(t pgtype.Date) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("02.01.2006")
}

func (s *Server) handleExportMembersCSV(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	members, err := s.Queries.ListMemberDetails(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list members", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"members.csv\"")

	writer := csv.NewWriter(w)
	// Comma separated as per import support (though import supports comma, German usually uses semicolon. Import code: reader.Comma = ',')
	// We'll stick to Comma as per Import handler configuration.
	defer writer.Flush()

	header := []string{
		"Mitgliedsnr.", "Vorname", "Nachname", "Geburtstag", "Geschlecht",
		"Strasse & Hausnr.", "PLZ", "Ort", "Ehrenmitglied", "Status", "Anrede",
		"Briefanrede", "Telefon", "Telefon Notiz", "Telefon 2", "Telefon 2 Notiz",
		"Mobil", "Mobil Notiz", "E-Mail", "E-Mail Notiz", "Land", "Mitglied seit",
		"Mitglied bis", "Notizen", "Familienstand", "Titel",
		"IBAN", "Kontoinhaber", "SEPA-Mandat vorhanden", "Mandatsreferenz", "Mandat erteilt am",
		"Zahlungsart", "Beitrag (Bezeichnung)", "Beitrag (Betrag)", "Beitrag (Zeitraum)", "Beitrag (Start)", "Beitrag (Fälligkeit)", "Creditor Account ID",
	}

	if err := writer.Write(header); err != nil {
		http.Error(w, "Failed to write CSV header", http.StatusInternalServerError)
		return
	}

	for _, m := range members {
		honorary := "Nein"
		if m.Honorary {
			honorary = "Ja"
		}

		sepa := "Nein"
		if m.SepaMandateGranted {
			sepa = "Ja"
		}

		amountStr := fmt.Sprintf("%.2f", m.FeeAmount)
		amountStr = strings.Replace(amountStr, ".", ",", 1) // German decimal separator

		creditorAccountIDStr := ""
		if m.CreditorAccountID.Valid {
			creditorAccountIDStr = uuid.UUID(m.CreditorAccountID.Bytes).String()
		}

		record := []string{
			m.MemberNumber,
			m.FirstName,
			m.LastName,
			dateToGermanString(m.BirthDate),
			m.Gender.String,
			m.StreetHouseNumber.String,
			m.PostalCode.String,
			m.City.String,
			honorary,
			m.Status,
			m.Salutation.String,
			m.LetterSalutation.String,
			m.Phone1.String,
			m.Phone1Note.String,
			m.Phone2.String,
			m.Phone2Note.String,
			m.Mobile.String,
			m.MobileNote.String,
			m.Email.String,
			m.EmailNote.String,
			m.Nation.String,
			dateToGermanString(m.JoinedAt),
			dateToGermanString(m.MemberUntil),
			m.Note.String,
			m.MaritalStatus.String,
			m.Title.String,
			m.Iban,
			m.AccountHolder,
			sepa,
			m.MandateReference,
			dateToGermanString(m.MandateGrantedAt),
			m.PaymentMethod,
			m.FeeLabel,
			amountStr,
			m.FeePeriod,
			dateToGermanString(m.FeeStartsAt),
			dateToGermanString(m.FeeMaturity),
			creditorAccountIDStr,
		}

		if err := writer.Write(record); err != nil {
			http.Error(w, "Failed to write CSV record", http.StatusInternalServerError)
			return
		}
	}
}

func (s *Server) handleExportMembersContactPDF(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	members, err := s.Queries.ListMemberDetails(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list members", http.StatusInternalServerError)
		return
	}

	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to fetch club details", http.StatusInternalServerError)
		return
	}

	var entries []pdf.ContactEntry
	for _, m := range members {
		entries = append(entries, pdf.ContactEntry{
			MemberNumber:      m.MemberNumber,
			FirstName:         m.FirstName,
			LastName:          m.LastName,
			StreetHouseNumber: m.StreetHouseNumber.String,
			PostalCode:        m.PostalCode.String,
			City:              m.City.String,
			Phone1:            m.Phone1.String,
			Mobile:            m.Phone2.String,
			Email:             m.Email.String,
		})
	}

	pdfBytes, err := pdf.GenerateContactList(club.Name, entries)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"kontaktliste.pdf\"")
	w.Write(pdfBytes)
}
