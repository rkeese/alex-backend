package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ReceiptData struct {
	ClubName      string
	ClubAddress   string
	DonorName     string
	DonorAddress  string
	Amount        float64
	Date          time.Time
	ReceiptNumber string
}

func GenerateDonationReceipt(data ReceiptData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Spendenquittung")
	pdf.Ln(12)

	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Verein: %s", data.ClubName))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Anschrift: %s", data.ClubAddress))
	pdf.Ln(12)

	pdf.Cell(0, 10, fmt.Sprintf("Spender: %s", data.DonorName))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Anschrift: %s", data.DonorAddress))
	pdf.Ln(12)

	pdf.Cell(0, 10, fmt.Sprintf("Betrag: %.2f EUR", data.Amount))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Datum der Zuwendung: %s", data.Date.Format("02.01.2006")))
	pdf.Ln(8)
	pdf.Cell(0, 10, fmt.Sprintf("Belegnummer: %s", data.ReceiptNumber))
	pdf.Ln(12)

	pdf.Cell(0, 10, "Es handelt sich um den Verzicht auf Erstattung von Aufwendungen.")
	pdf.Cell(0, 10, "Wir sind wegen Förderung gemeinnütziger Zwecke nach dem letzten uns zugegangenen Freistellungsbescheid des Finanzamtes ... von der Körperschaftsteuer befreit.")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
