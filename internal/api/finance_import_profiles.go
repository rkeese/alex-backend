package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type bookingRow struct {
	BookingDate     time.Time
	ValutaDate      time.Time
	ClientRecipient string
	BookingText     string
	Purpose         string
	Amount          float64
	Currency        string
	ClientIBAN      *string
	ClientBIC       *string
}

// ImportProfile defines how to identify and parse a specific CSV format
type ImportProfile interface {
	// Name returns the friendly name of the profile
	Name() string
	// Matches returns true if the header suggests this profile fits
	Matches(header []string) bool
	// ParseRow converts a CSV record into a bookingRow
	// Returns the row, the source account IBAN (if found in row), and error
	ParseRow(record []string) (bookingRow, string, error)
}

// Registry of available profiles
var importProfiles = []ImportProfile{
	&SparkasseProfile{},
	&VolksbankProfile{},
}

// DetectProfile finds the first matching profile for a given header
func DetectProfile(header []string) ImportProfile {
	for _, p := range importProfiles {
		if p.Matches(header) {
			return p
		}
	}
	return nil
}

// --- Shared Helpers ---

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.TrimSpace(item) == val {
			return true
		}
	}
	return false
}

func parseBankDate(s string) (time.Time, error) {
	// Try DD.MM.YY then DD.MM.YYYY
	t, err := time.Parse("02.01.06", s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse("02.01.2006", s)
	return t, err
}

func parseGermanAmount(s string) float64 {
	// -526,5 or 1.840,68 +
	// 1. Remove points (thousands)
	s = strings.ReplaceAll(s, ".", "")
	// 2. Replace comma with dot
	s = strings.ReplaceAll(s, ",", ".")
	// 3. Handle trailing + or -
	sign := 1.0
	if strings.HasSuffix(s, "-") {
		sign = -1.0
		s = strings.TrimSuffix(s, "-")
	} else if strings.HasSuffix(s, "+") {
		s = strings.TrimSuffix(s, "+")
	}
	s = strings.TrimSpace(s)

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return val * sign
}

// --- Sparkasse Implementation ---

type SparkasseProfile struct{}

func (p *SparkasseProfile) Name() string { return "Sparkasse / Standard" }

func (p *SparkasseProfile) Matches(header []string) bool {
	return contains(header, "Auftragskonto") && contains(header, "Buchungstag") && contains(header, "Valutadatum")
}

func (p *SparkasseProfile) ParseRow(record []string) (bookingRow, string, error) {
	// Index mapping based on:
	// Auftragskonto;Buchungstag;Valutadatum;Buchungstext;Verwendungszweck;Beguenstigter/Zahlungspflichtiger;Kontonummer/IBAN;BIC (SWIFT-Code);Betrag;Waehrung;Info;Kategorie
	// 0            1           2           3            4                5                                 6                  7                8      9        10   11
	if len(record) < 10 {
		return bookingRow{}, "", fmt.Errorf("row too short")
	}

	bDate, _ := parseBankDate(record[1])
	vDate, _ := parseBankDate(record[2])
	amount := parseGermanAmount(record[8])

	clientIBAN := record[6] // Kontonummer/IBAN
	clientBIC := record[7]  // BIC (SWIFT-Code)

	return bookingRow{
		BookingDate:     bDate,
		ValutaDate:      vDate,
		ClientRecipient: record[5],
		BookingText:     record[3],
		Purpose:         record[4],
		Amount:          amount,
		Currency:        record[9],
		ClientIBAN:      &clientIBAN,
		ClientBIC:       &clientBIC,
	}, record[0], nil
}

// --- Volksbank Implementation ---

type VolksbankProfile struct{}

func (p *VolksbankProfile) Name() string { return "Volksbank / Raiffeisen" }

func (p *VolksbankProfile) Matches(header []string) bool {
	return contains(header, "Bezeichnung Auftragskonto") && contains(header, "IBAN Auftragskonto")
}

func (p *VolksbankProfile) ParseRow(record []string) (bookingRow, string, error) {
	// Index based on:
	// Bez.;IBAN;BIC;Bank;Buchungstag;Valuta;Name;IBAN_Part;BIC_Part;Text;Zweck;Betrag;Waehr;...
	// 0    1    2   3    4           5      6    7         8        9    10    11     12
	if len(record) < 13 {
		return bookingRow{}, "", fmt.Errorf("row too short")
	}

	bDate, _ := parseBankDate(record[4])
	vDate, _ := parseBankDate(record[5])
	amount := parseGermanAmount(record[11])

	clientIBAN := record[7]
	clientBIC := record[8]

	return bookingRow{
		BookingDate:     bDate,
		ValutaDate:      vDate,
		ClientRecipient: record[6],
		BookingText:     record[9],
		Purpose:         record[10],
		Amount:          amount,
		Currency:        record[12],
		ClientIBAN:      &clientIBAN,
		ClientBIC:       &clientBIC,
	}, record[1], nil
}
