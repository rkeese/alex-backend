package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type BankBalanceRow struct {
	Name         string
	StartBalance float64
	Income       float64
	Expense      float64
	EndBalance   float64
}

type OverviewRow struct {
	Name    string
	Income  float64
	Expense float64
	Result  float64
}

type DetailItem struct {
	Date        string
	BookingText string
	Purpose     string
	Amount      float64
	AccountName string
}

type FinanceStatementData struct {
	ClubName         string
	Year             int
	GeneratedAt      time.Time
	BankBalances     []BankBalanceRow
	TotalBankBalance BankBalanceRow
	Overview         []OverviewRow
	TotalOverview    OverviewRow
	Details          map[string][]DetailItem // Key: Category Name (e.g., "Ideeller Bereich")
}

func GenerateFinanceStatement(data FinanceStatementData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// 1. Title
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 10, tr(data.ClubName))
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, tr(fmt.Sprintf("Finanz-Jahresabschluss für das Geschäftsjahr %d", data.Year)))
	pdf.Ln(15)

	// 2. Bank Balances
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 10, tr("1. Finanzbestände"))
	pdf.Ln(8)

	// Table Header
	headers := []string{"Konto", "Anfang (€)", "Einnahmen (€)", "Ausgaben (€)", "Ende (€)"}
	widths := []float64{60, 30, 30, 30, 30}

	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 8, tr(h), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)

	// Table Body
	pdf.SetFont("Arial", "", 10)
	for _, row := range data.BankBalances {
		pdf.CellFormat(widths[0], 7, tr(row.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(widths[1], 7, fmt.Sprintf("%.2f", row.StartBalance), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[2], 7, fmt.Sprintf("%.2f", row.Income), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[3], 7, fmt.Sprintf("%.2f", row.Expense), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[4], 7, fmt.Sprintf("%.2f", row.EndBalance), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	// Table Footer (Total)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(widths[0], 8, tr("Gesamt"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[1], 8, fmt.Sprintf("%.2f", data.TotalBankBalance.StartBalance), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[2], 8, fmt.Sprintf("%.2f", data.TotalBankBalance.Income), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[3], 8, fmt.Sprintf("%.2f", data.TotalBankBalance.Expense), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[4], 8, fmt.Sprintf("%.2f", data.TotalBankBalance.EndBalance), "1", 0, "R", true, 0, "")
	pdf.Ln(15)

	// 3. Overview
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 10, tr("2. Übersicht der Bereiche"))
	pdf.Ln(8)

	ovHeaders := []string{"Bereich", "Einnahmen (€)", "Ausgaben (€)", "Ergebnis (€)"}
	ovWidths := []float64{70, 40, 40, 30}

	pdf.SetFont("Arial", "B", 10)
	for i, h := range ovHeaders {
		pdf.CellFormat(ovWidths[i], 8, tr(h), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 10)
	for _, row := range data.Overview {
		pdf.CellFormat(ovWidths[0], 7, tr(row.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(ovWidths[1], 7, fmt.Sprintf("%.2f", row.Income), "1", 0, "R", false, 0, "")
		pdf.CellFormat(ovWidths[2], 7, fmt.Sprintf("%.2f", row.Expense), "1", 0, "R", false, 0, "")
		pdf.CellFormat(ovWidths[3], 7, fmt.Sprintf("%.2f", row.Result), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(ovWidths[0], 8, tr("Gesamt"), "1", 0, "L", true, 0, "")
	pdf.CellFormat(ovWidths[1], 8, fmt.Sprintf("%.2f", data.TotalOverview.Income), "1", 0, "R", true, 0, "")
	pdf.CellFormat(ovWidths[2], 8, fmt.Sprintf("%.2f", data.TotalOverview.Expense), "1", 0, "R", true, 0, "")
	pdf.CellFormat(ovWidths[3], 8, fmt.Sprintf("%.2f", data.TotalOverview.Result), "1", 0, "R", true, 0, "")
	pdf.Ln(10)

	// 4. Details Pages
	// Define the order of categories. Typically standard ones first, then others.
	categories := []string{"Ideeller Bereich", "Vermögens-Bereich", "Zweckbetrieb", "Wirtschaftlicher Geschäftsbetrieb", "Sammelposten"}

	// Create a map to track which have been printed to print remaining custom categories if any
	printed := make(map[string]bool)
	for _, c := range categories {
		printed[c] = true
	}

	// Helper for checking page break
	checkPageBreak := func(h float64) {
		if pdf.GetY()+h > 270 {
			pdf.AddPage()
		}
	}

	// Print predefined categories in order
	for _, catName := range categories {
		items, ok := data.Details[catName]
		if !ok || len(items) == 0 {
			continue
		}

		pdf.AddPage()
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, tr(fmt.Sprintf("Detail: %s", catName)))
		pdf.Ln(10)

		// Detail Header
		pdf.SetFont("Arial", "B", 9)
		// Date(20) | Text(45) | Purpose(45) | Account(40) | Amount(30) = 180
		detWidths := []float64{20, 45, 45, 40, 30}
		headers := []string{"Datum", "Buchungstext", "Verwendungszweck", "Konto", "Betrag (€)"}

		pdf.SetFillColor(240, 240, 240)
		for i, h := range headers {
			pdf.CellFormat(detWidths[i], 8, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for _, item := range items {
			checkPageBreak(6)
			// Simple truncation
			pdf.CellFormat(detWidths[0], 6, item.Date, "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[1], 6, truncate(pdf, tr(item.BookingText), detWidths[1]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[2], 6, truncate(pdf, tr(item.Purpose), detWidths[2]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[3], 6, truncate(pdf, tr(item.AccountName), detWidths[3]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[4], 6, fmt.Sprintf("%.2f", item.Amount), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// Print any other categories not in the standard list
	for catName, items := range data.Details {
		if printed[catName] {
			continue
		}
		if len(items) == 0 {
			continue
		}

		pdf.AddPage()
		pdf.SetFont("Arial", "B", 14)
		pdf.Cell(0, 10, tr(fmt.Sprintf("Detail: %s", catName)))
		pdf.Ln(10)

		// ... same block for unknown categories ...
		// Re-use logic:
		// Detail Header
		pdf.SetFont("Arial", "B", 9)
		detWidths := []float64{20, 45, 45, 40, 30}
		headers := []string{"Datum", "Buchungstext", "Verwendungszweck", "Konto", "Betrag (€)"}

		pdf.SetFillColor(240, 240, 240)
		for i, h := range headers {
			pdf.CellFormat(detWidths[i], 8, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Arial", "", 8)
		for _, item := range items {
			checkPageBreak(6)
			pdf.CellFormat(detWidths[0], 6, item.Date, "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[1], 6, truncate(pdf, tr(item.BookingText), detWidths[1]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[2], 6, truncate(pdf, tr(item.Purpose), detWidths[2]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[3], 6, truncate(pdf, tr(item.AccountName), detWidths[3]), "1", 0, "L", false, 0, "")
			pdf.CellFormat(detWidths[4], 6, fmt.Sprintf("%.2f", item.Amount), "1", 0, "R", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// 5. Signatures
	pdf.AddPage()
	pdf.SetY(50)
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 10, fmt.Sprintf("Erstellt am: %s", data.GeneratedAt.Format("02.01.2006")))
	pdf.Ln(20)

	pdf.Cell(80, 10, "__________________________")
	pdf.Cell(20, 10, "")
	pdf.Cell(80, 10, "__________________________")
	pdf.Ln(5)

	pdf.Cell(80, 10, tr("Vorstand Finanzen"))
	pdf.Cell(20, 10, "")
	pdf.Cell(80, 10, tr("Kassenprüfer"))
	pdf.Ln(10)

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func truncate(pdf *gofpdf.Fpdf, text string, width float64) string {
	if pdf.GetStringWidth(text) <= width {
		return text
	}
	// Simple char truncation loop
	runes := []rune(text)
	for i := len(runes); i > 0; i-- {
		sub := string(runes[:i]) + "..."
		if pdf.GetStringWidth(sub) <= width {
			return sub
		}
	}
	return "..."
}
