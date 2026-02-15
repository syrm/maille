package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"github.com/syrm/maille/internal/domain"
	"go.opentelemetry.io/otel/trace"
)

type Transaction struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildTransaction(ctx context.Context, pool *pgxpool.Pool, tracer trace.Tracer) *Transaction {
	return &Transaction{
		pool:   pool,
		tracer: tracer,
	}
}

func (s *Transaction) GetAll(ctx context.Context, after uint64, size uint) ([]domain.Transaction, error) {
	ctx, span := s.tracer.Start(ctx, "GetTransactions")
	defer span.End()

	rows, errQuery := s.pool.Query(
		context.WithValue(ctx, SQLName, "get transactions"),
		`SELECT transaction.id, date, payee, narration, account_id, amount
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
		WHERE transaction.id > @after
		ORDER BY transaction.id
		LIMIT @size
		`,
		pgx.NamedArgs{
			"after": after,
			"size":  size,
		},
	)

	if errQuery != nil {
		return []domain.Transaction{},
			oops.
				In("Transaction").
				WithContext(ctx).
				Wrapf(errQuery, "failed to get transactions")
	}
	defer rows.Close()

	txs := make([]domain.Transaction, 0, size)
	tx := domain.Transaction{}
	postings := []domain.Posting{}
	lastID := 0
	for rows.Next() {
		var id int
		var date time.Time
		var payee string
		var narration *string
		var accountID uint64
		var amount float64

		rows.Scan(&id, &date, &payee, &narration, &accountID, &amount)

		if lastID != 0 && lastID != id {
			tx.Postings = postings
			txs = append(txs, tx)

			tx = domain.Transaction{}
			postings = []domain.Posting{}
		}

		tx.ID = uint64(id)
		tx.Date = date
		tx.Payee = payee
		tx.Narration = narration
		posting := domain.Posting{
			AccountID: accountID,
			Amount:    amount,
		}
		postings = append(postings, posting)
		lastID = id
	}

	if err := rows.Err(); err != nil {
		return []domain.Transaction{},
			oops.
				In("Transaction").
				WithContext(ctx).
				Wrapf(err, "failed to row iterate")
	}

	return txs, nil
}

func (s *Transaction) Save(ctx context.Context, transactions []domain.Transaction) error {
	ctx, span := s.tracer.Start(ctx, "Transaction")
	defer span.End()

	node, _ := snowflake.NewNode(1)

	rows := make([][]any, 0, len(transactions))
	rowsPos := make([][]any, 0, len(transactions)*3)
	for _, transaction := range transactions {
		transactionID := node.Generate()

		// Build date.
		date := transaction.Date.Format(time.DateOnly)
		var sb strings.Builder
		sb.Grow(10)
		sb.WriteString(date[0:4])
		sb.WriteByte('-')
		sb.WriteString(date[5:7])
		sb.WriteByte('-')
		sb.WriteString(date[8:10])

		rows = append(rows, []any{transactionID, sb.String(), true, transaction.Payee, transaction.ExternalID})

		for _, posting := range transaction.Postings {
			rowsPos = append(
				rowsPos,
				[]any{
					node.Generate(),
					transactionID,
					posting.AccountID,
					posting.Amount,
					posting.Currency.ID,
				},
			)
		}
	}

	tx, errBegin := s.pool.Begin(context.WithValue(ctx, SQLName, "begin transaction"))

	if errBegin != nil {
		return oops.
			In("Transaction").
			WithContext(ctx).
			Wrapf(errBegin, "failed to begin transaction")
	}

	defer tx.Rollback(context.WithValue(ctx, SQLName, "rollback transaction"))

	// _, errDrop1 := tx.Exec(
	// 	context.WithValue(ctx, SQLName, "drop constraint currency_id"),
	// 	"ALTER TABLE posting DROP CONSTRAINT posting_currency_id_fkey",
	// )
	// if errDrop1 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errDrop1, "failed to exec drop constraint currency_id")
	// }

	// _, errDrop2 := tx.Exec(
	// 	context.WithValue(ctx, SQLName, "drop constraint origin_position_id"),
	// 	"ALTER TABLE posting DROP CONSTRAINT posting_origin_position_id_fkey",
	// )
	// if errDrop2 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errDrop2, "failed to exec drop constraint currency_id")
	// }

	// _, errDrop3 := tx.Exec(
	// 	context.WithValue(ctx, SQLName, "drop constraint price_currency_id"),
	// 	"ALTER TABLE posting DROP CONSTRAINT posting_price_currency_id_fkey",
	// )
	// if errDrop3 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errDrop3, "failed to exec drop constraint price_currency_id")
	// }

	// _, errDrop4 := tx.Exec(
	// 	context.WithValue(ctx, SQLName, "drop constraint transaction_id"),
	// 	"ALTER TABLE posting DROP CONSTRAINT posting_transaction_id_fkey",
	// )
	// if errDrop4 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errDrop4, "failed to drop constraint")
	// }

	_, errCopyTr := tx.CopyFrom(
		context.WithValue(ctx, SQLName, "copy transaction"),
		pgx.Identifier{"transaction"},
		[]string{"id", "date", "completed", "payee", "external_id"},
		pgx.CopyFromRows(rows),
	)
	if errCopyTr != nil {
		return oops.
			In("Transaction").
			WithContext(ctx).
			Wrapf(errCopyTr, "failed to exec copy transaction")
	}
	_, errCopyPosting := tx.CopyFrom(
		context.WithValue(ctx, SQLName, "copy posting"),
		pgx.Identifier{"posting"},
		[]string{"id", "transaction_id", "account_id", "amount", "currency_id"},
		pgx.CopyFromRows(rowsPos),
	)
	if errCopyPosting != nil {
		return oops.
			In("Transaction").
			WithContext(ctx).
			Wrapf(errCopyPosting, "failed to exec copy posting")
	}

	// _, errAdd1 := tx.Exec(
	// 	context.WithValue(ctx, SQLName, "add constraint currency_id"),
	// 	`ALTER TABLE posting ADD CONSTRAINT posting_currency_id_fkey FOREIGN KEY (currency_id) REFERENCES currency(id)`,
	// )
	// if errAdd1 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errAdd1, "failed to add constraint currency_id")
	// }
	// _, errAdd2 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint origin_position_id"),
	// 	"ALTER TABLE posting ADD CONSTRAINT posting_origin_position_id_fkey FOREIGN KEY (origin_position_id) REFERENCES position(id)",
	// )
	// if errAdd2 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errAdd2, "failed to add constraint origin_position_id")
	// }
	// _, errAdd3 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint price_currency_id"), "ALTER TABLE posting ADD CONSTRAINT posting_price_currency_id_fkey FOREIGN KEY (price_currency_id) REFERENCES currency(id)")
	// if errAdd3 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errAdd3, "failed to add constraint currency_id")
	// }
	// _, errAdd4 := tx.Exec(context.WithValue(ctx, SQLName, "add constraint transaction_id"), "ALTER TABLE posting ADD CONSTRAINT posting_transaction_id_fkey FOREIGN KEY (transaction_id) REFERENCES transaction(id)")
	// if errAdd4 != nil {
	// 	return oops.
	// 		In("Transaction").
	// 		WithContext(ctx).
	// 		Wrapf(errAdd4, "failed to add constraint transaction_id")
	// }

	errCmt := tx.Commit(context.WithValue(ctx, SQLName, "commit transaction"))
	if errCmt != nil {
		return oops.
			In("Transaction").
			WithContext(ctx).
			Wrapf(errCmt, "failed to commit data")
	}

	return nil
}
