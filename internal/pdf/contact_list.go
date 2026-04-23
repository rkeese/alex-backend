package pdf

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
)

type ContactEntry struct {
	MemberNumber      string
	FirstName         string
	LastName          string
	StreetHouseNumber string
	PostalCode        string
	City              string
	Phone1            string
	Mobile            string
	Email             string
}

func GenerateContactList(clubName string, entries []ContactEntry) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Reduce margins to maximize usable width: 297 - 2*5 = 287mm
	pdf.SetMargins(5, 10, 5)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, tr(fmt.Sprintf("%s - Kontaktliste", clubName)))
	pdf.Ln(10)

	// Column widths tuned to 287mm total
	colWidths := []float64{18, 32, 32, 55, 15, 32, 30, 30, 43}
	headers := []string{"Mitgl.Nr.", "Nachname", "Vorname", "Straße & Hausnr.", "PLZ", "Ort", "Telefon", "Mobil", "E-Mail"}
	_, pageH, _ := pdf.PageSize(pdf.PageNo())

	printHeader := func() {
		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(220, 220, 220)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], 7, tr(h), "1", 0, "", true, 0, "")
		}
		pdf.Ln(-1)
	}

	printHeader()

	// Table rows
	pdf.SetFont("Arial", "", 7)
	for _, e := range entries {
		if pdf.GetY()+6 > pageH-10 {
			pdf.AddPage()
			printHeader()
			pdf.SetFont("Arial", "", 7)
		}

		pdf.CellFormat(colWidths[0], 6, e.MemberNumber, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, tr(e.LastName), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, tr(e.FirstName), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, tr(e.StreetHouseNumber), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, e.PostalCode, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, tr(e.City), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[6], 6, e.Phone1, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[7], 6, e.Mobile, "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[8], 6, e.Email, "1", 0, "", false, 0, "")
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
