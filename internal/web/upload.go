package web

import (
	"log/slog"
	"net/http"

	"github.com/abiosoft/mold"
	"github.com/go-chi/chi/v5"
	"github.com/syrm/maille/internal"
	"go.opentelemetry.io/otel/trace"
)

type Upload struct {
	Importer internal.Importer
	Engine   mold.Engine
	Tracer   trace.Tracer
	Logger   *slog.Logger
}

func (u Upload) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/upload", u.Get)
	r.Post("/upload", u.Post)
	r.Get("/transaction-classifier", u.GetTransactionClassifier)

	return r
}

func (u Upload) GetTransactionClassifier(w http.ResponseWriter, r *http.Request) {
	// transactions, e := u.PostgresTransaction.GetTransactions(r.Context())

	// if e != nil {
	// 	println(e.Error())
	// }

	// startComp := time.Now()
	// amazon := `payee contains "AMAZON" and any(postings, .account == "FR:BNP:Checking" and .amount < 0)`
	// programA, errCompile := expr.Compile(amazon, expr.Env(domain.Transaction{}), expr.AsBool())

	// if errCompile != nil {
	// 	fmt.Print("errComp program A ", errCompile.Error(), "\n")
	// 	return
	// }

	// netflix := `payee contains "NETFLIX" and posting.amount < 0`
	// programN, _ := expr.Compile(netflix, expr.Env(domain.Transaction{}), expr.AsBool())
	// fmt.Println("time comp ", time.Since(startComp).Microseconds())
	// timeCheck := 0
	// for _, transaction := range transactions {

	// 	startC := time.Now()
	// 	match, err := expr.Run(programA, transaction)
	// 	timeCheck += int(time.Since(startC).Microseconds())
	// 	if err != nil {
	// 		fmt.Print(err.Error())
	// 		continue
	// 	}

	// 	isMatch, ok := match.(bool)
	// 	if !ok {
	// 		fmt.Printf("rule expected bool, got %T\n", match)
	// 		continue
	// 	}

	// 	if isMatch {
	// 		w.Write([]byte("ITSSSSSSSSS A AMAZON MATTTCHH"))
	// 	}

	// 	startC = time.Now()
	// 	match, _ = expr.Run(programN, transaction)
	// 	timeCheck += int(time.Since(startC).Microseconds())
	// 	if match != nil {
	// 		w.Write([]byte("ITSSSSSSSSS A NETFLIX MATTTCHH"))
	// 	}
	// 	w.Write([]byte(fmt.Sprintf("%+v\n", transaction)))
	// }

	// fmt.Println("time check ", timeCheck)
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
		w.Write([]byte("Good 3"))
		return
		// @TODO redirection
	}

	errImport := u.Importer.Import(ctx, file)
	// errParse := u.Parser.Parse(r.Context(), file, 200_000, u.PostgresTransaction.Process)

	if errImport != nil {
		u.Logger.ErrorContext(r.Context(), "failed to import file", slog.Any("error", errImport))
		w.Write([]byte("Good 5"))
		return
		// @TODO redirection
	}

	w.Write([]byte("Good"))
}
