package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/database"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
	clubIDKey contextKey = "clubID"
	claimsKey contextKey = "claims"
)

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		ctx = context.WithValue(ctx, claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) ClubContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clubIDStr := r.PathValue("club_id")
		if clubIDStr == "" {
			// Fallback to query param or header if needed
			clubIDStr = r.URL.Query().Get("club_id")
		}
		if clubIDStr == "" {
			clubIDStr = r.Header.Get("X-Club-ID")
		}

		if clubIDStr == "" {
			http.Error(w, "Club ID required", http.StatusBadRequest)
			return
		}

		clubID, err := uuid.Parse(clubIDStr)
		if err != nil {
			http.Error(w, "Invalid Club ID format", http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), clubIDKey, clubID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) RequirePermission(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(userIDKey).(uuid.UUID)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get Club ID from header or query param
		clubIDStr := r.Header.Get("X-Club-ID")
		if clubIDStr == "" {
			clubIDStr = r.URL.Query().Get("club_id")
		}
		if clubIDStr == "" {
			http.Error(w, "X-Club-ID header or club_id query parameter required", http.StatusBadRequest)
			return
		}

		clubID, err := uuid.Parse(clubIDStr)
		if err != nil {
			http.Error(w, "Invalid Club ID", http.StatusBadRequest)
			return
		}

		// Check claims first (Bootstrapping / Token Auth)
		claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
		if ok {
			// Super Admin Bypass for "system administrator" (User Request)
			if claims.IsSysAdmin {
				ctx := context.WithValue(r.Context(), clubIDKey, clubID)
				next(w, r.WithContext(ctx))
				return
			}

			for _, role := range claims.Roles {
				if role.ClubID == clubID.String() && role.Role == "admin" {
					// Admin has all permissions
					ctx := context.WithValue(r.Context(), clubIDKey, clubID)
					next(w, r.WithContext(ctx))
					return
				}
			}
		}

		// Check permission
		hasPermission, err := s.Queries.HasPermission(r.Context(), database.HasPermissionParams{
			UserID: uuidToPgtype(userID),
			ClubID: uuidToPgtype(clubID),
			Name:   permission,
		})
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !hasPermission {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), clubIDKey, clubID)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) RequireSysAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
		if !ok || claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !claims.IsSysAdmin {
			http.Error(w, "Forbidden: System Administrator rights required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
