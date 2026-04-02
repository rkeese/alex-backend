package pdf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateEventList(t *testing.T) {
	entries := []EventEntry{
		{
			Date:        time.Now(),
			Time:        "10:00",
			Description: "Bürgerbüro",
		},
	}

	pdfBytes, err := GenerateEventList("Test Club", "Event List", entries)
	assert.NoError(t, err)
	assert.NotEmpty(t, pdfBytes)
	assert.Equal(t, "%PDF", string(pdfBytes[:4])) // Check PDF header
}
