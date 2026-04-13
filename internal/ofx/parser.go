package ofx

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"iter"
	"strings"
	"time"

	pkgcurrency "github.com/bojanz/currency"
	"github.com/davecgh/go-spew/spew"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type TransactionParsedType string

const (
	TransactionParsedDebit  TransactionParsedType = "DEBIT"
	TransactionParsedCredit TransactionParsedType = "CREDIT"
)

var (
	STMTopen     = []byte("<STMTTRN>")
	STMTopenLen  = len(STMTopen)
	STMTclose    = []byte("</STMTTRN>")
	STMTcloseLen = len(STMTclose)
)

type TransactionParsed struct {
	ID            string
	Type          TransactionParsedType
	Payee         string
	Date          time.Time
	Amount        pkgcurrency.Amount
	BankAccountID string
}

type Parser struct {
	Tracer trace.Tracer
}

func (p Parser) parseBlock(ctx context.Context, block string, currency string) (TransactionParsed, error) {
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

func (p Parser) getCurrency(ctx context.Context, reader io.Reader) string {
	ctx, span := p.Tracer.Start(ctx, "getCurrency")
	defer span.End()

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
	var timeToSplit int64 = 0
	var countSplit int64 = 0
	return func(yield func(TransactionParsed, error) bool) {
		ctx, span := p.Tracer.Start(orgCtx, "Parse")
		defer func() {
			span.AddEvent("splitOnStmtTrn", trace.WithAttributes(
				attribute.Int64("call_count", countSplit),
				attribute.Float64("total_duration_avg_us", float64(timeToSplit)/float64(countSplit)),
				attribute.Int64("total_duration_us", timeToSplit),
			),
			)
			span.End()
		}()

		currency := p.getCurrency(ctx, reader)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

		scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
			start := time.Now()
			advance, token, err := p.splitOnStmtTrn(data, atEOF)
			timeToSplit += time.Since(start).Microseconds()
			countSplit++

			return advance, token, err
		})

		for scanner.Scan() {
			transaction, errParse := p.parseBlock(ctx, scanner.Text(), currency)

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
			return len(data), nil, nil
		}
		return 0, nil, nil
	}

	return start + end + STMTcloseLen, data[start+STMTopenLen : start+end], nil
}
