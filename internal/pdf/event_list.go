package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type EventEntry struct {
	Date        time.Time
	Time        string
	Description string
}

func GenerateEventList(clubName string, title string, entries []EventEntry) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, tr(fmt.Sprintf("%s - %s", clubName, title)))
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(30, 10, "Datum")
	pdf.Cell(20, 10, "Zeit")
	pdf.Cell(130, 10, "Beschreibung")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 12)
	for _, entry := range entries {
		pdf.Cell(30, 10, entry.Date.Format("02.01.2006"))
		pdf.Cell(20, 10, entry.Time)
		pdf.Cell(130, 10, tr(entry.Description))
		pdf.Ln(8)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
