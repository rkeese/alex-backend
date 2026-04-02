package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
)

type CreateDepartmentRequest struct {
	Name        string  `json:"name"`
	Subdivision *string `json:"subdivision"`
	ParentID    *string `json:"parent_id"`
}

func (s *Server) handleCreateDepartment(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var parentID pgtype.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err == nil {
			parentID = uuidToPgtype(id)
		}
	}

	arg := database.CreateDepartmentParams{
		ClubID:      uuidToPgtype(clubID),
		Name:        req.Name,
		Subdivision: stringToPgText(req.Subdivision),
		ParentID:    parentID,
	}

	dept, err := s.Queries.CreateDepartment(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create department: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dept)
}

func (s *Server) handleListDepartments(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	depts, err := s.Queries.ListDepartments(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list departments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(depts)
}

func (s *Server) handleGetDepartment(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	deptID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid department ID", http.StatusBadRequest)
		return
	}

	dept, err := s.Queries.GetDepartmentByID(r.Context(), database.GetDepartmentByIDParams{
		ID:     uuidToPgtype(deptID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Department not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dept)
}

func (s *Server) handleUpdateDepartment(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	deptID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid department ID", http.StatusBadRequest)
		return
	}

	var req CreateDepartmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var parentID pgtype.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err == nil {
			parentID = uuidToPgtype(id)
		}
	}

	arg := database.UpdateDepartmentParams{
		ID:          uuidToPgtype(deptID),
		ClubID:      uuidToPgtype(clubID),
		Name:        req.Name,
		Subdivision: stringToPgText(req.Subdivision),
		ParentID:    parentID,
	}

	dept, err := s.Queries.UpdateDepartment(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update department: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dept)
}

func (s *Server) handleDeleteDepartment(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	deptID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid department ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteDepartment(r.Context(), database.DeleteDepartmentParams{
		ID:     uuidToPgtype(deptID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete department", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
