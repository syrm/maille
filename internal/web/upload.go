package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/abiosoft/mold"
	"github.com/expr-lang/expr"
	"github.com/go-chi/chi/v5"
	"github.com/syrm/maille/internal"
	"go.opentelemetry.io/otel/trace"
)

type PostingEvalCtx struct {
	Payee     string       `expr:"payee"`
	Narration string       `expr:"narration"`
	Date      time.Time    `expr:"date"`
	DayOfWeek time.Weekday `expr:"day_of_week"`
	Month     int          `expr:"month"`
	Year      int          `expr:"year"`

	Amount      float64 `expr:"amount"`
	Currency    string  `expr:"currency"`
	AccountName string  `expr:"account"`
}

type Upload struct {
	AccountStore     internal.AccountStore
	TransactionStore internal.TransactionStore
	Importer         internal.Importer
	Classifier       internal.Classifier
	Engine           mold.Engine
	Tracer           trace.Tracer
	Logger           *slog.Logger
}

func (u Upload) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/upload", u.Get)
	r.Post("/upload", u.Post)
	r.Get("/transaction-classifier", u.GetTransactionClassifier)

	return r
}

func (u Upload) GetTransactionClassifier(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	accountsID := make(map[uint64]string)
	{
		accounts, errAccount := u.AccountStore.GetAll(ctx)
		if errAccount != nil {
			println(errAccount)
			// return oops.
			// 	In("importer").
			// 	WithContext(ctx).
			// 	Wrapf(errAccount, "failed to get accounts")
		}

		for _, account := range accounts {
			accountsID[account.ID] = string(account.Type) + ":" + account.Name
		}
	}

	transactions, e := u.TransactionStore.GetAll(r.Context(), 0, 1000)

	if e != nil {
		println(e.Error())
	}

	postingEvalCtx := make([]PostingEvalCtx, 0, len(transactions))

	for _, transaction := range transactions {
		accountName, ok := accountsID[transaction.Postings[0].AccountID]

		if !ok {
			fmt.Println("accountID pas ok")
			continue
		}

		narration := ""

		if transaction.Narration != nil {
			narration = *transaction.Narration
		}

		postingEvalCtx = append(postingEvalCtx, PostingEvalCtx{
			Payee:       transaction.Payee,
			Narration:   narration,
			Date:        transaction.Date,
			DayOfWeek:   transaction.Date.Weekday(),
			Month:       int(transaction.Date.Month()),
			Year:        transaction.Date.Year(),
			Amount:      transaction.Postings[0].Amount,
			Currency:    transaction.Postings[0].Currency.Name,
			AccountName: accountName,
		})
	}

	startComp := time.Now()
	amazon := `payee contains "AMAZON" and account == "Assets:Bank:Checking" and amount < 0`
	programA, errCompile := expr.Compile(amazon, expr.Env(PostingEvalCtx{}), expr.AsBool())

	if errCompile != nil {
		fmt.Print("errComp program A ", errCompile.Error(), "\n")
		return
	}

	netflix := `payee contains "NETFLIX" and amount < 0`
	programN, errCompN := expr.Compile(netflix, expr.Env(PostingEvalCtx{}), expr.AsBool())
	if errCompN != nil {
		fmt.Print("errComp program A ", errCompN.Error(), "\n")
		return
	}
	fmt.Println("time comp ", time.Since(startComp).Microseconds())
	timeCheck := 0
	for _, transaction := range postingEvalCtx {

		startC := time.Now()
		match, err := expr.Run(programA, transaction)
		timeCheck += int(time.Since(startC).Microseconds())
		if err != nil {
			fmt.Print(err.Error())
			continue
		}

		isMatch, ok := match.(bool)
		if !ok {
			fmt.Printf("rule expected bool, got %T\n", match)
			continue
		}

		if isMatch {
			w.Write([]byte("ITSSSSSSSSS A AMAZON MATTTCHH"))
		}

		startC = time.Now()
		match, _ = expr.Run(programN, transaction)
		timeCheck += int(time.Since(startC).Microseconds())

		isMatch, ok = match.(bool)
		if !ok {
			fmt.Printf("rule expected bool, got %T\n", match)
			continue
		}

		if isMatch {
			w.Write([]byte("ITSSSSSSSSS A NETFLIX MATTTCHH"))
		}
		w.Write([]byte(fmt.Sprintf("%+v\n", transaction)))
	}

	fmt.Println("time check ", timeCheck)
}

func (u Upload) Get(w http.ResponseWriter, r *http.Request) {
	errRender := u.Engine.Render(w, "template/upload.html", nil)

	if errRender != nil {
		u.Logger.ErrorContext(r.Context(), "failed to render", slog.Any("error", errRender))
	}
}

func (u Upload) Post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		w.Write([]byte("Good 1"))
		return
	}

	errForm := r.ParseForm()

	if errForm != nil {
		u.Logger.ErrorContext(r.Context(), "failed to parse form", slog.Any("error", errForm))
		w.Write([]byte("Good 2"))
		return
		// @TODO redirection
	}

	file, _, errFile := r.FormFile("file")
	_ = file
	_ = errFile

	if errFile != nil {
		u.Logger.ErrorContext(r.Context(), "failed to read file form", slog.Any("error", errForm))
		w.Write([]byte("pas good 2"))
		return
		// @TODO redirection
	}

	// errImport := u.Importer.Import(ctx, file)
	// if errImport != nil {
	// 	u.Logger.ErrorContext(r.Context(), "failed to import file", slog.Any("error", errImport))
	// 	w.Write([]byte("Good 5"))
	// 	return
	// 	// @TODO redirection
	// }

	errClass := u.Classifier.Classify(ctx)
	if errClass != nil {
		u.Logger.ErrorContext(r.Context(), "failed to classify transaction", slog.Any("error", errClass))
		w.Write([]byte("pas good"))
		return
		// @TODO redirection
	}

	w.Write([]byte("Good"))
}
