package api

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rkeese/alex-backend/internal/auth"
	"github.com/rkeese/alex-backend/internal/config"
	"github.com/rkeese/alex-backend/internal/database"
	"github.com/stretchr/testify/assert"
)

var testQueries *database.Queries
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// Load configuration
	cfg, err := config.LoadConfig("../../config/config.json")
	if err != nil {
		log.Printf("Failed to load config: %v. Skipping integration tests.", err)
		os.Exit(0)
	}

	// Set JWT Secret for testing
	auth.SetJWTSecret("test-secret")

	ctx := context.Background()
	testPool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("Unable to connect to database: %v. Skipping integration tests.", err)
		os.Exit(0)
	}
	defer testPool.Close()

	testQueries = database.New(testPool)

	os.Exit(m.Run())
}

func TestCreateClubIntegration(t *testing.T) {
	if testQueries == nil {
		t.Skip("Skipping integration test: database not connected")
	}

	server := NewServer(testQueries, testPool)

	// 1. Generate a token
	token, err := auth.GenerateToken(uuid.New(), "test@example.com", true, false, nil)
	assert.NoError(t, err)

	// 2. Prepare Request Body
	reqBody := CreateClubRequest{
		RegisteredAssociation:     true,
		Name:                      "API Integration Club",
		Type:                      "music_club",
		Number:                    "999",
		StreetHouseNumber:         "Music Lane 1",
		PostalCode:                "54321",
		City:                      "Music City",
		TaxOfficeName:             "Tax Office",
		TaxOfficeTaxNumber:        "999/888/777",
		TaxOfficeAssessmentPeriod: "2024",
		TaxOfficePurpose:          "Charity",
		TaxOfficeDecisionDate:     "2024-01-01",
		TaxOfficeDecisionType:     "Notice",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/clubs", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := server.Routes()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("Response Body: %s", rr.Body.String())
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Check response
	var responseClub database.Club
	err = json.Unmarshal(rr.Body.Bytes(), &responseClub)
	assert.NoError(t, err)
	assert.Equal(t, reqBody.Name, responseClub.Name)
	assert.NotEmpty(t, responseClub.ID)
}
