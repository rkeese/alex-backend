package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

func (s *Server) handleInviteMember(w http.ResponseWriter, r *http.Request) {
	clubIDStr := r.PathValue("club_id")
	memberIDStr := r.PathValue("member_id")

	clubID, err := uuid.Parse(clubIDStr)
	if err != nil {
		http.Error(w, "Invalid Club ID", http.StatusBadRequest)
		return
	}

	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		http.Error(w, "Invalid Member ID", http.StatusBadRequest)
		return
	}

	// Permission check (Manual because path param differs from header expectation in RequirePermission)
	// Alternatively, rely on RequirePermission wrapping this and ensuring club_id is extracted correctly.
	// But standard RequirePermission looks for Header/Query. Here it is in Path.
	// We can trust the caller to check permission or check it here.
	// Let's rely on standard check inside this handler for safety.
	// Actually, API router uses s.RequirePermission which validates token but keys off Header/Query.
	// The router setup: mux.Handle("...", s.AuthMiddleware(http.HandlerFunc(s.handleInviteMember)))
	// We should verify the user has access to THIS club.
	claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	hasAccess := false
	if claims.IsSysAdmin {
		hasAccess = true
	} else {
		for _, role := range claims.Roles {
			if role.ClubID == clubID.String() && role.Role == "admin" {
				hasAccess = true
				break
			}
		}
	}
	if !hasAccess {
		http.Error(w, "Forbidden: Only admins can invite members", http.StatusForbidden)
		return
	}

	// Logic
	member, err := s.Queries.GetMemberByID(r.Context(), database.GetMemberByIDParams{
		ID:     uuidToPgtype(memberID),
		ClubID: uuidToPgtype(clubID),
	})
	if err != nil {
		http.Error(w, "Member not found", http.StatusNotFound)
		return
	}

	if member.UserID.Valid {
		http.Error(w, "Member already has a user account", http.StatusConflict)
		return
	}

	if !member.Email.Valid || member.Email.String == "" {
		http.Error(w, "Member has no email address", http.StatusBadRequest)
		return
	}

	// Check if user exists
	email := strings.ToLower(strings.TrimSpace(member.Email.String))
	var user database.User
	var isNewUser bool
	var generatedPassword string

	existingUser, err := s.Queries.GetUserByEmail(r.Context(), email)
	if err == nil {
		user = existingUser
		isNewUser = false
	} else {
		isNewUser = true
		// Create new user
		generatedPassword = generateRandomPassword()
		hashedPassword, err := auth.HashPassword(generatedPassword)
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		user, err = s.Queries.CreateUser(r.Context(), database.CreateUserParams{
			Email:              email,
			PasswordHash:       hashedPassword,
			IsSysAdmin:         false,
			MustChangePassword: true,
			IsBlocked:          false,
		})
		if err != nil {
			http.Error(w, "Could not create user", http.StatusInternalServerError)
			return
		}
	}

	// Link Member to User
	userID, _ := uuid.FromBytes(user.ID.Bytes[:])
	err = s.Queries.UpdateMemberUser(r.Context(), database.UpdateMemberUserParams{
		ID:     member.ID,
		UserID: uuidToPgtype(userID),
	})
	if err != nil {
		http.Error(w, "Failed to link user", http.StatusInternalServerError)
		return
	}

	// Assign 'new_user' role
	role, err := s.Queries.GetRoleByName(r.Context(), "new_user")
	if err == nil {
		s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
			UserID: uuidToPgtype(userID),
			RoleID: role.ID,
			ClubID: uuidToPgtype(clubID),
		})
	}

	// Send Email
	if isNewUser {
		subject := "Welcome to Club Management Framework"
		body := fmt.Sprintf("Hello,\n\nYou have been invited to join. Your temporary password is: %s\n\nPlease log in and change your password.", generatedPassword)
		if err := s.EmailSender.SendEmail([]string{email}, subject, body); err != nil {
			fmt.Printf("Failed to send welcome email to %s: %v\n", email, err)
		}
	} else {
		subject := "Club Access Granted"
		body := fmt.Sprintf("Hello,\n\nYou have been granted access to a new club context.")
		if err := s.EmailSender.SendEmail([]string{email}, subject, body); err != nil {
			fmt.Printf("Failed to send access email to %s: %v\n", email, err)
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"invited"}`))
}

func generateRandomPassword() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "fallbackPassword123"
	}
	return hex.EncodeToString(bytes)
}
