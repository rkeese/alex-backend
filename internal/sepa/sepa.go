package sepa

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// Document is the root of the SEPA XML
type Document struct {
	XMLName           xml.Name          `xml:"urn:iso:std:iso:20022:tech:xsd:pain.008.001.02 Document"`
	XmlnsXsi          string            `xml:"xmlns:xsi,attr"`
	XmlnsXsd          string            `xml:"xmlns:xsd,attr"`
	SchemaLocation    string            `xml:"xsi:schemaLocation,attr"`
	CstmrDrctDbtInitn CstmrDrctDbtInitn `xml:"CstmrDrctDbtInitn"`
}

type CstmrDrctDbtInitn struct {
	GrpHdr GroupHeader          `xml:"GrpHdr"`
	PmtInf []PaymentInformation `xml:"PmtInf"`
}

type GroupHeader struct {
	MsgId    string  `xml:"MsgId"`
	CreDtTm  string  `xml:"CreDtTm"`
	NbOfTxs  int     `xml:"NbOfTxs"`
	CtrlSum  Decimal `xml:"CtrlSum"`
	InitgPty PartyId `xml:"InitgPty"`
}

type PartyId struct {
	Nm string `xml:"Nm"`
}

type PaymentInformation struct {
	PmtInfId     string                   `xml:"PmtInfId"`
	PmtMtd       string                   `xml:"PmtMtd"` // DD
	NbOfTxs      int                      `xml:"NbOfTxs"`
	CtrlSum      Decimal                  `xml:"CtrlSum"`
	PmtTpInf     PaymentType              `xml:"PmtTpInf"`
	ReqdColltnDt string                   `xml:"ReqdColltnDt"`
	Cdtr         PartyId                  `xml:"Cdtr"`
	CdtrAcct     Account                  `xml:"CdtrAcct"`
	CdtrAgt      Agent                    `xml:"CdtrAgt"`
	DrctDbtTxInf []DirectDebitTransaction `xml:"DrctDbtTxInf"`
}

type CreditorSchemeId struct {
	Id CreditorSchemeIdDetails `xml:"Id"`
}

type CreditorSchemeIdDetails struct {
	PrvtId PrivateIdentification `xml:"PrvtId"`
}

type PrivateIdentification struct {
	Othr OtherIdentification `xml:"Othr"`
}

type OtherIdentification struct {
	Id      string     `xml:"Id"`
	SchmeNm SchemeName `xml:"SchmeNm"`
}

type SchemeName struct {
	Prtry string `xml:"Prtry"`
}

type PaymentType struct {
	SvcLvl    ServiceLevel    `xml:"SvcLvl"`
	LclInstrm LocalInstrument `xml:"LclInstrm"`
	SeqTp     string          `xml:"SeqTp"` // RCUR or FRST
}

type ServiceLevel struct {
	Cd string `xml:"Cd"` // SEPA
}

type LocalInstrument struct {
	Cd string `xml:"Cd"` // CORE
}

type Account struct {
	Id AccountId `xml:"Id"`
}

type AccountId struct {
	IBAN string `xml:"IBAN"`
}

type Agent struct {
	FinInstnId FinancialInstitution `xml:"FinInstnId"`
}

type FinancialInstitution struct {
	BIC  string              `xml:"BIC,omitempty"`
	Othr *GenericFinancialId `xml:"Othr,omitempty"`
}

type GenericFinancialId struct {
	Id string `xml:"Id"`
}

type DirectDebitTransaction struct {
	PmtId     PaymentIdentification         `xml:"PmtId"`
	InstdAmt  Amount                        `xml:"InstdAmt"`
	DrctDbtTx DirectDebitTransactionDetails `xml:"DrctDbtTx"`
	DbtrAgt   Agent                         `xml:"DbtrAgt"`
	Dbtr      PartyId                       `xml:"Dbtr"`
	DbtrAcct  Account                       `xml:"DbtrAcct"`
	RmtInf    RemittanceInformation         `xml:"RmtInf"`
}

type PaymentIdentification struct {
	EndToEndId string `xml:"EndToEndId"`
}

type Amount struct {
	Ccy string  `xml:"Ccy,attr"`
	Val float64 `xml:",chardata"`
}

// Decimal is a float64 that marshals with 2 decimal places
type Decimal float64

func (d Decimal) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	s := fmt.Sprintf("%.2f", float64(d))
	return e.EncodeElement(s, start)
}

func (a Amount) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// Manually create the element to control the formatting of the value
	type AmountAlias Amount
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "Ccy"}, Value: a.Ccy},
	}
	s := fmt.Sprintf("%.2f", a.Val)
	return e.EncodeElement(s, start)
}

type DirectDebitTransactionDetails struct {
	MndtRltdInf MandateRelatedInformation `xml:"MndtRltdInf"`
	CdtrSchmeId *CreditorSchemeId         `xml:"CdtrSchmeId,omitempty"`
}

type MandateRelatedInformation struct {
	MndtId    string `xml:"MndtId"`
	DtOfSgntr string `xml:"DtOfSgntr"`
}

type RemittanceInformation struct {
	Ustrd string `xml:"Ustrd"`
}

type Transaction struct {
	EndToEndID     string
	Amount         float64
	DebtorName     string
	DebtorIBAN     string
	DebtorBIC      string
	MandateID      string
	MandateDate    time.Time
	RemittanceInfo string
}

type PaymentGroup struct {
	CreditorName string
	CreditorIBAN string
	CreditorBIC  string
	CreditorID   string // Creditor Identifier (Gläubiger-ID)
	SequenceType string
	Transactions []Transaction
}

type Config struct {
	MessageID      string
	InitiatorName  string
	CollectionDate time.Time
}

