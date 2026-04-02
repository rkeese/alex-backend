package database

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestCreateAndGetClub(t *testing.T) {
	if testQueries == nil {
		t.Skip("Skipping integration test: database not connected")
	}

	ctx := context.Background()

	// Create a club
	arg := CreateClubParams{
		RegisteredAssociation:     true,
		Name:                      "Integration Test Club",
		Type:                      "sport_club",
		Category:                  pgtype.Text{String: "Football", Valid: true},
		Number:                    "12345",
		StreetHouseNumber:         "Test St. 1",
		PostalCode:                "12345",
		City:                      "Test City",
		TaxOfficeName:             "Test Tax Office",
		TaxOfficeTaxNumber:        "123/456/789",
		TaxOfficeAssessmentPeriod: "2023",
		TaxOfficePurpose:          "Non-profit",
		TaxOfficeDecisionDate:     pgtype.Date{Time: time.Now(), Valid: true},
		TaxOfficeDecisionType:     "Exemption",
	}

	club, err := testQueries.CreateClub(ctx, arg)
	assert.NoError(t, err)
	assert.NotEmpty(t, club.ID)
	assert.Equal(t, arg.Name, club.Name)

	// Get the club
	fetchedClub, err := testQueries.GetClubByID(ctx, pgtype.UUID{Bytes: club.ID.Bytes, Valid: true})
	assert.NoError(t, err)
	assert.Equal(t, club.ID, fetchedClub.ID)
	assert.Equal(t, club.Name, fetchedClub.Name)

	// Cleanup (Optional: delete the club if you have a DeleteClub query)
	// err = testQueries.DeleteClub(ctx, club.ID)
	// assert.NoError(t, err)
}
