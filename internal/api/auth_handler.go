package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user, err := s.Queries.CreateUser(r.Context(), database.CreateUserParams{
		Email:              req.Email,
		PasswordHash:       hashedPassword,
		IsSysAdmin:         false,
		MustChangePassword: false,
		IsBlocked:          false,
	})
	if err != nil {
		if database.IsDuplicateKeyError(err) {
			http.Error(w, "User already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Could not create user", http.StatusInternalServerError)
		return
	}

	// Convert pgtype.UUID to uuid.UUID
	userID, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// ------------------------------------------------------------
	// AUTO-ASSIGN 'new_user' ROLE (Minimal Rights)
	// ------------------------------------------------------------
	// 1. Get the default club (first created)
	club, err := s.Queries.GetFirstClub(r.Context())
	if err == nil { // Only if a club exists
		// 2. Get 'new_user' role
		role, err := s.Queries.GetRoleByName(r.Context(), "new_user")
		if err == nil {
			// 3. Assign role
			err = s.Queries.AddUserRole(r.Context(), database.AddUserRoleParams{
				UserID: user.ID,
				RoleID: role.ID,
				ClubID: club.ID,
			})
			if err != nil {
				// Log error but don't fail registration
			}
		}
	}
	// ------------------------------------------------------------

	// Fetch roles
	var roleClaims []auth.RoleClaim
	dbRoles, err := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(userID))
	if err == nil {
		for _, dbRole := range dbRoles {
			clubIDStr := uuid.UUID(dbRole.ClubID.Bytes).String()
			roleClaims = append(roleClaims, auth.RoleClaim{
				ClubID: clubIDStr,
				Role:   dbRole.RoleName,
			})
		}
	}

	token, err := auth.GenerateToken(userID, user.Email, user.IsSysAdmin, false, roleClaims)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	ip := clientIP(r)

	user, err := s.Queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("SECURITY: Failed login attempt for unknown email=%q from IP=%s", req.Email, ip)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check if account is temporarily locked
	if user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now()) {
		remaining := time.Until(user.LockedUntil.Time)
		retryAfter := int(remaining.Seconds()) + 1
		w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
		log.Printf("SECURITY: Login blocked for locked account email=%q from IP=%s (locked for %d more seconds)", req.Email, ip, retryAfter)
		http.Error(w, fmt.Sprintf("Account temporarily locked due to too many failed login attempts. Try again in %d minutes.", int(remaining.Minutes())+1), http.StatusTooManyRequests)
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		// Increment failed attempts (may trigger lockout)
		_ = s.Queries.IncrementFailedLoginAttempts(r.Context(), user.ID)
		newAttempts := user.FailedLoginAttempts + 1
		log.Printf("SECURITY: Failed login attempt #%d for email=%q from IP=%s", newAttempts, req.Email, ip)
		if newAttempts >= 5 {
			log.Printf("SECURITY: Account email=%q locked for 15 minutes after %d failed attempts from IP=%s", req.Email, newAttempts, ip)
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if user.IsBlocked {
		log.Printf("SECURITY: Login attempt for admin-blocked account email=%q from IP=%s", req.Email, ip)
		http.Error(w, "Account blocked", http.StatusForbidden)
		return
	}

	// Successful login: reset failed attempts
	if user.FailedLoginAttempts > 0 {
		_ = s.Queries.ResetFailedLoginAttempts(r.Context(), user.ID)
	}

	// Convert pgtype.UUID to uuid.UUID
	userID, err := uuid.FromBytes(user.ID.Bytes[:])
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Fetch roles
	var roleClaims []auth.RoleClaim
	dbRoles, err := s.Queries.GetUserRoles(r.Context(), uuidToPgtype(userID))
	if err == nil {
		for _, dbRole := range dbRoles {
			clubIDStr := uuid.UUID(dbRole.ClubID.Bytes).String()
			roleClaims = append(roleClaims, auth.RoleClaim{
				ClubID: clubIDStr,
				Role:   dbRole.RoleName,
			})
		}
	}

	token, err := auth.GenerateToken(userID, user.Email, user.IsSysAdmin, user.MustChangePassword, roleClaims)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

// Helper to convert uuid.UUID to pgtype.UUID
func uuidToPgtype(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
