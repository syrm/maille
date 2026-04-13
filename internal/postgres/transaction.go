package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bojanz/currency"
	pkgcurrency "github.com/bojanz/currency"
	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/oops"
	"go.opentelemetry.io/otel/trace"

	"github.com/syrm/maille/internal/domain"
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
		amount    pkgcurrency.Amount
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
       (ARRAY_AGG(amount ORDER BY posting.id))[1] as amount
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
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

	dtos, errCollect := pgx.CollectRows(rows, pgx.RowToStructByName[dto.TransactionToClassify])

	if errCollect != nil {
		return nil, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCollect, "failed to collect rows transactions to classify")
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
		}
	}

	return txs, nil
}

func (t *Transaction) GetRecentTransactions(ctx context.Context, size uint) ([]domain.RecentTransaction, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetRecentTransactions")
	defer span.End()

	txs := make([]domain.RecentTransaction, 0, size)

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get recent transactions"),
		`SELECT date,
       payee,
       (ARRAY_AGG(account.name ORDER BY posting.id))[2] as account,
       (ARRAY_AGG(account.alias ORDER BY posting.id))[2] as alias,
       (ARRAY_AGG(account.icon ORDER BY posting.id))[2] as icon,
       (ARRAY_AGG(account.color ORDER BY posting.id))[2] as color,
       (ARRAY_AGG(amount ORDER BY posting.id))[1] as amount
		FROM transaction
		INNER JOIN posting ON (posting.transaction_id = transaction.id)
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
		txs = append(txs, domain.RecentTransaction{
			Date:    dbTransaction.Date,
			Payee:   dbTransaction.Payee,
			Account: dbTransaction.Account,
			Amount:  dbTransaction.Amount,
			Alias:   dbTransaction.Alias,
			Icon:    dbTransaction.Icon,
			Color:   dbTransaction.Color,
		})
	}

	return txs, nil
}

func (t *Transaction) Save(ctx context.Context, transactions []domain.Transaction) error {
	ctx, span := t.tracer.Start(ctx, "Transaction.Save")
	defer span.End()

	node, _ := snowflake.NewNode(1)

	// Build text data for both transaction and posting tables in a single loop
	var trSb strings.Builder
	var posSb strings.Builder

	for _, transaction := range transactions {
		transactionID := node.Generate()

		// Add transaction to transaction builder
		date := transaction.Date.Format(time.DateOnly)
		trSb.WriteString(fmt.Sprintf("%d\t%s\tt\t%s\t%s\n",
			transactionID,
			date,
			transaction.Payee,
			transaction.ExternalID))

		// Add postings for this transaction using the same transactionID
		for _, posting := range transaction.Postings {
			postingID := node.Generate()
			// Format price composite type as (number,currency_code)
			posSb.WriteString(fmt.Sprintf("%d\t%d\t%d\t(%s,%s)\n",
				postingID,
				transactionID,
				posting.AccountID,
				posting.Amount.Number(),
				posting.Amount.CurrencyCode()))
		}
	}
	trSb.WriteString("\\.\n")
	posSb.WriteString("\\.\n")

	tx, errBegin := t.pool.Begin(context.WithValue(ctx, SQLName, "begin transaction"))
	if errBegin != nil {
		return oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errBegin, "failed to begin transaction")
	}
	defer tx.Rollback(context.WithValue(ctx, SQLName, "rollback transaction"))

	// Copy transaction data using text format
	pgConn := tx.Conn().PgConn()
	trReader := strings.NewReader(trSb.String())
	_, errCopyTr := pgConn.CopyFrom(
		context.WithValue(ctx, SQLName, "copy transaction"),
		trReader,
		"COPY transaction (id, date, completed, payee, external_id) FROM STDIN",
	)
	if errCopyTr != nil {
		return oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCopyTr, "failed to exec copy transaction")
	}

	// Copy posting data using text format
	posReader := strings.NewReader(posSb.String())
	_, errCopyPosting := pgConn.CopyFrom(
		context.WithValue(ctx, SQLName, "copy posting"),
		posReader,
		"COPY posting (id, transaction_id, account_id, amount) FROM STDIN",
	)
	if errCopyPosting != nil {
		return oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCopyPosting, "failed to exec copy posting")
	}

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
		`SELECT a.name, COALESCE(ROW(ROUND(SUM((amount).number)), 'EUR')::price, ROW(0, 'EUR')::price) AS amount
		FROM account AS a
		INNER JOIN posting ON posting.account_id = a.id
		WHERE type = 'Assets' AND a.name IN ('Bank:Checking', 'Bank:Savings', 'Bank:Investment')
		GROUP BY a.id`,
	)

	if errQuery != nil {
		return balanceSummary, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get balanceSummary")
	}

	var name string
	var amount pkgcurrency.Amount

	totalAmount, _ := currency.NewAmount("0", "EUR")
	balanceSummary.TotalBalance = totalAmount

	_, errScan := pgx.ForEachRow(
		rows,
		[]any{&name, &amount},
		func() error {
			if name == "Bank:Checking" {
				amountWithChecking, errAddChecking := balanceSummary.TotalBalance.Add(amount)
				if errAddChecking != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddChecking, "failed to add checking balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithChecking
				balanceSummary.CheckingBalance = amount
			}

			if name == "Bank:Savings" {
				amountWithSavings, errAddSavings := balanceSummary.TotalBalance.Add(amount)
				if errAddSavings != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddSavings, "failed to add savings balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithSavings
				balanceSummary.SavingsBalance = amount
			}

			if name == "Bank:Investment" {
				amountWithInvestment, errAddInvestment := balanceSummary.TotalBalance.Add(amount)
				if errAddInvestment != nil {
					return oops.
						In("postgres.transaction").
						WithContext(ctx).
						Wrapf(errAddInvestment, "failed to add investment balance for %s", name)
				}

				balanceSummary.TotalBalance = amountWithInvestment
				balanceSummary.InvestmentBalance = amount
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
