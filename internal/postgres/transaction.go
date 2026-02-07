package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal"
)

var mappingAccount = map[string]string{
	"123456789": "FR:BNP:Checking",
}

type Store struct {
	Pool *pgxpool.Pool
}

var (
	countID    = 0
	insert1    = 0
	insert2    = 0
	totalQuery = 0
)

func (s *Store) Process(ctx context.Context, currency string, stmts []internal.Transaction) error {
	sTQ := time.Now()
	rowCurrency := s.Pool.QueryRow(ctx, `SELECT id FROM currency WHERE name = $1`, currency)
	// entropy := rand.New(rand.NewSource(time.Now().UnixNano()))
	// ms := ulid.Timestamp(time.Now())
	//
	node, _ := snowflake.NewNode(1)

	var currencyID uint32
	errScan := rowCurrency.Scan(&currencyID)

	if errScan != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errScan, "failed to scan row")
	}

	rows := make([][]any, 0, len(stmts))
	rowsPos := make([][]any, 0, len(stmts)*3)
	for _, stmt := range stmts {
		if _, ok := mappingAccount[stmt.Account]; !ok {
			return oops.
				In("Store").
				WithContext(ctx).
				With("account", stmt.Account).
				Errorf("account not found")
		}

		sID := time.Now()
		id := node.Generate()
		id2 := node.Generate()
		id3 := node.Generate()

		// id, _ := ulid.New(ms, entropy)
		// id2, _ := ulid.New(ms, entropy)
		// id3, _ := ulid.New(ms, entropy)
		// id := bid.New()
		// id2 := bid.New()
		// id3 := bid.New()
		countID += int(time.Since(sID).Milliseconds())

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

	tx, _ := s.Pool.Begin(ctx)
	defer tx.Rollback(ctx)
	tx.Exec(ctx, "ALTER TABLE posting DROP CONSTRAINT posting_currency_id_fkey")
	tx.Exec(ctx, "ALTER TABLE posting DROP CONSTRAINT posting_origin_position_id_fkey")
	tx.Exec(ctx, "ALTER TABLE posting DROP CONSTRAINT posting_price_currency_id_fkey")
	_, err4 := tx.Exec(ctx, "ALTER TABLE posting DROP CONSTRAINT posting_transaction_id_fkey")
	if err4 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(err4, "failed to drop constraint")
	}

	sCopy1 := time.Now()
	// _, err := s.Pool.CopyFrom(
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"transaction"},
		[]string{"id", "date", "completed", "payee", "external_id"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(err, "failed to copy data")
	}
	insert1 += int(time.Since(sCopy1).Milliseconds())

	sCopy2 := time.Now()
	// _, err := s.Pool.CopyFrom(
	_, err2 := tx.CopyFrom(
		ctx,
		pgx.Identifier{"posting"},
		[]string{"id", "transaction_id", "type", "account", "amount", "currency_id"},
		pgx.CopyFromRows(rowsPos),
	)
	if err2 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(err2, "failed to copy data")
	}
	insert2 += int(time.Since(sCopy2).Milliseconds())

	fmt.Println("id timer ", countID, "ms")
	fmt.Println("copy 1 timer ", insert1, "ms")
	fmt.Println("copy 2 timer ", insert2, "ms")

	_, errA1 := tx.Exec(ctx, "ALTER TABLE posting ADD CONSTRAINT posting_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES currency(id)")
	if errA1 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errA1, "failed to add constraint 1")
	}
	_, errA2 := tx.Exec(ctx, "ALTER TABLE posting ADD CONSTRAINT posting_origin_position_id_fkey FOREIGN KEY (origin_position_id) REFERENCES position(id)")
	if errA2 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errA2, "failed to add constraint 2")
	}
	_, errA3 := tx.Exec(ctx, "ALTER TABLE posting ADD CONSTRAINT posting_price_currency_id_fkey FOREIGN KEY (price_currency_id) REFERENCES currency(id)")
	if errA3 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errA3, "failed to add constraint 3")
	}
	_, errA4 := tx.Exec(ctx, "ALTER TABLE posting ADD CONSTRAINT posting_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES transaction(id)")
	if errA4 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errA4, "failed to add constraint 4")
	}

	errCmt := tx.Commit(ctx)
	if errCmt != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errCmt, "failed to commit data")
	}
	totalQuery += int(time.Since(sTQ).Milliseconds())
	fmt.Println("total query timer ", totalQuery, "ms")

	return nil
}
