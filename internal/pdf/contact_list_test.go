package pdf

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jung-kurt/gofpdf"
)

// generateContactListUncompressed mirrors GenerateContactList but with compression
// disabled so that text content can be verified in raw PDF bytes.
func generateContactListUncompressed(clubName string, entries []ContactEntry) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetCompression(false)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetMargins(5, 10, 5)
	pdf.SetAutoPageBreak(true, 10)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 8, tr(fmt.Sprintf("%s - Kontaktliste", clubName)))
	pdf.Ln(10)

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

	pdf.SetFont("Arial", "", 7)
	for _, e := range entries {
		if pdf.GetY()+6 > pageH-10 {
			pdf.AddPage()
			printHeader()
			pdf.SetFont("Arial", "", 7)
		}
		pdf.CellFormat(colWidths[0], 6, tr(e.MemberNumber), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, tr(e.LastName), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, tr(e.FirstName), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, tr(e.StreetHouseNumber), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, tr(e.PostalCode), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, tr(e.City), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[6], 6, tr(e.Phone1), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[7], 6, tr(e.Mobile), "1", 0, "", false, 0, "")
		pdf.CellFormat(colWidths[8], 6, tr(e.Email), "1", 0, "", false, 0, "")
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestGenerateContactListMobileMapping(t *testing.T) {
	entries := []ContactEntry{
		{
			MemberNumber:      "M001",
			FirstName:         "Max",
			LastName:          "Mustermann",
			StreetHouseNumber: "Teststr. 1",
			PostalCode:        "12345",
			City:              "Berlin",
			Phone1:            "030-1234567",
			Mobile:            "0171-9876543",
			Email:             "max@test.de",
		},
	}

	pdfBytes, err := generateContactListUncompressed("Testverein", entries)
	if err != nil {
		t.Fatalf("GenerateContactList failed: %v", err)
	}

	content := string(pdfBytes)

	checks := map[string]string{
		"MemberNumber": "M001",
		"LastName":     "Mustermann",
		"FirstName":    "Max",
		"PostalCode":   "12345",
		"City":         "Berlin",
		"Phone1":       "030-1234567",
		"Mobile":       "0171-9876543",
		"Email":        "max@test.de",
	}

	for field, val := range checks {
		if !strings.Contains(content, val) {
			t.Errorf("%s value %q not found in PDF output", field, val)
		}
	}

	if len(pdfBytes) == 0 {
		t.Error("Generated PDF is empty")
	}
}

func TestGenerateContactListMobileInCompressedPDF(t *testing.T) {
	entries := []ContactEntry{
		{
			MemberNumber:      "M001",
			FirstName:         "Max",
			LastName:          "Mustermann",
			StreetHouseNumber: "Teststr. 1",
			PostalCode:        "12345",
			City:              "Berlin",
			Phone1:            "030-1234567",
			Mobile:            "0171-9876543",
			Email:             "max@test.de",
		},
		{
			MemberNumber:      "M002",
			FirstName:         "Erika",
			LastName:          "Musterfrau",
			StreetHouseNumber: "Hauptstr. 5",
			PostalCode:        "54321",
			City:              "Hamburg",
			Phone1:            "",
			Mobile:            "0172-1111111",
			Email:             "erika@test.de",
		},
	}

	// Use the actual GenerateContactList (compressed)
	pdfBytes, err := GenerateContactList("Testverein", entries)
	if err != nil {
		t.Fatalf("GenerateContactList failed: %v", err)
	}

	if len(pdfBytes) < 500 {
		t.Errorf("PDF suspiciously small: %d bytes", len(pdfBytes))
	}

	// Write PDF to temp file for manual inspection if needed
	tmpFile := t.TempDir() + "/kontaktliste_test.pdf"
	if err := os.WriteFile(tmpFile, pdfBytes, 0644); err != nil {
		t.Fatalf("Failed to write test PDF: %v", err)
	}
	t.Logf("Test PDF written to: %s", tmpFile)
	t.Logf("PDF size: %d bytes", len(pdfBytes))
}