func GenerateSEPA(config Config, groups []PaymentGroup) ([]byte, error) {
	totalAmount := 0.0
	totaltxs := 0

	// Sanitize MessageID to remove invalid characters that might cause bank upload issues
	safeMsgID := sanitizeMsgID(config.MessageID)

	var pmtInfs []PaymentInformation

	for i, group := range groups {
		groupAmount := 0.0
		groupTxs := make([]DirectDebitTransaction, len(group.Transactions))

		for j, tx := range group.Transactions {
			groupAmount += tx.Amount

			var bankID FinancialInstitution
			if tx.DebtorBIC != "" {
				bankID = FinancialInstitution{BIC: tx.DebtorBIC}
			} else {
				// Required by older SEPA formats if BIC is missing (IBAN only)
				bankID = FinancialInstitution{Othr: &GenericFinancialId{Id: "NOTPROVIDED"}}
			}

			groupTxs[j] = DirectDebitTransaction{
				PmtId:    PaymentIdentification{EndToEndId: sanitizeEndToEndID(tx.EndToEndID)},
				InstdAmt: Amount{Ccy: "EUR", Val: tx.Amount},
				DrctDbtTx: DirectDebitTransactionDetails{
					MndtRltdInf: MandateRelatedInformation{
						MndtId:    tx.MandateID,
						DtOfSgntr: tx.MandateDate.Format("2006-01-02"),
					},
					CdtrSchmeId: &CreditorSchemeId{
						Id: CreditorSchemeIdDetails{
							PrvtId: PrivateIdentification{
								Othr: OtherIdentification{
									Id: group.CreditorID,
									SchmeNm: SchemeName{
										Prtry: "SEPA",
									},
								},
							},
						},
					},
				},
				DbtrAgt:  Agent{FinInstnId: bankID},
				Dbtr:     PartyId{Nm: sanitizeString(tx.DebtorName)},
				DbtrAcct: Account{Id: AccountId{IBAN: tx.DebtorIBAN}},
				RmtInf:   RemittanceInformation{Ustrd: sanitizeString(tx.RemittanceInfo)},
			}
		}

		totalAmount += groupAmount
		totaltxs += len(group.Transactions)

		pmtInfs = append(pmtInfs, PaymentInformation{
			PmtInfId: fmt.Sprintf("%s-%d", safeMsgID, i+1),
			PmtMtd:   "DD",
			NbOfTxs:  len(group.Transactions),
			CtrlSum:  Decimal(groupAmount),
			PmtTpInf: PaymentType{
				SvcLvl:    ServiceLevel{Cd: "SEPA"},
				LclInstrm: LocalInstrument{Cd: "CORE"},
				SeqTp:     mapSequenceType(group.SequenceType),
			},
			ReqdColltnDt: config.CollectionDate.Format("2006-01-02"),
			Cdtr:         PartyId{Nm: sanitizeString(group.CreditorName)},
			CdtrAcct:     Account{Id: AccountId{IBAN: group.CreditorIBAN}},
			CdtrAgt:      Agent{FinInstnId: FinancialInstitution{BIC: group.CreditorBIC}},
			DrctDbtTxInf: groupTxs,
		})
	}

	doc := Document{
		XmlnsXsi:       "http://www.w3.org/2001/XMLSchema-instance",
		XmlnsXsd:       "http://www.w3.org/2001/XMLSchema",
		SchemaLocation: "urn:iso:std:iso:20022:tech:xsd:pain.008.001.02 pain.008.001.02.xsd",
		CstmrDrctDbtInitn: CstmrDrctDbtInitn{
			GrpHdr: GroupHeader{
				MsgId:    safeMsgID,
				CreDtTm:  time.Now().Format("2006-01-02T15:04:05"),
				NbOfTxs:  totaltxs,
				CtrlSum:  Decimal(totalAmount),
				InitgPty: PartyId{Nm: sanitizeString(config.InitiatorName)},
			},
			PmtInf: pmtInfs,
		},
	}

	bytes, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}

	header := []byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n")
	return append(header, bytes...), nil
}

func mapSequenceType(seqType string) string {
	switch seqType {
	case "first":
		return "FRST"
	case "recurring":
		return "RCUR"
	case "final":
		return "FNAL"
	case "one-off":
		return "OOFF"
	default:
		return seqType
	}
}

func sanitizeEndToEndID(id string) string {
	if len(id) > 35 {
		// Strip hyphens if present to try and shorten it (common for UUIDs)
		sanitized := strings.ReplaceAll(id, "-", "")
		if len(sanitized) > 35 {
			// Hard truncate if still too long
			return sanitized[:35]
		}
		return sanitized
	}
	return id
}

func sanitizeString(s string) string {
	// Transliterate umlauts
	r := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue",
		"Ä", "AE", "Ö", "OE", "Ü", "UE",
		"ß", "ss",
	)
	s = r.Replace(s)

	// Keep only allowed characters.
	// SEPA characters: a-z A-Z 0-9 / - ? : ( ) . , ' + Space
	valid := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '.' || r == ':' || r == '+' || r == '/' || r == ' ' ||
			r == '?' || r == '(' || r == ')' || r == ',' || r == '\'' {
			valid = append(valid, r)
		}
	}
	return string(valid)
}

func sanitizeMsgID(id string) string {
	// MsgID is a subset of SEPA charset, fewer special chars allowed typically
	s := sanitizeString(id)

	// Further restrict for MsgID if needed, but for now reuse sanitizeString logic
	// but maybe strip spaces for ID? existing logic allowed spaces.
	// Let's stick to the previous restricted set for MsgID to be safe as it often used in filenames

	valid := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '.' || r == ':' || r == '+' || r == '/' || r == ' ' {
			valid = append(valid, r)
		}
	}
	return string(valid)
}
