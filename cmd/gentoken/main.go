package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func main() {
	userID := flag.String("user", "test", "User ID")
	role := flag.String("role", "admin", "Role (admin, finance, member)")
	clubID := flag.String("club", "", "Club ID (optional, defaults to random)")
	flag.Parse()

	finalClubID := *clubID
	if finalClubID == "" {
		finalClubID = uuid.New().String()
	}

	claims := jwt.MapClaims{
		"sub": *userID,
		"roles": []map[string]string{{
			"club_id": finalClubID,
			"role":    *role,
		}},
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte("dev_secret_key_change_in_production"))
	if err != nil {
		log.Fatal("Failed to sign token:", err)
	}

	fmt.Println(signedToken)
}
