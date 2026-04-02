package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type BirthdayEntry struct {
	Date      time.Time
	FirstName string
	LastName  string
	NewAge    int
}

func GenerateBirthdayList(clubName string, year int, entries []BirthdayEntry) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, fmt.Sprintf("%s - Geburtstagsliste %d", clubName, year))
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 10, "Datum")
	pdf.Cell(80, 10, "Name")
	pdf.Cell(30, 10, "Alter")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	for _, entry := range entries {
		pdf.Cell(40, 10, entry.Date.Format("02.01."))
		pdf.Cell(80, 10, fmt.Sprintf("%s %s", entry.FirstName, entry.LastName))
		pdf.Cell(30, 10, fmt.Sprintf("%d", entry.NewAge))
		pdf.Ln(8)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
