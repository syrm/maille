package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bojanz/currency"
	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/domain/api"
	"github.com/syrm/maille/internal/postgres/dto"
)

type Transaction struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

func BuildTransaction(pool *pgxpool.Pool, tracer trace.Tracer) *Transaction {
	return &Transaction{
		pool:   pool,
		tracer: tracer,
	}
}

func (t *Transaction) GetAllWithPosting(ctx context.Context, after uint64, size uint) (map[uint64]*domain.Transaction, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetAllWithPosting")
	defer span.End()

	txs := make(map[uint64]*domain.Transaction, size)

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get transactions"),
		`SELECT transaction.id, date, payee, narration, posting.id, account_id, amount
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
		WHERE transaction.id > @after
		ORDER BY transaction.id, posting.id
		LIMIT @size
		`,
		pgx.NamedArgs{
			"after": after,
			"size":  size,
		},
	)

	if errQuery != nil {
		return txs, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get transactions")
	}

	var (
		id        uint64
		date      time.Time
		payee     string
		narration *string
		postingID uint64
		accountID uint64
		amount    float64
	)

	_, errScan := pgx.ForEachRow(
		rows,
		[]any{&id, &date, &payee, &narration, &postingID, &accountID, &amount},
		func() error {

			tx, exist := txs[id]

			if !exist {
				tx = &domain.Transaction{
					ID:        id,
					Date:      date,
					Payee:     payee,
					Narration: narration,
				}
				txs[id] = tx
			}

			tx.Postings = append(tx.Postings, domain.Posting{
				ID:        postingID,
				AccountID: accountID,
				Amount:    amount,
			})

			return nil
		},
	)

	if errScan != nil {
		return txs, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errScan, "failed to scan transaction")
	}

	if err := rows.Err(); err != nil {
		return txs, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(err, "failed to row iterate")
	}

	return txs, nil
}

func (t *Transaction) GetAllToClassify(ctx context.Context, after uint64, size uint) (map[uint64]domain.TransactionToClassify, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetAllToClassify")
	defer span.End()

	txs := make(map[uint64]domain.TransactionToClassify, size)

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get transactions to classify"),
		`SELECT transaction.id,
       date,
       payee,
       (ARRAY_AGG(account.name ORDER BY posting.id))[2] as account,
       (ARRAY_AGG(posting.id ORDER BY posting.id))[2] as posting_id,
       (ARRAY_AGG(account.id ORDER BY posting.id))[2] as account_id,
       (ARRAY_AGG(amount ORDER BY posting.id))[1] as amount,
       (ARRAY_AGG(currency.name ORDER BY posting.id))[1] as currency
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
		INNER JOIN currency ON (currency.id = posting.currency_id)
		INNER JOIN account ON (account.id = posting.account_id)
		WHERE transaction.id > @after
		GROUP BY transaction.id
		ORDER BY transaction.id
		LIMIT @size`,
		pgx.NamedArgs{
			"after": after,
			"size":  size,
		})

	if errQuery != nil {
		return nil, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get transactions to classify")
	}

	dtos, err := pgx.CollectRows(rows, pgx.RowToStructByName[dto.TransactionToClassify])

	if err != nil {
		return nil, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to collect rows transactions to classify")
	}

	for _, toClassify := range dtos {
		txs[toClassify.ID] = domain.TransactionToClassify{
			ID:        toClassify.ID,
			Date:      toClassify.Date,
			Payee:     toClassify.Payee,
			PostingID: toClassify.PostingID,
			AccountID: toClassify.AccountID,
			Account:   toClassify.Account,
			Amount:    toClassify.Amount,
			Currency:  toClassify.Currency,
		}
	}

	return txs, nil
}

