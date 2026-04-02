package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Copied from finance_import_profiles.go
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.TrimSpace(item) == val {
			return true
		}
	}
	return false
}

type ImportProfile interface {
	Name() string
	Matches(header []string) bool
}

type SparkasseProfile struct{}

func (p *SparkasseProfile) Name() string { return "Sparkasse / Standard" }
func (p *SparkasseProfile) Matches(header []string) bool {
	return contains(header, "Auftragskonto") && contains(header, "Buchungstag") && contains(header, "Valutadatum")
}

type VolksbankProfile struct{}

func (p *VolksbankProfile) Name() string { return "Volksbank / Raiffeisen" }
func (p *VolksbankProfile) Matches(header []string) bool {
	return contains(header, "Bezeichnung Auftragskonto") && contains(header, "IBAN Auftragskonto")
}

var importProfiles = []ImportProfile{
	&SparkasseProfile{},
	&VolksbankProfile{},
}

func DetectProfile(header []string) ImportProfile {
	for _, p := range importProfiles {
		if p.Matches(header) {
			return p
		}
	}
	return nil
}

func main() {
	filePath := `c:\Users\Raimund.Keese\go\src\github.com\rkeese\data\Umsaetze_DE75520900000046817400_2025.12.31.csv`
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := charmap.Windows1252.NewDecoder()
	utf8Reader := transform.NewReader(file, decoder)

	reader := csv.NewReader(utf8Reader)
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Failed to read CSV: %v\n", err)
		return
	}

	if len(records) < 1 {
		fmt.Println("Empty CSV")
		return
	}

	header := records[0]
	fmt.Printf("Header (%d cols): %v\n", len(header), header)

	// Print hex of first column to check for BOM or garbage
	if len(header) > 0 {
		fmt.Printf("First col hex: %x\n", []byte(header[0]))
	}

	profile := DetectProfile(header)
	if profile == nil {
		fmt.Println("No profile detected")
		fmt.Printf("Checking Sparkasse: %v\n", (&SparkasseProfile{}).Matches(header))
		fmt.Printf("Checking Volksbank: %v\n", (&VolksbankProfile{}).Matches(header))

		// Debug contains
		fmt.Printf("Contains 'Auftragskonto': %v\n", contains(header, "Auftragskonto"))
		fmt.Printf("Contains 'Bezeichnung Auftragskonto': %v\n", contains(header, "Bezeichnung Auftragskonto"))
	} else {
		fmt.Printf("Detected Profile: %s\n", profile.Name())
	}
}
