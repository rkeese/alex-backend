package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

type CreateClubRequest struct {
	RegisteredAssociation     bool    `json:"registered_association"`
	Name                      string  `json:"name"`
	Type                      string  `json:"type"`
	Category                  *string `json:"category"`
	Number                    *string `json:"number"`
	StreetHouseNumber         *string `json:"street_house_number"`
	PostalCode                *string `json:"postal_code"`
	City                      *string `json:"city"`
	NameExtension             *string `json:"name_extension"`
	AddressExtension          *string `json:"address_extension"`
	TaxOfficeName             *string `json:"tax_office_name"`
	TaxOfficeTaxNumber        *string `json:"tax_office_tax_number"`
	TaxOfficeAssessmentPeriod *string `json:"tax_office_assessment_period"`
	TaxOfficePurpose          *string `json:"tax_office_purpose"`
	TaxOfficeDecisionDate     *string `json:"tax_office_decision_date"`
	TaxOfficeDecisionType     *string `json:"tax_office_decision_type"`
}

func isValidClubType(t string) bool {
	switch t {
	case "sport_club", "music_club", "social_club", "environment_club", "cultural_club", "hobby_club", "rescue_service":
		return true
	default:
		return false
	}
}

func (s *Server) handleCreateClub(w http.ResponseWriter, r *http.Request) {
	var req CreateClubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidClubType(req.Type) {
		http.Error(w, "Invalid club type provided. Allowed values are: sport_club, music_club, social_club, environment_club, cultural_club, hobby_club, rescue_service", http.StatusBadRequest)
		return
	}

	var decisionDate pgtype.Date
	if req.TaxOfficeDecisionDate != nil && *req.TaxOfficeDecisionDate != "" {
		parsed, err := parseDate(*req.TaxOfficeDecisionDate)
		if err != nil {
			http.Error(w, "Invalid date format for tax_office_decision_date", http.StatusBadRequest)
			return
		}
		decisionDate = dateToPgDate(parsed)
	} else {
		decisionDate = pgtype.Date{Valid: false}
	}

	arg := database.CreateClubParams{
		RegisteredAssociation:     req.RegisteredAssociation,
		Name:                      req.Name,
		Type:                      req.Type,
		Category:                  stringToPgText(req.Category),
		Number:                    stringToPgText(req.Number),
		StreetHouseNumber:         stringToPgText(req.StreetHouseNumber),
		PostalCode:                stringToPgText(req.PostalCode),
		City:                      stringToPgText(req.City),
		AddressExtension:          stringToPgText(req.AddressExtension),
		NameExtension:             stringToPgText(req.NameExtension),
		TaxOfficeName:             stringToPgText(req.TaxOfficeName),
		TaxOfficeTaxNumber:        stringToPgText(req.TaxOfficeTaxNumber),
		TaxOfficeAssessmentPeriod: stringToPgText(req.TaxOfficeAssessmentPeriod),
		TaxOfficePurpose:          stringToPgText(req.TaxOfficePurpose),
		TaxOfficeDecisionDate:     decisionDate,
		TaxOfficeDecisionType:     stringToPgText(req.TaxOfficeDecisionType),
	}

	club, err := s.Queries.CreateClub(r.Context(), arg)
	if err != nil {
		if database.IsCheckViolationError(err) {
			http.Error(w, "Invalid club type provided (database constraint).", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to create club: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Assign 'admin' role to the creator
	userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
	if ok {
		adminRole, err := s.Queries.GetRoleByName(r.Context(), "admin")
		if err == nil {
			_ = s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
				UserID: uuidToPgtype(userID),
				RoleID: adminRole.ID,
				ClubID: club.ID,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(club)
}

func (s *Server) handleListClubs(w http.ResponseWriter, r *http.Request) {
	// Filter logic:
	// 1. SysAdmin sees all clubs.
	// 2. Normal users see only clubs they have a role in.

	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var clubs []database.Club
	var err error

	if claims.IsSysAdmin {
		clubs, err = s.Queries.ListClubs(r.Context())
	} else {
		// Filter by user's roles
		// This requires a new query or filtering locally.
		// Since we have token claims with roles, we could use that, but token might be stale.
		// Use DB query: GetClubsForUser(userID)
		// Assuming we don't have that query yet, we can fetch all and filter in memory if list is small,
		// or better: add a query.
		//
		// Given I cannot edit .sql files easily to add new queries without re-generating sqlc code (which requires sqlc tool),
		// I will maintain the logic in Go using the roles found in the DB.
		//
		// Strategy:
		// 1. Fetch user roles from DB to get fresh ClubIDs.
		// 2. Fetch specific clubs.

		// However, s.Queries.ListClubs returns ALL clubs.
		// Let's rely on the token claims for Club IDs as a first pass, or fetch roles fresh.
		dbRoles, errRoles := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(claims.UserID))
		if errRoles != nil {
			http.Error(w, "Failed to fetch user permissions", http.StatusInternalServerError)
			return
		}

		userClubIDs := make(map[string]bool)
		for _, r := range dbRoles {
			cid := uuid.UUID(r.ClubID.Bytes).String()
			userClubIDs[cid] = true
		}

		allClubs, errList := s.Queries.ListClubs(r.Context())
		if errList != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		for _, c := range allClubs {
			cid := uuid.UUID(c.ID.Bytes).String()
			if userClubIDs[cid] {
				clubs = append(clubs, c)
			}
		}
		err = nil // clear error
	}

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Should return empty array instead of null if empty
	if clubs == nil {
		clubs = []database.Club{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(clubs)
}

func (s *Server) handleGetClub(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	clubID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid club ID", http.StatusBadRequest)
		return
	}

	// Security check:
	// SysAdmin can access any club.
	// Users can only access clubs they are a member of (have a role in).
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !claims.IsSysAdmin {
		// Check membership
		// We can check the DB for 'has any role in this club'
		// We reuse the cached roles logic or query DB.
		// Query DB is safer.
		// "HasPermission" checks for specific permission. We just want *membership*.
		// If the user has *any* role in this club, they can see the club details.
		userRoles, err := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(claims.UserID))
		if err != nil {
			http.Error(w, "Failed to verify access", http.StatusInternalServerError)
			return
		}

		hasAccess := false
		for _, ur := range userRoles {
			if uuid.UUID(ur.ClubID.Bytes) == clubID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			http.Error(w, "Forbidden: You are not a member of this club", http.StatusForbidden)
			return
		}
	}

	club, err := s.Queries.GetClubByID(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Club not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(club)
}

func (s *Server) handleUpdateClub(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	clubID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid club ID", http.StatusBadRequest)
		return
	}

	var req CreateClubRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidClubType(req.Type) {
		http.Error(w, "Invalid club type provided. Allowed values are: sport_club, music_club, social_club, environment_club, cultural_club, hobby_club, rescue_service", http.StatusBadRequest)
		return
	}

	// Update params validation similar to Create is needed or reuse logic?
	// For update, typically we might want to check if date exists.
	// Since UpdateClubParams also updated by sqlc, we need to adapt it.

	var decisionDate pgtype.Date
	if req.TaxOfficeDecisionDate != nil && *req.TaxOfficeDecisionDate != "" {
		parsed, err := parseDate(*req.TaxOfficeDecisionDate)
		if err != nil {
			http.Error(w, "Invalid date format for tax_office_decision_date", http.StatusBadRequest)
			return
		}
		decisionDate = dateToPgDate(parsed)
	} else {
		decisionDate = pgtype.Date{Valid: false}
	}

	arg := database.UpdateClubParams{
		ID:                        uuidToPgtype(clubID),
		RegisteredAssociation:     req.RegisteredAssociation,
		Name:                      req.Name,
		Type:                      req.Type,
		Category:                  stringToPgText(req.Category),
		Number:                    stringToPgText(req.Number),
		StreetHouseNumber:         stringToPgText(req.StreetHouseNumber),
		PostalCode:                stringToPgText(req.PostalCode),
		City:                      stringToPgText(req.City),
		NameExtension:             stringToPgText(req.NameExtension),
		AddressExtension:          stringToPgText(req.AddressExtension),
		TaxOfficeName:             stringToPgText(req.TaxOfficeName),
		TaxOfficeTaxNumber:        stringToPgText(req.TaxOfficeTaxNumber),
		TaxOfficeAssessmentPeriod: stringToPgText(req.TaxOfficeAssessmentPeriod),
		TaxOfficePurpose:          stringToPgText(req.TaxOfficePurpose),
		TaxOfficeDecisionDate:     decisionDate,
		TaxOfficeDecisionType:     stringToPgText(req.TaxOfficeDecisionType),
	}

	club, err := s.Queries.UpdateClub(r.Context(), arg)
	if err != nil {
		if database.IsCheckViolationError(err) {
			http.Error(w, "Invalid club type provided (database constraint).", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to update club: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(club)
}

func (s *Server) handleDeleteClub(w http.ResponseWriter, r *http.Request) {
	// Security check: Only System Administrators can delete clubs
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok || !claims.IsSysAdmin {
		http.Error(w, "Forbidden: Only System Administrators can delete clubs", http.StatusForbidden)
		return
	}

	idStr := r.PathValue("id")
	clubID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid club ID", http.StatusBadRequest)
		return
	}

	err = s.Queries.DeleteClub(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to delete club: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
