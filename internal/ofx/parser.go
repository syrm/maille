package ofx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"strings"
	"time"

	pkgcurrency "github.com/bojanz/currency"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"
)

type TransactionParsedType string

const (
	TransactionParsedDebit  TransactionParsedType = "DEBIT"
	TransactionParsedCredit TransactionParsedType = "CREDIT"
)

var (
	ErrNoTransactions = errors.New("OFX file contains no transactions")

	STMTopen     = []byte("<STMTTRN>")
	STMTopenLen  = len(STMTopen)
	STMTclose    = []byte("</STMTTRN>")
	STMTcloseLen = len(STMTclose)
)

type TransactionParsed struct {
	ID            string
	Type          TransactionParsedType
	Payee         string
	Narration     string
	Date          time.Time
	Amount        pkgcurrency.Amount
	BankAccountID string
}

type Parser struct {
	Tracer trace.Tracer
}

func (p Parser) parseBlock(ctx context.Context, block string, currency string, bankAccountID string) (TransactionParsed, error) {
	var tx TransactionParsed

	amountRaw := p.extractTag(block, "TRNAMT")
	amount, errAmount := pkgcurrency.NewAmount(amountRaw, currency)

	if errAmount != nil {
		return tx, p.oops(ctx).
			With("amount", amountRaw).
			With("currency", currency).
			Wrapf(errAmount, "failed to parse amount")
	}
	tx.Amount = amount

	tx.BankAccountID = bankAccountID

	dateRaw := p.extractTag(block, "DTPOSTED")
	if len(dateRaw) < len("20060102") {
		return tx, p.oops(ctx).
			With("date", dateRaw).
			Errorf("failed to parse date")
	}
	date, errDate := time.Parse("20060102", dateRaw[:8])

	if errDate != nil {
		return tx, p.oops(ctx).
			With("date", dateRaw).
			Wrapf(errDate, "failed to parse date")
	}
	tx.Date = date

	tx.Payee = p.extractTag(block, "NAME")
	tx.Narration = p.extractTag(block, "MEMO")
	if tx.Payee == "" {
		tx.Payee = tx.Narration
	}
	if tx.Payee == "" {
		return tx, p.oops(ctx).Errorf("transaction payee is missing")
	}

	tx.ID = p.extractTag(block, "FITID")
	if tx.ID == "" {
		return tx, p.oops(ctx).Errorf("transaction identifier is missing")
	}

	typeRaw := p.extractTag(block, "TRNTYPE")

	switch typeRaw {
	case "DEBIT", "CHECK", "PAYMENT", "CASH", "DIRECTDEBIT", "ATM", "POS", "FEE":
		tx.Type = TransactionParsedDebit
	case "CREDIT", "DIRECTDEP", "DEP", "INTEREST":
		tx.Type = TransactionParsedCredit
	default:
		return tx, p.oops(ctx).
			With("type", typeRaw).
			Errorf("failed to parse type")
	}

	return tx, nil
}

func (p Parser) extractTag(block, tag string) string {
	idx := strings.Index(block, "<"+tag+">")
	if idx == -1 {
		return ""
	}
	start := idx + len(tag) + 2
	rest := block[start:]
	end := strings.Index(rest, "</"+tag+">")
	if end == -1 {
		end = strings.IndexByte(rest, '<')
		if end == -1 {
			return strings.TrimSpace(rest)
		}
	}
	return strings.TrimSpace(rest[:end])
}

func (p Parser) oops(ctx context.Context) oops.OopsErrorBuilder {
	return oops.In("OFXParser").WithContext(ctx)
}

func (p Parser) Parse(orgCtx context.Context, reader io.Reader) iter.Seq2[TransactionParsed, error] {
	return func(yield func(TransactionParsed, error) bool) {
		ctx, span := p.Tracer.Start(orgCtx, "Parse")
		defer span.End()

		data, errRead := io.ReadAll(reader)
		if errRead != nil {
			yield(TransactionParsed{}, p.oops(ctx).Wrapf(errRead, "failed to read OFX file"))
			return
		}

		document := string(data)
		currency := p.extractTag(document, "CURDEF")
		if currency == "" {
			yield(TransactionParsed{}, p.oops(ctx).Errorf("OFX currency is missing"))
			return
		}

		bankAccountID := p.extractTag(document, "ACCTID")
		if bankAccountID == "" {
			bankAccountID = p.extractTag(document, "DTUSER")
		}
		if bankAccountID == "" {
			yield(TransactionParsed{}, p.oops(ctx).Errorf("OFX bank account identifier is missing"))
			return
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
		scanner.Split(p.splitOnStmtTrn)

		count := 0
		for scanner.Scan() {
			transaction, errParse := p.parseBlock(ctx, scanner.Text(), currency, bankAccountID)

			if errParse != nil {
				yield(TransactionParsed{}, p.oops(ctx).Wrapf(errParse, "failed to process"))
				return
			}

			if !yield(transaction, nil) {
				return
			}
			count++
		}

		if err := scanner.Err(); err != nil {
			yield(TransactionParsed{}, p.oops(ctx).Wrapf(err, "scanner error"))
			return
		}
		if count == 0 {
			yield(TransactionParsed{}, ErrNoTransactions)
		}
	}
}

func (p Parser) splitOnStmtTrn(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := bytes.Index(data, STMTopen)
	if start == -1 {
		if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	}

	end := bytes.Index(data[start:], STMTclose)
	if end == -1 {
		if atEOF {
			return 0, nil, io.ErrUnexpectedEOF
		}
		return 0, nil, nil
	}

	return start + end + STMTcloseLen, data[start+STMTopenLen : start+end], nil
}
