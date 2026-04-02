package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
)

const MaxUploadSize = 10 << 20 // 10 MB

// =====================
// Document Categories
// =====================

func (s *Server) handleCreateDocumentCategory(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int32  `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	cat, err := s.Queries.CreateDocumentCategory(r.Context(), database.CreateDocumentCategoryParams{
		ClubID:      uuidToPgtype(clubID),
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		http.Error(w, "Failed to create category: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cat)
}

func (s *Server) handleListDocumentCategories(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	cats, err := s.Queries.ListDocumentCategories(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

func (s *Server) handleUpdateDocumentCategory(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	catID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int32  `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Category name is required", http.StatusBadRequest)
		return
	}

	cat, err := s.Queries.UpdateDocumentCategory(r.Context(), database.UpdateDocumentCategoryParams{
		ID:          uuidToPgtype(catID),
		ClubID:      uuidToPgtype(clubID),
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		http.Error(w, "Failed to update category: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cat)
}

func (s *Server) handleDeleteDocumentCategory(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	catID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteDocumentCategory(r.Context(), database.DeleteDocumentCategoryParams{
		ID:     uuidToPgtype(catID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete category", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// =====================
// Documents
// =====================

func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadSize)
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		http.Error(w, "File too big", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	// Optional category_id from form field
	var categoryID pgtype.UUID
	if catStr := r.FormValue("category_id"); catStr != "" {
		parsed, err := uuid.Parse(catStr)
		if err != nil {
			http.Error(w, "Invalid category_id", http.StatusBadRequest)
			return
		}
		categoryID = uuidToPgtype(parsed)
	}

	// Optional description from form field
	description := pgtype.Text{}
	if desc := r.FormValue("description"); desc != "" {
		description = pgtype.Text{String: desc, Valid: true}
	}

	arg := database.CreateDocumentParams{
		ClubID:      uuidToPgtype(clubID),
		Name:        header.Filename,
		Content:     fileBytes,
		CategoryID:  categoryID,
		Description: description,
	}

	doc, err := s.Queries.CreateDocument(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to upload document: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(doc)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	// Optional filter by category
	if catStr := r.URL.Query().Get("category_id"); catStr != "" {
		catID, err := uuid.Parse(catStr)
		if err != nil {
			http.Error(w, "Invalid category_id", http.StatusBadRequest)
			return
		}
		docs, err := s.Queries.ListDocumentsByCategory(r.Context(), database.ListDocumentsByCategoryParams{
			ClubID:     uuidToPgtype(clubID),
			CategoryID: uuidToPgtype(catID),
		})
		if err != nil {
			http.Error(w, "Failed to list documents", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docs)
		return
	}

	docs, err := s.Queries.ListDocuments(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(docs)
}

func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	docID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		CategoryID  *string `json:"category_id"`
		Description string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Document name is required", http.StatusBadRequest)
		return
	}

	var categoryID pgtype.UUID
	if req.CategoryID != nil && *req.CategoryID != "" {
		parsed, err := uuid.Parse(*req.CategoryID)
		if err != nil {
			http.Error(w, "Invalid category_id", http.StatusBadRequest)
			return
		}
		categoryID = uuidToPgtype(parsed)
	}

	doc, err := s.Queries.UpdateDocument(r.Context(), database.UpdateDocumentParams{
		ID:          uuidToPgtype(docID),
		ClubID:      uuidToPgtype(clubID),
		Name:        req.Name,
		CategoryID:  categoryID,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, "Failed to update document: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (s *Server) handleDownloadDocument(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	doc, err := s.Queries.GetDocument(r.Context(), database.GetDocumentParams{
		ID:     uuidToPgtype(docID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+doc.Name)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(doc.Content)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	docID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteDocument(r.Context(), database.DeleteDocumentParams{
		ID:     uuidToPgtype(docID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
