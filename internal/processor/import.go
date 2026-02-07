package processor

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
)

var mappingAccount = map[string]string{
	"123456789": "FR:BNP:Checking",
}

type Transaction struct {
	ID         string
	DatePosted string
	TrnType    string
	TrnAmount  float64
	Name       string
	Account    string
}

type Currency struct {
	ID   uint
	Name string
}

type OFXParser struct{}

func (o OFXParser) parseBlock(ctx context.Context, block string) (Transaction, error) {
	var trn Transaction
	trn.Account = o.extractTag(block, "DTUSER")
	trn.DatePosted = o.extractTag(block, "DTPOSTED")
	trn.Name = o.extractTag(block, "NAME")
	trn.ID = o.extractTag(block, "FITID")
	trn.TrnType = o.extractTag(block, "TRNTYPE")
	amountText := o.extractTag(block, "TRNAMT")
	amount, err := strconv.ParseFloat(amountText, 64)
	if err != nil {
		return trn, o.oops(ctx).
			With("amount", amountText).
			Wrapf(err, "failed to parse amount")
	}

	trn.TrnAmount = amount

	return trn, nil
}

func (o OFXParser) extractTag(block, tag string) string {
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

func (o *OFXParser) oops(ctx context.Context) oops.OopsErrorBuilder {
	return oops.In("OFXParser").WithContext(ctx)
}

func (o *OFXParser) getCurrency(reader io.Reader) string {
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

func (o *OFXParser) Parse(
	ctx context.Context,
	reader io.Reader,
	batchSize int,
	process func(context.Context, string, []Transaction) error,
) error {
	currency := o.getCurrency(reader)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	scanner.Split(o.splitOnStmtTrn)

	stmts := make([]Transaction, 0, batchSize)

	for scanner.Scan() {
		stmt, errParse := o.parseBlock(ctx, scanner.Text())

		if errParse != nil {
			return o.oops(ctx).Wrapf(errParse, "failed to process")
		}

		stmts = append(stmts, stmt)
		if len(stmts) >= batchSize {

			err := process(ctx, currency, stmts)
			if err != nil {
				return o.oops(ctx).Wrapf(err, "failed to process")
			}
			stmts = stmts[:0]
		}
	}

	if len(stmts) > 0 {
		err := process(ctx, currency, stmts)
		if err != nil {
			return o.oops(ctx).Wrapf(err, "failed to process")
		}
	}

	println("count ID ", countID)
	println("insert1 ", insert1)
	println("insert2 ", insert2)
	return scanner.Err()
}

func (o OFXParser) splitOnStmtTrn(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

type Processor struct {
	Pool *pgxpool.Pool
}

var (
	countID = 0
	insert1 = 0
	insert2 = 0
)

func (p *Processor) Process(ctx context.Context, currency string, stmts []Transaction) error {
	rowCurrency := p.Pool.QueryRow(ctx, `SELECT id FROM currency WHERE name = $1`, currency)

	var currencyID uint32
	errScan := rowCurrency.Scan(&currencyID)

	if errScan != nil {
		return oops.
			In("Processor").
			WithContext(ctx).
			Wrapf(errScan, "failed to scan row")
	}

	bid := Bid{}
	_ = bid
	rows := make([][]any, 0, len(stmts))
	rowsPos := make([][]any, 0, len(stmts)*3)
	for _, stmt := range stmts {
		if _, ok := mappingAccount[stmt.Account]; !ok {
			return oops.
				In("Processor").
				WithContext(ctx).
				With("account", stmt.Account).
				Errorf("account not found")
		}

		sID := time.Now()
		id := bid.New()
		id2 := bid.New()
		id3 := bid.New()
		countID += int(time.Since(sID).Microseconds())

		var sb strings.Builder
		sb.Grow(10)
		sb.WriteString(stmt.DatePosted[0:4])
		sb.WriteByte('-')
		sb.WriteString(stmt.DatePosted[4:6])
		sb.WriteByte('-')
		sb.WriteString(stmt.DatePosted[6:8])
		rows = append(rows, []any{id, sb.String(), true, stmt.Name, stmt.ID})
		rowsPos = append(rowsPos, []any{id2, id, "ASSETS", mappingAccount[stmt.Account], stmt.TrnAmount, currencyID})
		if stmt.TrnType == "CREDIT" {
			rowsPos = append(rowsPos, []any{id3, id, "INCOME", "Others", -1 * stmt.TrnAmount, currencyID})
		} else {
			rowsPos = append(rowsPos, []any{id3, id, "EXPENSES", "Others", -1 * stmt.TrnAmount, currencyID})
		}
	}

	sCopy1 := time.Now()
	_, err := p.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"transaction"},
		[]string{"id", "date", "completed", "payee", "external_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return oops.
			In("Processor").
			WithContext(ctx).
			Wrapf(err, "failed to copy data")
	}
	insert1 += int(time.Since(sCopy1).Microseconds())

	sCopy2 := time.Now()
	_, err2 := p.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"posting"},
		[]string{"id", "transaction_id", "type", "account", "amount", "currency_id"},
		pgx.CopyFromRows(rowsPos),
	)
	if err2 != nil {
		return oops.
			In("Processor").
			WithContext(ctx).
			Wrapf(err2, "failed to copy data")
	}
	insert2 += int(time.Since(sCopy2).Microseconds())

	return nil
}
