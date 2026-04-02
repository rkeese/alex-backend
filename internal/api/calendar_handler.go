package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/rkeese/alex-backend/internal/pdf"
)

type CreateEventRequest struct {
	Date        string `json:"date"` // YYYY-MM-DD
	Time        string `json:"time"` // HH:MM:SS or HH:MM
	Description string `json:"description"`
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	// Parse time. Assuming "15:04" or "15:04:05"
	// pgtype.Time stores microseconds since midnight.
	parsedTime, err := time.Parse("15:04", req.Time)
	if err != nil {
		parsedTime, err = time.Parse("15:04:05", req.Time)
		if err != nil {
			http.Error(w, "Invalid time format", http.StatusBadRequest)
			return
		}
	}

	// Convert to microseconds since midnight
	microseconds := int64(parsedTime.Hour())*3600*1000000 + int64(parsedTime.Minute())*60*1000000 + int64(parsedTime.Second())*1000000

	arg := database.CreateEventParams{
		ClubID:      uuidToPgtype(clubID),
		Date:        pgtype.Date{Time: date, Valid: true},
		Time:        pgtype.Time{Microseconds: microseconds, Valid: true},
		Description: req.Description,
	}

	event, err := s.Queries.CreateEvent(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to create event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	events, err := s.Queries.ListEvents(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to list events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

func (s *Server) handleUpdateEvent(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	parsedTime, err := time.Parse("15:04", req.Time)
	if err != nil {
		parsedTime, err = time.Parse("15:04:05", req.Time)
		if err != nil {
			http.Error(w, "Invalid time format", http.StatusBadRequest)
			return
		}
	}
	microseconds := int64(parsedTime.Hour())*3600*1000000 + int64(parsedTime.Minute())*60*1000000 + int64(parsedTime.Second())*1000000

	arg := database.UpdateEventParams{
		ID:          uuidToPgtype(eventID),
		ClubID:      uuidToPgtype(clubID),
		Date:        pgtype.Date{Time: date, Valid: true},
		Time:        pgtype.Time{Microseconds: microseconds, Valid: true},
		Description: req.Description,
	}

	event, err := s.Queries.UpdateEvent(r.Context(), arg)
	if err != nil {
		http.Error(w, "Failed to update event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	idStr := r.PathValue("id")
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteEvent(r.Context(), database.DeleteEventParams{
		ID:     uuidToPgtype(eventID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Failed to delete event", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleExportEventsPDF(w http.ResponseWriter, r *http.Request) {
	clubID, ok := r.Context().Value(clubIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query()
	yearStr := query.Get("year")
	monthStr := query.Get("month")

	now := time.Now()
	year := now.Year()
	if yearStr != "" {
		y, err := strconv.Atoi(yearStr)
		if err != nil {
			http.Error(w, "Invalid year", http.StatusBadRequest)
			return
		}
		year = y
	}

	var fromDate, toDate time.Time
	var title string

	if monthStr != "" {
		month, err := strconv.Atoi(monthStr)
		if err != nil || month < 1 || month > 12 {
			http.Error(w, "Invalid month", http.StatusBadRequest)
			return
		}
		fromDate = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		// Last day of month: first day of next month minus 1 nanosecond
		toDate = time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
		title = fmt.Sprintf("Termine %s %d", getMonthName(month), year)
	} else {
		fromDate = time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		toDate = time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
		title = fmt.Sprintf("Termine %d", year)
	}

	events, err := s.Queries.ListEventsByDateRange(r.Context(), database.ListEventsByDateRangeParams{
		ClubID:   uuidToPgtype(clubID),
		FromDate: pgtype.Date{Time: fromDate, Valid: true},
		ToDate:   pgtype.Date{Time: toDate, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to list events", http.StatusInternalServerError)
		return
	}

	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to fetch club details", http.StatusInternalServerError)
		return
	}

	var pdfEntries []pdf.EventEntry
	for _, e := range events {
		micro := e.Time.Microseconds
		totalSeconds := micro / 1000000
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		timeStr := fmt.Sprintf("%02d:%02d", hours, minutes)

		pdfEntries = append(pdfEntries, pdf.EventEntry{
			Date:        e.Date.Time,
			Time:        timeStr,
			Description: e.Description,
		})
	}

	pdfBytes, err := pdf.GenerateEventList(club.Name, title, pdfEntries)
	if err != nil {
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"events_%d.pdf\"", year))
	w.Write(pdfBytes)
}

func getMonthName(m int) string {
	months := []string{"", "Januar", "Februar", "März", "April", "Mai", "Juni", "Juli", "August", "September", "Oktober", "November", "Dezember"}
	if m >= 1 && m <= 12 {
		return months[m]
	}
	return ""
}
