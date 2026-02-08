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
	"github.com/syrm/maille/internal/ofx"
	"go.opentelemetry.io/otel/trace"
)

var mappingAccount = map[string]string{
	"123456789": "FR:BNP:Checking",
}

type Transaction struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildTransaction(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) (*Transaction, error) {
	return &Transaction{
		pool:   pool,
		tracer: tracer,
	}, nil
}

func (s *Transaction) GetTransactions(ctx context.Context) ([]internal.Transaction, error) {
	ctx, span := s.tracer.Start(ctx, "GetTransactions")
	defer span.End()

	rows, errQuery := s.pool.Query(
		context.WithValue(ctx, SQLName, "get transactions"),
		`SELECT transaction.id, date, payee, narration, account, amount
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
		ORDER BY transaction.id
		`,
	)

	if errQuery != nil {
		return []internal.Transaction{}, oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get transactions")
	}
	defer rows.Close()

	var transactions []internal.Transaction

	transaction := internal.Transaction{}
	postings := []internal.Posting{}
	lastID := 0
	for rows.Next() {
		var id int
		var date time.Time
		var payee string
		var narration *string
		var account string
		var amount float64

		rows.Scan(&id, &date, &payee, &narration, &account, &amount)

		if lastID != 0 && lastID != id {
			transaction.Postings = postings
			transactions = append(transactions, transaction)

			transaction = internal.Transaction{}
			postings = []internal.Posting{}
		}

		transaction.ID = uint64(id)
		transaction.Date = date
		transaction.Payee = payee
		transaction.Narration = narration
		posting := internal.Posting{
			Account: account,
			Amount:  amount,
		}
		postings = append(postings, posting)
		lastID = id
	}

	if err := rows.Err(); err != nil {
		return transactions, fmt.Errorf("rows iteration: %w", err)
	}

	return transactions, nil
}

func (s *Transaction) Process(ctx context.Context, currency string, stmts []ofx.Transaction) error {
	ctx, span := s.tracer.Start(ctx, "Process")
	defer span.End()

	rowCurrency := s.pool.QueryRow(context.WithValue(ctx, SQLName, "get currency"), `SELECT id FROM currency WHERE name = $1`, currency)
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

		transactionID := node.Generate()
		positionID1 := node.Generate()
		positionID2 := node.Generate()

		// Build date.
		var sb strings.Builder
		sb.Grow(10)
		sb.WriteString(stmt.DatePosted[0:4])
		sb.WriteByte('-')
		sb.WriteString(stmt.DatePosted[4:6])
		sb.WriteByte('-')
		sb.WriteString(stmt.DatePosted[6:8])

		rows = append(rows, []any{transactionID, sb.String(), true, stmt.Name, stmt.ID})
		rowsPos = append(
			rowsPos,
			[]any{
				positionID1,
				transactionID,
				"ASSETS",
				mappingAccount[stmt.Account],
				stmt.TrnAmount,
				currencyID,
			},
		)
		if stmt.TrnType == "CREDIT" {
			rowsPos = append(rowsPos, []any{positionID2, transactionID, "INCOME", "Others", -1 * stmt.TrnAmount, currencyID})
		} else {
			rowsPos = append(rowsPos, []any{positionID2, transactionID, "EXPENSES", "Others", -1 * stmt.TrnAmount, currencyID})
		}
	}

	tx, errBegin := s.pool.Begin(context.WithValue(ctx, SQLName, "begin transaction"))

	if errBegin != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errBegin, "failed to begin transaction")
	}

	defer tx.Rollback(context.WithValue(ctx, SQLName, "rollback transaction"))

	_, errDrop1 := tx.Exec(
		context.WithValue(ctx, SQLName, "drop constraint currency_id"),
		"ALTER TABLE posting DROP CONSTRAINT posting_currency_id_fkey",
	)
	if errDrop1 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errDrop1, "failed to exec drop constraint currency_id")
	}

	_, errDrop2 := tx.Exec(
		context.WithValue(ctx, SQLName, "drop constraint origin_position_id"),
		"ALTER TABLE posting DROP CONSTRAINT posting_origin_position_id_fkey",
	)
	if errDrop2 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errDrop2, "failed to exec drop constraint currency_id")
	}

	_, errDrop3 := tx.Exec(
		context.WithValue(ctx, SQLName, "drop constraint price_currency_id"),
		"ALTER TABLE posting DROP CONSTRAINT posting_price_currency_id_fkey",
	)
	if errDrop3 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errDrop3, "failed to exec drop constraint price_currency_id")
	}

	_, errDrop4 := tx.Exec(
		context.WithValue(ctx, SQLName, "drop constraint transaction_id"),
		"ALTER TABLE posting DROP CONSTRAINT posting_transaction_id_fkey",
	)
	if errDrop4 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errDrop4, "failed to drop constraint")
	}

	_, errCopyTr := tx.CopyFrom(
		context.WithValue(ctx, SQLName, "copy transaction"),
		pgx.Identifier{"transaction"},
		[]string{"id", "date", "completed", "payee", "external_id"},
		pgx.CopyFromRows(rows),
	)
	if errCopyTr != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errCopyTr, "failed to exec copy transaction")
	}
	_, errCopyPosting := tx.CopyFrom(
		context.WithValue(ctx, SQLName, "copy posting"),
		pgx.Identifier{"posting"},
		[]string{"id", "transaction_id", "type", "account", "amount", "currency_id"},
		pgx.CopyFromRows(rowsPos),
	)
	if errCopyPosting != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errCopyPosting, "failed to exec copy posting")
	}

	_, errAdd1 := tx.Exec(
		context.WithValue(ctx, SQLName, "add constraint currency_id"),
		`ALTER TABLE posting ADD CONSTRAINT posting_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES currency(id)`,
	)
	if errAdd1 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errAdd1, "failed to add constraint currency_id")
	}
	_, errAdd2 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint origin_position_id"),
		"ALTER TABLE posting ADD CONSTRAINT posting_origin_position_id_fkey FOREIGN KEY (origin_position_id) REFERENCES position(id)",
	)
	if errAdd2 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errAdd2, "failed to add constraint origin_position_id")
	}
	_, errAdd3 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint price_currency_id"), "ALTER TABLE posting ADD CONSTRAINT posting_price_currency_id_fkey FOREIGN KEY (price_currency_id) REFERENCES currency(id)")
	if errAdd3 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errAdd3, "failed to add constraint currency_id")
	}
	_, errAdd4 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint transaction_id"), "ALTER TABLE posting ADD CONSTRAINT posting_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES transaction(id)")
	if errAdd4 != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errAdd4, "failed to add constraint transaction_id")
	}

	errCmt := tx.Commit(context.WithValue(ctx, SQLName, "commit transaction"))
	if errCmt != nil {
		return oops.
			In("Store").
			WithContext(ctx).
			Wrapf(errCmt, "failed to commit data")
	}

	return nil
}
