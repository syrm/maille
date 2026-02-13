package ofx

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"
)

type TransactionParsedType string

const (
	TransactionParsedDebit  TransactionParsedType = "DEBIT"
	TransactionParsedCredit TransactionParsedType = "CREDIT"
)

type TransactionParsed struct {
	ID            string
	Type          TransactionParsedType
	Payee         string
	Date          time.Time
	Amount        float64
	Currency      string
	BankAccountID string
}

type Parser struct {
	Tracer trace.Tracer
}

func (p Parser) parseBlock(ctx context.Context, block string) (TransactionParsed, error) {
	var tx TransactionParsed

	amountRaw := p.extractTag(block, "TRNAMT")
	amount, errAmount := strconv.ParseFloat(amountRaw, 64)

	if errAmount != nil {
		return tx, p.oops(ctx).
			With("amount", amountRaw).
			Wrapf(errAmount, "failed to parse amount")
	}
	tx.Amount = amount

	tx.BankAccountID = p.extractTag(block, "DTUSER")

	if tx.BankAccountID == "" {
		spew.Dump(block, tx)
	}

	dateRaw := p.extractTag(block, "DTPOSTED")
	date, errDate := time.Parse("20060102", dateRaw)

	if errDate != nil {
		return tx, p.oops(ctx).
			With("date", dateRaw).
			Wrapf(errDate, "failed to parse date")
	}
	tx.Date = date

	tx.Payee = p.extractTag(block, "NAME")

	tx.ID = p.extractTag(block, "FITID")

	typeRaw := p.extractTag(block, "TRNTYPE")

	switch typeRaw {
	case "DEBIT":
		tx.Type = TransactionParsedDebit
	case "CREDIT":
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
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:end])
}

func (p Parser) oops(ctx context.Context) oops.OopsErrorBuilder {
	return oops.In("OFXParser").WithContext(ctx)
}

func (p Parser) getCurrency(reader io.Reader) string {
	bufreader := bufio.NewReader(reader)

	currency := "NOP"

	for {
		data, err := bufreader.ReadBytes('<')
		if err != nil {
			return currency
		}

		if len(data) < 8 {
			continue
		}

		if string(data[0:7]) == "CURDEF>" {
			index := strings.IndexByte(string(data[7:]), '<')

			if index == -1 {
				return currency
			}

			return string(data[7 : 7+index])
		}
	}
}

func (p Parser) Parse(orgCtx context.Context, reader io.Reader) iter.Seq2[TransactionParsed, error] {
	return func(yield func(TransactionParsed, error) bool) {
		ctx, span := p.Tracer.Start(orgCtx, "Parse")
		defer span.End()

		currency := p.getCurrency(reader)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

		scanner.Split(p.splitOnStmtTrn)

		for scanner.Scan() {
			transaction, errParse := p.parseBlock(ctx, scanner.Text())
			transaction.Currency = currency

			if errParse != nil {
				yield(TransactionParsed{}, p.oops(ctx).Wrapf(errParse, "failed to process"))
				return
			}

			if !yield(transaction, nil) {
				return
			}
		}

		if err := scanner.Err(); err != nil {
			yield(TransactionParsed{}, p.oops(ctx).Wrapf(err, "scanner error"))
		}
	}
}

func (p Parser) splitOnStmtTrn(data []byte, atEOF bool) (advance int, token []byte, err error) {
	open := []byte("<STMTTRN>")
	close := []byte("</STMTTRN>")

	start := bytes.Index(data, open)
	if start == -1 {
		if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	}

	end := bytes.Index(data[start:], close)
	if end == -1 {
		if atEOF {
			return len(data), nil, nil
		}
		return 0, nil, nil
	}

	blockStart := start + len(open)
	blockEnd := start + end
	advance = start + end + len(close)
	return advance, data[blockStart:blockEnd], nil
}
