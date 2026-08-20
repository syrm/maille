package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bojanz/currency"
	pkgcurrency "github.com/bojanz/currency"
	"github.com/bwmarrin/snowflake"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func copyText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\t", `\t`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, "\r", `\r`)
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
		WHERE transaction.id > @after AND transaction.classified_at IS NULL
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

func (t *Transaction) List(ctx context.Context, filter domain.TransactionListFilter) ([]domain.TransactionListItem, uint64, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.List")
	defer span.End()

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "list transactions"),
		`WITH transaction_rows AS (
			SELECT transaction.id,
			       transaction.date,
			       transaction.payee,
			       transaction.classified_at IS NOT NULL AS classified,
			       (ARRAY_AGG(account.id ORDER BY posting.id))[2] AS category_id,
			       (ARRAY_AGG(account.name ORDER BY posting.id))[2] AS category,
			       (ARRAY_AGG(account.alias ORDER BY posting.id))[2] AS category_alias,
			       (ARRAY_AGG(account.icon ORDER BY posting.id))[2] AS category_icon,
			       (ARRAY_AGG(account.color ORDER BY posting.id))[2] AS category_color,
			       (ARRAY_AGG(account.alias ORDER BY posting.id))[1] AS bank_account,
			       (ARRAY_AGG(posting.amount ORDER BY posting.id))[1] AS amount
			FROM transaction
			INNER JOIN posting ON posting.transaction_id = transaction.id
			INNER JOIN account ON account.id = posting.account_id
			GROUP BY transaction.id
		)
		SELECT id, date, payee, classified, category_id, category, category_alias,
		       category_icon, category_color, bank_account, amount, COUNT(*) OVER() AS total
		FROM transaction_rows
		WHERE (@search = '' OR payee ILIKE '%' || @search || '%')
		  AND (@categoryID = 0 OR category_id = @categoryID)
		  AND (NOT @uncategorized OR NOT classified)
		ORDER BY date DESC, id DESC
		LIMIT @limit OFFSET @offset`,
		pgx.NamedArgs{
			"search":        filter.Search,
			"categoryID":    filter.CategoryID,
			"uncategorized": filter.Uncategorized,
			"limit":         filter.Limit,
			"offset":        filter.Offset,
		},
	)
	if errQuery != nil {
		return nil, 0, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to list transactions")
	}
	defer rows.Close()

	transactions := make([]domain.TransactionListItem, 0, filter.Limit)
	var total uint64
	for rows.Next() {
		var transaction domain.TransactionListItem
		if err := rows.Scan(
			&transaction.ID,
			&transaction.Date,
			&transaction.Payee,
			&transaction.Classified,
			&transaction.CategoryID,
			&transaction.Category,
			&transaction.CategoryAlias,
			&transaction.CategoryIcon,
			&transaction.CategoryColor,
			&transaction.BankAccount,
			&transaction.Amount,
			&total,
		); err != nil {
			return nil, 0, oops.
				In("postgres.transaction").
				WithContext(ctx).
				Wrapf(err, "failed to scan transaction list")
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(err, "failed to iterate transaction list")
	}

	return transactions, total, nil
}

