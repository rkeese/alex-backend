package pdf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDonationReceipt(t *testing.T) {
	data := ReceiptData{
		ClubName:      "Test Club",
		ClubAddress:   "Test Address 1",
		DonorName:     "John Doe",
		DonorAddress:  "Donor Address 1",
		Amount:        100.00,
		Date:          time.Now(),
		ReceiptNumber: "2023-001",
	}

	pdfBytes, err := GenerateDonationReceipt(data)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.Equal(t, "%PDF", string(pdfBytes[:4])) // Check PDF header
}
