package web

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/go-chi/chi/v5"

	"github.com/syrm/maille/internal/domain"
	"github.com/syrm/maille/internal/middleware"
)

type classificationRuleStore interface {
	GetAll(context.Context) ([]domain.TransactionClassifierRule, error)
	Create(context.Context, string, uint64) error
	Update(context.Context, uint64, string, uint64) error
	Delete(context.Context, uint64) error
}

type classificationEngine interface {
	ValidateRule(string) error
	Reset(context.Context) error
	Classify(context.Context) error
}

type Rules struct {
	Renderer Renderer
	Rules    classificationRuleStore
	Accounts categoryProvider
	Engine   classificationEngine
	Logger   *slog.Logger
}

type classificationRuleView struct {
	ID       uint64
	Priority int
	Rule     string
	Account  domain.Account
}

func (h Rules) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", h.Get)
	r.Post("/", h.PostCreate)
	r.Post("/run", h.PostRun)
	r.Post("/{ruleID}", h.PostUpdate)
	r.Post("/{ruleID}/delete", h.PostDelete)
	return r
}

func (h Rules) Get(w http.ResponseWriter, r *http.Request) {
	rules, err := h.Rules.GetAll(r.Context())
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to list classification rules", slog.Any("error", err))
		http.Error(w, "Impossible de charger les règles.", http.StatusInternalServerError)
		return
	}
	accounts, err := h.Accounts.GetAll(r.Context())
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to list rule categories", slog.Any("error", err))
		http.Error(w, "Impossible de charger les catégories.", http.StatusInternalServerError)
		return
	}
	ruleViews := make([]classificationRuleView, 0, len(rules))
	for index, rule := range rules {
		ruleViews = append(ruleViews, classificationRuleView{
			ID: rule.ID, Priority: index + 1, Rule: rule.Rule, Account: rule.Account,
		})
	}

	variables := jet.VarMap{}
	variables.Set("lang", r.Context().Value(middleware.LangKey))
	variables.Set("pageTitle", "Règles de classification")
	variables.Set("currentPage", "rules")
	variables.Set("rules", ruleViews)
	variables.Set("hasRules", len(rules) > 0)
	variables.Set("categories", categoryOptions(accounts))
	variables.Set("notice", ruleNotice(r.URL.Query().Get("status")))
	variables.Set("errorMessage", ruleError(r.URL.Query().Get("error")))

	if err := h.Renderer.Render(w, "rules", variables); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h Rules) PostCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	rule := strings.TrimSpace(r.FormValue("rule"))
	accountID, errAccountID := strconv.ParseUint(r.FormValue("account_id"), 10, 64)
	if len(rule) == 0 || len(rule) > 500 || errAccountID != nil || accountID == 0 {
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	if err := h.Engine.ValidateRule(rule); err != nil {
		h.Logger.InfoContext(r.Context(), "invalid classification rule", slog.String("rule", rule), slog.Any("error", err))
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	if err := h.Rules.Create(r.Context(), rule, accountID); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to create classification rule", slog.Any("error", err))
		redirectRules(w, r, "", "create-failed")
		return
	}
	if err := h.Engine.Classify(r.Context()); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to run new classification rule", slog.Any("error", err))
		redirectRules(w, r, "created", "run-failed")
		return
	}
	redirectRules(w, r, "created", "")
}

func (h Rules) PostRun(w http.ResponseWriter, r *http.Request) {
	if err := h.Engine.Classify(r.Context()); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to run classification rules", slog.Any("error", err))
		redirectRules(w, r, "", "run-failed")
		return
	}
	redirectRules(w, r, "ran", "")
}

func (h Rules) PostUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	ruleID, errRuleID := strconv.ParseUint(chi.URLParam(r, "ruleID"), 10, 64)
	rule := strings.TrimSpace(r.FormValue("rule"))
	accountID, errAccountID := strconv.ParseUint(r.FormValue("account_id"), 10, 64)
	if errRuleID != nil || ruleID == 0 || len(rule) == 0 || len(rule) > 500 || errAccountID != nil || accountID == 0 {
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	if err := h.Engine.ValidateRule(rule); err != nil {
		h.Logger.InfoContext(r.Context(), "invalid classification rule update", slog.Uint64("rule_id", ruleID), slog.Any("error", err))
		redirectRules(w, r, "", "invalid-rule")
		return
	}
	if err := h.Rules.Update(r.Context(), ruleID, rule, accountID); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to update classification rule", slog.Uint64("rule_id", ruleID), slog.Any("error", err))
		redirectRules(w, r, "", "update-failed")
		return
	}
	if err := h.Engine.Classify(r.Context()); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to run updated classification rule", slog.Uint64("rule_id", ruleID), slog.Any("error", err))
		redirectRules(w, r, "updated", "run-failed")
		return
	}
	redirectRules(w, r, "updated", "")
}

func (h Rules) PostDelete(w http.ResponseWriter, r *http.Request) {
	ruleID, err := strconv.ParseUint(chi.URLParam(r, "ruleID"), 10, 64)
	if err != nil || ruleID == 0 {
		redirectRules(w, r, "", "delete-failed")
		return
	}
	if err := h.Engine.Reset(r.Context()); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to reset classifications before deleting rule", slog.Any("error", err))
		redirectRules(w, r, "", "delete-failed")
		return
	}
	if err := h.Rules.Delete(r.Context(), ruleID); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to delete classification rule", slog.Any("error", err))
		if classifyErr := h.Engine.Classify(r.Context()); classifyErr != nil {
			h.Logger.ErrorContext(r.Context(), "failed to restore classifications after rule deletion failed", slog.Any("error", classifyErr))
		}
		redirectRules(w, r, "", "delete-failed")
		return
	}
	if err := h.Engine.Classify(r.Context()); err != nil {
		h.Logger.ErrorContext(r.Context(), "failed to reclassify after deleting rule", slog.Any("error", err))
		redirectRules(w, r, "deleted", "run-failed")
		return
	}
	redirectRules(w, r, "deleted", "")
}

func redirectRules(w http.ResponseWriter, r *http.Request, status, errorCode string) {
	location := "/rules"
	separator := "?"
	if status != "" {
		location += separator + "status=" + status
		separator = "&"
	}
	if errorCode != "" {
		location += separator + "error=" + errorCode
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func ruleNotice(status string) string {
	switch status {
	case "created":
		return "Règle créée et moteur de classification exécuté."
	case "updated":
		return "Règle modifiée et classifications recalculées."
	case "deleted":
		return "Règle supprimée et classifications recalculées."
	case "ran":
		return "Moteur de classification exécuté."
	default:
		return ""
	}
}

func ruleError(code string) string {
	switch code {
	case "invalid-rule":
		return "Expression invalide. Vérifiez sa syntaxe et la catégorie cible."
	case "create-failed":
		return "La règle n’a pas pu être créée."
	case "update-failed":
		return "La règle n’a pas pu être modifiée."
	case "delete-failed":
		return "La règle n’a pas pu être supprimée."
	case "run-failed":
		return "Le moteur n’a pas pu appliquer toutes les règles."
	default:
		return ""
	}
}
