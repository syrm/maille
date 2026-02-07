package ofx

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"

	"github.com/samber/oops"
	"github.com/syrm/maille/internal"
)

type Parser struct{}

func (p Parser) parseBlock(ctx context.Context, block string) (internal.Transaction, error) {
	var trn internal.Transaction
	trn.Account = p.extractTag(block, "DTUSER")
	trn.DatePosted = p.extractTag(block, "DTPOSTED")
	trn.Name = p.extractTag(block, "NAME")
	trn.ID = p.extractTag(block, "FITID")
	trn.TrnType = p.extractTag(block, "TRNTYPE")
	amountText := p.extractTag(block, "TRNAMT")
	amount, err := strconv.ParseFloat(amountText, 64)
	if err != nil {
		return trn, p.oops(ctx).
			With("amount", amountText).
			Wrapf(err, "failed to parse amount")
	}

	trn.TrnAmount = amount

	return trn, nil
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

func (p *Parser) oops(ctx context.Context) oops.OopsErrorBuilder {
	return oops.In("OFXParser").WithContext(ctx)
}

func (p *Parser) getCurrency(reader io.Reader) string {
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

func (p *Parser) Parse(
	ctx context.Context,
	reader io.Reader,
	batchSize int,
	process func(context.Context, string, []internal.Transaction) error,
) error {
	currency := p.getCurrency(reader)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	scanner.Split(p.splitOnStmtTrn)

	stmts := make([]internal.Transaction, 0, batchSize)

	for scanner.Scan() {
		stmt, errParse := p.parseBlock(ctx, scanner.Text())

		if errParse != nil {
			return p.oops(ctx).Wrapf(errParse, "failed to process")
		}

		stmts = append(stmts, stmt)
		if len(stmts) >= batchSize {

			err := process(ctx, currency, stmts)
			if err != nil {
				return p.oops(ctx).Wrapf(err, "failed to process")
			}
			stmts = stmts[:0]
		}
	}

	if len(stmts) > 0 {
		err := process(ctx, currency, stmts)
		if err != nil {
			return p.oops(ctx).Wrapf(err, "failed to process")
		}
	}
	return scanner.Err()
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