func (t *Transaction) UpdateCategory(ctx context.Context, transactionID uint64, accountID uint64) error {
	ctx, span := t.tracer.Start(ctx, "Transaction.UpdateCategory")
	defer span.End()

	result, errExec := t.pool.Exec(
		context.WithValue(ctx, SQLName, "update transaction category"),
		`WITH updated_posting AS (
		    UPDATE posting
		    SET account_id = @accountID
		    WHERE transaction_id = @transactionID
		      AND id = (
		          SELECT posting_to_update.id
		          FROM posting AS posting_to_update
		          INNER JOIN account AS current_account ON current_account.id = posting_to_update.account_id
		          WHERE posting_to_update.transaction_id = @transactionID
		            AND current_account.type <> 'Assets'
		          ORDER BY posting_to_update.id
		          LIMIT 1
		      )
		      AND EXISTS (
		          SELECT 1 FROM account AS target_account
		          WHERE target_account.id = @accountID
		            AND target_account.type IN ('Expenses', 'Income')
		      )
		    RETURNING transaction_id
		)
		UPDATE transaction
		SET classified_at = CURRENT_TIMESTAMP
		WHERE id = (SELECT transaction_id FROM updated_posting)`,
		pgx.NamedArgs{
			"transactionID": transactionID,
			"accountID":     accountID,
		},
	)
	if errExec != nil {
		return oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errExec, "failed to update transaction category")
	}
	if result.RowsAffected() != 1 {
		return oops.
			In("postgres.transaction").
			WithContext(ctx).
			With("transaction_id", transactionID).
			With("account_id", accountID).
			Errorf("transaction or category not found")
	}

	return nil
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
		trSb.WriteString(fmt.Sprintf("%d\t%s\tt\t%s\t%s\t%s\n",
			transactionID,
			date,
			copyText(transaction.Payee),
			copyText(transaction.ExternalID),
			copyText(transaction.ExternalID)))

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
		"COPY transaction (id, date, completed, payee, external_id, import_key) FROM STDIN",
	)
	if errCopyTr != nil {
		var pgErr *pgconn.PgError
		if errors.As(errCopyTr, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDuplicateTransaction
		}
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
		INNER JOIN transaction ON posting.transaction_id = transaction.id
		WHERE type = 'Assets' AND a.name IN ('Bank:Checking', 'Bank:Savings', 'Bank:Investment')
		AND transaction.date <= CURRENT_DATE
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

func (t *Transaction) NetWorthHistory(ctx context.Context) ([]domain.NetWorthHistory, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.GetCheckingBalance")
	defer span.End()

	var netWorthHistory []domain.NetWorthHistory

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get net worth history"),
		`WITH bounds AS (
    SELECT DATE_TRUNC('month', CURRENT_DATE - INTERVAL '12 MONTHS') AS window_start,
           DATE_TRUNC('month', CURRENT_DATE) AS window_end
),
months AS (
    SELECT generate_series(window_start, window_end, INTERVAL '1 month') AS month
    FROM bounds
),
accounts AS (
    SELECT id, name
    FROM account
    WHERE type = 'Assets'
      AND name IN ('Bank:Checking', 'Bank:Savings', 'Bank:Investment')
),
monthly_raw AS (
    SELECT
        a.id AS account_id,
        GREATEST(DATE_TRUNC('month', transaction.date), b.window_start) AS month,
        CAST(ROUND(SUM((posting.amount).number)) AS bigint) AS amount
    FROM accounts a
    INNER JOIN posting     ON posting.account_id     = a.id
    INNER JOIN transaction ON posting.transaction_id = transaction.id
    CROSS JOIN bounds b
    WHERE transaction.date <= CURRENT_DATE
    GROUP BY a.id, GREATEST(DATE_TRUNC('month', transaction.date), b.window_start)
),
per_month AS (
    SELECT
        a.id AS account_id,
        a.name,
        m.month,
        SUM(COALESCE(mr.amount, 0)) OVER (
            PARTITION BY a.id
            ORDER BY m.month
        ) AS amount_cumulative
    FROM accounts a
    CROSS JOIN months m
    LEFT JOIN monthly_raw mr
           ON mr.account_id = a.id
          AND mr.month      = m.month
)
SELECT
    name,
    ARRAY_AGG(month             ORDER BY month) AS dates,
    ARRAY_AGG(amount_cumulative ORDER BY month) AS amounts
FROM per_month
GROUP BY name
ORDER BY name`,
	)

	if errQuery != nil {
		return netWorthHistory, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get net worth history")
	}

	dtos, errCollect := pgx.CollectRows(rows, pgx.RowToStructByName[dto.NetWorthHistory])

	if errCollect != nil {
		return netWorthHistory, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCollect, "failed to collect rows net worth history")
	}

	for _, dbNetWorthHistory := range dtos {
		var amounts []pkgcurrency.Amount
		for _, amount := range dbNetWorthHistory.Amounts {
			amountCurr, errCurr := pkgcurrency.NewAmount(strconv.FormatFloat(amount, 'f', 2, 64), "EUR")
			if errCurr != nil {
				return netWorthHistory, oops.
					In("postgres.transaction").
					WithContext(ctx).
					Wrapf(errCurr, "failed to create currency for amount %f", amount)
			}
			amounts = append(amounts, amountCurr)
		}
		netWorthHistory = append(netWorthHistory, domain.NetWorthHistory{
			Name:    dbNetWorthHistory.Name,
			Dates:   dbNetWorthHistory.Dates,
			Amounts: amounts,
		})
	}

	return netWorthHistory, nil
}

func (t *Transaction) BreakdownCategory(ctx context.Context) ([]domain.BreakdownCategory, error) {
	ctx, span := t.tracer.Start(ctx, "Transaction.BreakdownCategory")
	defer span.End()

	var breakdownCategories []domain.BreakdownCategory

	rows, errQuery := t.pool.Query(
		context.WithValue(ctx, SQLName, "get breakdown category"),
		`SELECT a.alias as name, COALESCE(CAST(ROUND(SUM((amount).number)) AS bigint), 0) AS amount
		FROM account AS a
		INNER JOIN posting ON posting.account_id = a.id
		WHERE type = 'Expenses'
		GROUP BY a.id`,
	)

	if errQuery != nil {
		return breakdownCategories, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errQuery, "failed to get breakdown category")
	}

	dtos, errCollect := pgx.CollectRows(rows, pgx.RowToStructByName[dto.BreakdownCategory])

	if errCollect != nil {
		return breakdownCategories, oops.
			In("postgres.transaction").
			WithContext(ctx).
			Wrapf(errCollect, "failed to collect rows breakdown category")
	}

	for _, dbBreakdownCategory := range dtos {
		amountCurr, errCurr := pkgcurrency.NewAmount(strconv.FormatFloat(dbBreakdownCategory.Amount, 'f', 2, 64), "EUR")
		if errCurr != nil {
			return breakdownCategories, oops.
				In("postgres.transaction").
				WithContext(ctx).
				Wrapf(errCurr, "failed to create currency for amount %f", dbBreakdownCategory.Amount)
		}
		breakdownCategories = append(breakdownCategories, domain.BreakdownCategory{
			Name:   dbBreakdownCategory.Name,
			Amount: amountCurr,
		})
	}

	return breakdownCategories, nil
}
