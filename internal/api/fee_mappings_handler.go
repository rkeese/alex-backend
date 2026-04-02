package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/rkeese/alex-backend/internal/database"
)

type CreateFeeAccountMappingRequest struct {
	FeeType           string `json:"fee_type"`
	ClubBankAccountID string `json:"club_bank_account_id"`
}

type UpdateFeeAccountMappingRequest struct {
	ClubBankAccountID string `json:"club_bank_account_id"`
}

func (s *Server) handleCreateFeeAccountMapping(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateFeeAccountMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bankAccountID, err := uuid.Parse(req.ClubBankAccountID)
	if err != nil {
		http.Error(w, "Invalid bank account ID", http.StatusBadRequest)
		return
	}

	arg := database.CreateFeeAccountMappingParams{
		ClubID:            uuidToPgtype(clubID),
		FeeType:           req.FeeType,
		ClubBankAccountID: uuidToPgtype(bankAccountID),
	}

	mapping, err := s.Queries.CreateFeeAccountMapping(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapping)
}

func (s *Server) handleListFeeAccountMappings(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	mappings, err := s.Queries.ListFeeAccountMappings(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list mappings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mappings)
}

func (s *Server) handleUpdateFeeAccountMapping(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	// FeeType is in the path, but standard http.ServeMux in Go 1.22+ supports path values
	feeType := r.PathValue("feeType")
	if feeType == "" {
		http.Error(w, "Fee Type is required", http.StatusBadRequest)
		return
	}

	var req UpdateFeeAccountMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	bankAccountID, err := uuid.Parse(req.ClubBankAccountID)
	if err != nil {
		http.Error(w, "Invalid bank account ID", http.StatusBadRequest)
		return
	}

	arg := database.UpdateFeeAccountMappingParams{
		ClubID:            uuidToPgtype(clubID),
		FeeType:           feeType,
		ClubBankAccountID: uuidToPgtype(bankAccountID),
	}

	mapping, err := s.Queries.UpdateFeeAccountMapping(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mapping)
}

func (s *Server) handleDeleteFeeAccountMapping(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	feeType := r.PathValue("feeType")
	if feeType == "" {
		http.Error(w, "Fee Type is required", http.StatusBadRequest)
		return
	}

	err := s.Queries.DeleteFeeAccountMapping(r.Context(), database.DeleteFeeAccountMappingParams{
		ClubID:  uuidToPgtype(clubID),
		FeeType: feeType,
	})
	if err != nil {
		http.Error(w, "Failed to delete mapping", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
