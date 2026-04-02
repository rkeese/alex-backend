package sepa

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSEPA(t *testing.T) {
	config := Config{
		MessageID:      "MSG-001",
		InitiatorName:  "Test Club",
		CollectionDate: time.Now().Add(24 * time.Hour),
	}

	transactions := []Transaction{
		{
			EndToEndID:     "TX-001",
			Amount:         10.00,
			DebtorName:     "John Doe",
			DebtorIBAN:     "DE0987654321",
			DebtorBIC:      "TESTBIC2",
			MandateID:      "MANDATE-001",
			MandateDate:    time.Now().Add(-24 * time.Hour),
			RemittanceInfo: "Membership Fee",
		},
		{
			// UUID length is 36, should be sanitized to 32 by removing hyphens
			EndToEndID:     "b75da5d5-a5f0-47fb-9e33-cd26ebcfea26",
			Amount:         20.00,
			DebtorName:     "Jane Doe",
			DebtorIBAN:     "DE0987654322",
			DebtorBIC:      "TESTBIC3",
			MandateID:      "MANDATE-002",
			MandateDate:    time.Now().Add(-24 * time.Hour),
			RemittanceInfo: "Donation",
		},
	}

	group := PaymentGroup{
		CreditorName: "Test Club",
		CreditorIBAN: "DE1234567890",
		CreditorBIC:  "TESTBIC",
		CreditorID:   "DE98ZZZ09999999999",
		SequenceType: "first", // Should be mapped to FRST
		Transactions: transactions,
	}

	xmlBytes, err := GenerateSEPA(config, []PaymentGroup{group})
	assert.NoError(t, err)
	assert.NotEmpty(t, xmlBytes)

	var doc Document
	err = xml.Unmarshal(xmlBytes, &doc)
	assert.NoError(t, err)

	// Validate Group Header
	assert.Equal(t, config.MessageID, doc.CstmrDrctDbtInitn.GrpHdr.MsgId)
	assert.Equal(t, 2, doc.CstmrDrctDbtInitn.GrpHdr.NbOfTxs)
	assert.Equal(t, 30.00, doc.CstmrDrctDbtInitn.GrpHdr.CtrlSum)

	// Validate Payment Info
	assert.Equal(t, 1, len(doc.CstmrDrctDbtInitn.PmtInf))
	pmtInf := doc.CstmrDrctDbtInitn.PmtInf[0]
	assert.Equal(t, "FRST", pmtInf.PmtTpInf.SeqTp) // Check mapping

	// Validate Transaction Details
	assert.NotEmpty(t, pmtInf.DrctDbtTxInf)
	assert.Equal(t, 2, len(pmtInf.DrctDbtTxInf))

	tx1 := pmtInf.DrctDbtTxInf[0]
	assert.NotNil(t, tx1.DrctDbtTx.CdtrSchmeId, "Creditor Scheme ID should be present at transaction level")

	tx2 := pmtInf.DrctDbtTxInf[1]
	// Verify UUID sanitization (hyphens removed)
	assert.Equal(t, "b75da5d5a5f047fb9e33cd26ebcfea26", tx2.PmtId.EndToEndId)
}
