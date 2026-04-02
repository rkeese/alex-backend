package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	password := "secret"
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secret"
	hash, _ := HashPassword(password)

	assert.True(t, CheckPasswordHash(password, hash))
	assert.False(t, CheckPasswordHash("wrong", hash))
}

func TestGenerateAndValidateToken(t *testing.T) {
	userID := uuid.New()
	email := "test@example.com"

	tokenString, err := GenerateToken(userID, email)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	claims, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
}

func TestValidateInvalidToken(t *testing.T) {
	_, err := ValidateToken("invalid-token")
	assert.Error(t, err)
}

func TestExpiredToken(t *testing.T) {
	// Create a token that is already expired
	expirationTime := time.Now().Add(-1 * time.Hour)
	claims := &Claims{
		UserID: uuid.New(),
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(jwtKey)

	_, err := ValidateToken(tokenString)
	assert.Error(t, err)
}
