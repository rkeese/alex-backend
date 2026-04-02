package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

type CreateBoardMemberRequest struct {
	MemberID uuid.UUID `json:"member_id"`
	Task     string    `json:"task"`
	Roles    []string  `json:"roles"`
}

type UpdateBoardMemberRequest struct {
	Task  string   `json:"task"`
	Roles []string `json:"roles"`
}

type BoardMemberResponse struct {
	ID           uuid.UUID `json:"id"`
	ClubID       uuid.UUID `json:"club_id"`
	MemberID     uuid.UUID `json:"member_id"`
	UserID       uuid.UUID `json:"user_id"`
	Position     string    `json:"position"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	MemberNumber string    `json:"member_number"`
	Email        string    `json:"email"`
}

func pgtypeToUuid(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	return uuid.UUID(id.Bytes)
}

// Helper to ensure only Admins can access
func (s *Server) requireClubAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		clubIDVal := r.Context().Value(clubIDKey)
		if clubIDVal == nil {
			http.Error(w, "Club context missing", http.StatusInternalServerError)
			return
		}
		clubID := clubIDVal.(uuid.UUID)

		isAdmin := false
		if claims.IsSysAdmin {
			isAdmin = true
		} else {
			for _, role := range claims.Roles {
				if role.ClubID == clubID.String() && role.Role == "admin" {
					isAdmin = true
					break
				}
			}
		}

		if !isAdmin {
			http.Error(w, "Forbidden: Only admins can manage the board", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

func (s *Server) HandleGetBoardMembers(w http.ResponseWriter, r *http.Request) {
	clubIDVal := r.Context().Value(clubIDKey)
	if clubIDVal == nil {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}
	clubID := clubIDVal.(uuid.UUID)

	members, err := s.Queries.GetBoardMembers(r.Context(), uuidToPgtype(clubID))
	if err != nil {
		http.Error(w, "Failed to fetch board members", http.StatusInternalServerError)
		return
	}

	resp := make([]BoardMemberResponse, len(members))
	for i, m := range members {
		resp[i] = BoardMemberResponse{
			ID:           pgtypeToUuid(m.ID),
			ClubID:       pgtypeToUuid(m.ClubID),
			MemberID:     pgtypeToUuid(m.MemberID),
			UserID:       pgtypeToUuid(m.UserID),
			Position:     m.Position,
			FirstName:    m.FirstName,
			LastName:     m.LastName,
			MemberNumber: m.MemberNumber,
			Email:        m.Email,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) HandleCreateBoardMember(w http.ResponseWriter, r *http.Request) {
	clubIDVal := r.Context().Value(clubIDKey)
	if clubIDVal == nil {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}
	clubID := clubIDVal.(uuid.UUID)

	var req CreateBoardMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Get Member to verify existence and get email
	member, err := s.Queries.GetMemberByID(r.Context(), database.GetMemberByIDParams{
		ID:     uuidToPgtype(req.MemberID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	if !member.Email.Valid || member.Email.String == "" {
		http.Error(w, "Member has no email address, cannot create user", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(member.Email.String))

	// 2. Check or Create User
	var userID uuid.UUID
	user, err := s.Queries.GetUserByEmail(r.Context(), email)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Create User
			hashedPwd, _ := auth.HashPassword("Start123!") // Standard password
			newUser, err := s.Queries.CreateUser(r.Context(), database.CreateUserParams{
				Email:              email,
				PasswordHash:       hashedPwd,
				MustChangePassword: true,
				IsSysAdmin:         false,
				IsBlocked:          false,
			})
			if err != nil {
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
			userID = pgtypeToUuid(newUser.ID)
		} else {
			http.Error(w, "Database error checking user", http.StatusInternalServerError)
			return
		}
	} else {
		userID = pgtypeToUuid(user.ID) // User exists
	}

	// 3. Create Board Member
	bm, err := s.Queries.CreateBoardMember(r.Context(), database.CreateBoardMemberParams{
		ClubID:   uuidToPgtype(clubID),
		MemberID: uuidToPgtype(req.MemberID),
		UserID:   uuidToPgtype(userID),
		Position: req.Task,
	})
	if err != nil {
		// likely unique constraint violation
		http.Error(w, "Failed to add board member (already exists?)", http.StatusConflict)
		return
	}

	// 4. Assign Roles
	for _, roleName := range req.Roles {
		role, err := s.Queries.GetRoleByName(r.Context(), roleName)
		if err != nil {
			continue // Skip invalid roles or handle error
		}
		// Add role
		err = s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
			UserID: uuidToPgtype(userID),
			RoleID: role.ID,
			ClubID: uuidToPgtype(clubID),
		})
		if err != nil {
			// Ignore if already exists (PK constraint)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(BoardMemberResponse{
		ID:           pgtypeToUuid(bm.ID),
		ClubID:       pgtypeToUuid(bm.ClubID),
		MemberID:     pgtypeToUuid(bm.MemberID),
		UserID:       pgtypeToUuid(bm.UserID),
		Position:     bm.Position,
		FirstName:    member.FirstName,
		LastName:     member.LastName,
		MemberNumber: member.MemberNumber,
		Email:        email,
	})
}

func (s *Server) HandleUpdateBoardMember(w http.ResponseWriter, r *http.Request) {
	clubIDVal := r.Context().Value(clubIDKey)
	if clubIDVal == nil {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}
	clubID := clubIDVal.(uuid.UUID)

	bmIDStr := r.PathValue("id")
	bmID, err := uuid.Parse(bmIDStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req UpdateBoardMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify BM exists and belongs to club
	bm, err := s.Queries.GetBoardMember(r.Context(), uuidToPgtype(bmID))
	if err != nil {
		http.Error(w, "Board member not found", http.StatusNotFound)
		return
	}
	if pgtypeToUuid(bm.ClubID) != clubID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Update Position
	_, err = s.Queries.UpdateBoardMember(r.Context(), database.UpdateBoardMemberParams{
		ID:       uuidToPgtype(bmID),
		Position: req.Task,
	})
	if err != nil {
		http.Error(w, "Failed to update", http.StatusInternalServerError)
		return
	}

	// Update Roles
	userID := pgtypeToUuid(bm.UserID)

	// 1. Get all available roles to map name -> ID
	allRoles, err := s.Queries.ListRoles(r.Context())
	if err != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	roleMap := make(map[string]uuid.UUID)
	for _, r := range allRoles {
		roleMap[r.Name] = pgtypeToUuid(r.ID)
	}

	// 2. Remove roles logic
	currentRolesRows, err := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(userID))
	if err == nil {
		for _, row := range currentRolesRows {
			if pgtypeToUuid(row.ClubID) == clubID {
				// Find ID for this role name
				if rID, ok := roleMap[row.RoleName]; ok {
					s.Queries.RemoveUserRole(r.Context(), database.RemoveUserRoleParams{
						UserID: uuidToPgtype(userID),
						RoleID: uuidToPgtype(rID),
						ClubID: uuidToPgtype(clubID),
					})
				}
			}
		}
	}

	// 3. Add new roles
	for _, roleName := range req.Roles {
		if rID, ok := roleMap[roleName]; ok {
			s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
				UserID: uuidToPgtype(userID),
				RoleID: uuidToPgtype(rID),
				ClubID: uuidToPgtype(clubID),
			})
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleDeleteBoardMember(w http.ResponseWriter, r *http.Request) {
	clubIDVal := r.Context().Value(clubIDKey)
	if clubIDVal == nil {
		http.Error(w, "Club ID not found in context", http.StatusInternalServerError)
		return
	}
	clubID := clubIDVal.(uuid.UUID)

	bmIDStr := r.PathValue("id")
	bmID, err := uuid.Parse(bmIDStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// Verify BM exists
	bm, err := s.Queries.GetBoardMember(r.Context(), uuidToPgtype(bmID))
	if err != nil {
		http.Error(w, "Board member not found", http.StatusNotFound)
		return
	}
	if pgtypeToUuid(bm.ClubID) != clubID {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Delete
	err = s.Queries.DeleteBoardMember(r.Context(), uuidToPgtype(bmID))
	if err != nil {
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
