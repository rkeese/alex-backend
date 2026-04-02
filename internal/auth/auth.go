package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey []byte

func SetJWTSecret(secret string) {
	if secret == "" {
		panic("JWT secret must not be empty — set it in config.json")
	}
	jwtKey = []byte(secret)
}

type RoleClaim struct {
	ClubID string `json:"club_id"`
	Role   string `json:"role"`
}

type Claims struct {
	UserID             uuid.UUID   `json:"sub"` // Changed from user_id to sub to match standard and generator
	Email              string      `json:"email"`
	IsSysAdmin         bool        `json:"is_sys_admin"`
	MustChangePassword bool        `json:"must_change_password"`
	Roles              []RoleClaim `json:"roles"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GenerateToken(userID uuid.UUID, email string, isSysAdmin bool, mustChangePassword bool, roles []RoleClaim) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:             userID,
		Email:              email,
		IsSysAdmin:         isSysAdmin,
		MustChangePassword: mustChangePassword,
		Roles:              roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil {
		fmt.Printf("Token validation error: %v\n", err)
		return nil, err
	}

	if !token.Valid {
		fmt.Println("Token is invalid")
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