func (t *Transaction) GetRecentTransactions(ctx context.Context, size uint) ([]api.Transaction, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetRecent")
	defer span.End()

	txs := make([]api.Transaction, 0, size)

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get recent transactions"),
		`SELECT date,
       narration,
       (ARRAY_AGG(account.name ORDER BY posting.id))[2] as account,
       (ARRAY_AGG(amount ORDER BY posting.id))[1] as amount,
       (ARRAY_AGG(currency.name ORDER BY posting.id))[1] as currency
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
		INNER JOIN currency ON (currency.id = posting.currency_id)
		INNER JOIN account ON (account.id = posting.account_id)
		GROUP BY transaction.id
		ORDER BY transaction.id DESC
		LIMIT @size`,
		pgx.NamedArgs{
			"size": size,
		})

	if errQuery != nil {
		fmt.Printf("err %+v\n", errQuery)

		return nil, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get recent transactions")
	}

	dtos, errCollect := pgx.CollectRows(rows, pgx.RowToStructByName[dto.RecentTransaction])

	if errCollect != nil {
		return nil, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCollect, "failed to collect rows recent transactions")
	}

	for _, dbTransaction := range dtos {
		txs = append(txs, api.Transaction{
			Date:      dbTransaction.Date,
			Narration: dbTransaction.Narration,
			Account:   dbTransaction.Account,
			Amount:    dbTransaction.Amount,
			Currency:  dbTransaction.Currency,
		})
	}

	println("boum", len(txs))

	return txs, nil
}

func (t *Transaction) Save(ctx context.Context, transactions []domain.Transaction) error {
	ctx, span := t.tracer.Start(ctx, "Transaction.Save")
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

	tx, errBegin := t.pool.Begin(context.WithValue(ctx, SQLName, "begin transaction"))

	if errBegin != nil {
		return oops.
			In("postgres.transaction").
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
			In("postgres.transaction").
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
			In("postgres.transaction").
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
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCmt, "failed to commit data")
	}

	return nil
}

func (t *Transaction) GetCheckingBalance(ctx context.Context) (int64, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetCheckingBalance")
	defer span.End()

	var total int64

	row := t.pool.QueryRow(
		context.WithValue(ctx, SQLName, "get checkingBalance"),
		`SELECT COALESCE(CAST(ROUND(SUM(amount)) AS bigint), 0) AS total
		FROM posting
		WHERE account_id IN (SELECT id FROM account WHERE type = 'Assets' AND name = 'Bank:Checking')`,
	)

	errScan := row.Scan(&total)
	if errScan != nil {
		return 0, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errScan, "failed to scan transaction")
	}

	return total, nil
}

func (t *Transaction) BalanceSummary(ctx context.Context) (domain.BalanceSummary, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetCheckingBalance")
	defer span.End()

	var balanceSummary domain.BalanceSummary

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get balanceSummary"),
		`SELECT a.name, COALESCE(CAST(ROUND(SUM(amount)) AS bigint), 0) AS amount, currency.name
		FROM account AS a
		INNER JOIN posting ON posting.account_id = a.id
		INNER JOIN currency ON currency.id = posting.currency_id
		WHERE type = 'Assets' AND a.name IN ('Bank:Checking', 'Bank:Savings', 'Bank:Investment')
		GROUP BY a.id, currency.id`,
	)

	if errQuery != nil {
		return balanceSummary, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get balanceSummary")
	}

	var name string
	var amount string
	var currencyName string

	totalAmount, _ := currency.NewAmount("0", "EUR")
	balanceSummary.TotalBalance = totalAmount

	_, errScan := pgx.ForEachRow(
		rows,
		[]any{&name, &amount, &currencyName},
		func() error {
			currencyAmount, errCurrency := currency.NewAmount(amount, currencyName)
			if errCurrency != nil {
				return oops.
					In("postgres.transaction").
					WithContext(ctx).
					Wrapf(errCurrency, "failed to create currency amount for %s", name)
			}

			if name == "Bank:Checking" {
				amountWithChecking, errAddChecking := balanceSummary.TotalBalance.Add(currencyAmount)
				if errAddChecking != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddChecking, "failed to add checking balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithChecking
				balanceSummary.CheckingBalance = currencyAmount
			}

			if name == "Bank:Savings" {
				amountWithSavings, errAddSavings := balanceSummary.TotalBalance.Add(currencyAmount)
				if errAddSavings != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddSavings, "failed to add savings balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithSavings
				balanceSummary.SavingsBalance = currencyAmount
			}

			if name == "Bank:Investment" {
				amountWithInvestment, errAddInvestment := balanceSummary.TotalBalance.Add(currencyAmount)
				if errAddInvestment != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddInvestment, "failed to add investment balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithInvestment
				balanceSummary.InvestmentBalance = currencyAmount
			}

			return nil

		})

	if errScan != nil {
		return balanceSummary, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errScan, "failed to scan transaction")
	}

	return balanceSummary, nil
}
