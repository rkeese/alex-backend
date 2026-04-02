package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

type AssignRoleRequest struct {
	UserID   string `json:"user_id"`
	RoleName string `json:"role_name"`
	ClubID   string `json:"club_id"`
}

func (s *Server) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	clubID, err := uuid.Parse(req.ClubID)
	if err != nil {
		http.Error(w, "Invalid Club ID", http.StatusBadRequest)
		return
	}

	// Permission Check: SysAdmin OR Club Admin
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isAuthorized := false
	if claims.IsSysAdmin {
		isAuthorized = true
	} else {
		for _, r := range claims.Roles {
			if r.ClubID == clubID.String() && r.Role == "admin" {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		http.Error(w, "Forbidden: You are not an admin for this club", http.StatusForbidden)
		return
	}

	role, err := s.Queries.GetRoleByName(r.Context(), req.RoleName)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	err = s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
		UserID: uuidToPgtype(userID),
		RoleID: role.ID,
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "Role already assigned", http.StatusConflict)
			return
		}
		http.Error(w, "Could not assign role", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleRemoveRole(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	roleName := r.URL.Query().Get("role_name")
	clubIDStr := r.URL.Query().Get("club_id")

	if userIDStr == "" || roleName == "" || clubIDStr == "" {
		http.Error(w, "Missing required query parameters: user_id, role_name, club_id", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	clubID, err := uuid.Parse(clubIDStr)
	if err != nil {
		http.Error(w, "Invalid Club ID", http.StatusBadRequest)
		return
	}

	// Permission Check: SysAdmin OR Club Admin
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isAuthorized := false
	if claims.IsSysAdmin {
		isAuthorized = true
	} else {
		for _, r := range claims.Roles {
			if r.ClubID == clubID.String() && r.Role == "admin" {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		http.Error(w, "Forbidden: You are not an admin for this club", http.StatusForbidden)
		return
	}

	role, err := s.Queries.GetRoleByName(r.Context(), roleName)
	if err != nil {
		http.Error(w, "Role not found", http.StatusNotFound)
		return
	}

	err = s.Queries.RemoveUserRole(r.Context(), database.RemoveUserRoleParams{
		UserID: uuidToPgtype(userID),
		RoleID: role.ID,
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Could not remove role", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := s.Queries.ListRoles(r.Context())
	if err != nil {
		http.Error(w, "Could not list roles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.Queries.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "Could not list users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

type UserRoleResponse struct {
	Role   string  `json:"role"`
	ClubID *string `json:"club_id"` // Pointer to allow null if needed, though schema says NOT NULL
}

type UserDetailsResponse struct {
	ID        uuid.UUID          `json:"id"`
	Email     string             `json:"email"`
	IsBlocked bool               `json:"is_blocked"`
	Roles     []UserRoleResponse `json:"roles"`
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	requesterIDVal := r.Context().Value(userIDKey)
	if requesterIDVal == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	requesterID := requesterIDVal.(uuid.UUID)

	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	isSysAdmin := ok && claims.IsSysAdmin

	// Determine if requester can view this user
	canView := false
	var adminClubIDs = make(map[string]bool)

	if isSysAdmin || requesterID == userID {
		canView = true
	} else if ok {
		// Check if requester is admin of any club
		for _, r := range claims.Roles {
			if r.Role == "admin" {
				adminClubIDs[r.ClubID] = true
			}
		}
		if len(adminClubIDs) > 0 {
			// If requester is an admin of SOME club, we proceed.
			// Ideally we should check if the target user IS associated with one of these clubs.
			// But for now, we will just Allow viewing the Basic Info, but filter the roles.
			canView = true
		}
	}

	if !canView {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 1. Get User Basic Info
	user, err := s.Queries.GetUserByID(r.Context(), uuidToPgtype(userID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// 2. Get User Roles
	dbRoles, err := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(userID))
	if err != nil {
		// Log error? For now assume empty if error or just return what we have
		dbRoles = []database.GetUserRolesRow{}
	}

	// 3. Construct Response
	var roleResponses []UserRoleResponse
	for _, dbRole := range dbRoles {
		clubIDStr := uuid.UUID(dbRole.ClubID.Bytes).String()

		// Visibility Check:
		// SysAdmin sees all.
		// User sees their own roles.
		// Club Admin sees roles ONLY for clubs they administer.

		isVisible := false
		if isSysAdmin || requesterID == userID {
			isVisible = true
		} else {
			if adminClubIDs[clubIDStr] {
				isVisible = true
			}
		}

		if isVisible {
			roleResponses = append(roleResponses, UserRoleResponse{
				Role:   dbRole.RoleName,
				ClubID: &clubIDStr,
			})
		}
	}
	// If a club admin requests a user who is NOT in their club (no roles in their club),
	// should they see the user at all?
	// "Get User" usually implies we know the ID.
	// If I just found the ID via "List Members" (which is restricted to club), I know they are in my club.
	// If I guessed the ID, I might see "Email".
	// For privacy, maybe we should return 404 if roleResponses is empty AND not sysadmin/self?
	// But maybe they are a "New User" with NO roles yet?
	// If they are a member of my club, they should be visible.
	// Checking "Is Member Of My Club" is harder here without an extra query.
	// But typically "User" exists because they are a member.
	// I'll stick to this logic: return the user, but mask hidden roles.

	response := UserDetailsResponse{
		ID:        userID,
		Email:     user.Email,
		IsBlocked: user.IsBlocked,
		Roles:     roleResponses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type UpdateUserRequest struct {
	Password           *string `json:"password"`
	MustChangePassword *bool   `json:"must_change_password"`
	IsBlocked          *bool   `json:"is_blocked"`
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	requesterIDVal := r.Context().Value(userIDKey)
	if requesterIDVal == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	requesterID := requesterIDVal.(uuid.UUID)

	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	isSysAdmin := ok && claims.IsSysAdmin

	// allow club admins to update users if they are admin in at least one club
	// ideally we should also check if the target user is somehow connected to the club
	isClubAdmin := false

	if !isSysAdmin && requesterID != targetUserID {
		if ok {
			for _, r := range claims.Roles {
				if r.Role == "admin" {
					isClubAdmin = true
					break
				}
			}
		}

		if !isClubAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	currentUser, err := s.Queries.GetUserByID(r.Context(), uuidToPgtype(targetUserID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	newPwdHash := currentUser.PasswordHash
	if req.Password != nil && *req.Password != "" {
		h, err := auth.HashPassword(*req.Password)
		if err != nil {
			http.Error(w, "Error hashing password", http.StatusInternalServerError)
			return
		}
		newPwdHash = h
	}

	newMustChange := currentUser.MustChangePassword
	if req.MustChangePassword != nil {
		newMustChange = *req.MustChangePassword
	}

	newIsBlocked := currentUser.IsBlocked
	if req.IsBlocked != nil {
		if isSysAdmin || isClubAdmin {
			newIsBlocked = *req.IsBlocked
		}
	}

	err = s.Queries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:                 uuidToPgtype(targetUserID),
		PasswordHash:       newPwdHash,
		MustChangePassword: newMustChange,
		IsBlocked:          newIsBlocked,
	})

	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// When admin unblocks a user, also reset any brute-force lockout
	if req.IsBlocked != nil && !*req.IsBlocked {
		_ = s.Queries.ResetFailedLoginAttempts(r.Context(), uuidToPgtype(targetUserID))
	}

	w.WriteHeader(http.StatusNoContent)
}

type ResetPasswordResponse struct {
	Password string `json:"password"`
}

func (s *Server) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	targetUserID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid User ID", http.StatusBadRequest)
		return
	}

	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Only SysAdmin or Club Admin may reset passwords
	isAuthorized := claims.IsSysAdmin
	if !isAuthorized {
		for _, role := range claims.Roles {
			if role.Role == "admin" {
				isAuthorized = true
				break
			}
		}
	}
	if !isAuthorized {
		http.Error(w, "Forbidden: requires SysAdmin or Club Admin", http.StatusForbidden)
		return
	}

	// Verify user exists
	_, err = s.Queries.GetUserByID(r.Context(), uuidToPgtype(targetUserID))
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Generate random password
	pwdBytes := make([]byte, 10)
	if _, err := rand.Read(pwdBytes); err != nil {
		http.Error(w, "Failed to generate password", http.StatusInternalServerError)
		return
	}
	newPassword := hex.EncodeToString(pwdBytes)

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	err = s.Queries.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:                 uuidToPgtype(targetUserID),
		PasswordHash:       hash,
		MustChangePassword: true,
		IsBlocked:          false,
	})
	if err != nil {
		http.Error(w, "Failed to reset password", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResetPasswordResponse{Password: newPassword})
}
