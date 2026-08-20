package ofx

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
)

const validOFX = `OFXHEADER:100
DATA:OFXSGML

<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<CURDEF>EUR
<BANKACCTFROM><ACCTID>123456789</BANKACCTFROM>
<BANKTRANLIST>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20260819120000.000[+1:CET]
<TRNAMT>-42.50
<FITID>tx-debit-1
<NAME>AMAZON
</STMTTRN>
<STMTTRN>
<TRNTYPE>DIRECTDEP
<DTPOSTED>20260820
<TRNAMT>2850.00
<FITID>tx-credit-1
<MEMO>VIR SEPA SALAIRE ENTREPRISE A
</STMTTRN>
</BANKTRANLIST>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

func TestParserParse(t *testing.T) {
	p := Parser{Tracer: otel.Tracer("test")}

	var transactions []TransactionParsed
	for transaction, err := range p.Parse(context.Background(), strings.NewReader(validOFX)) {
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		transactions = append(transactions, transaction)
	}

	if len(transactions) != 2 {
		t.Fatalf("Parse() transaction count = %d, want 2", len(transactions))
	}
	if got := transactions[0]; got.ID != "tx-debit-1" || got.BankAccountID != "123456789" || got.Payee != "AMAZON" || got.Amount.Number() != "-42.50" {
		t.Errorf("first transaction = %#v", got)
	}
	if got := transactions[1]; got.Type != TransactionParsedCredit || got.Payee != "VIR SEPA SALAIRE ENTREPRISE A" || got.Narration != "VIR SEPA SALAIRE ENTREPRISE A" || got.Date.Day() != 20 {
		t.Errorf("second transaction = %#v", got)
	}
}

func TestParserRejectsFileWithoutTransactions(t *testing.T) {
	p := Parser{Tracer: otel.Tracer("test")}

	var got error
	for _, err := range p.Parse(context.Background(), strings.NewReader("<OFX><CURDEF>EUR<ACCTID>123456789</OFX>")) {
		got = err
	}

	if !errors.Is(got, ErrNoTransactions) {
		t.Fatalf("Parse() error = %v, want ErrNoTransactions", got)
	}
}
