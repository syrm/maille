package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	LangKey = contextKey("lang")
)

func Language(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := parseAcceptLanguage(r)
		ctx := context.WithValue(r.Context(), LangKey, lang)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func parseAcceptLanguage(r *http.Request) string {
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return "en" // fallback
	}

	// "fr-FR,fr;q=0.9,en-US;q=0.8" → "fr-FR"
	first := strings.SplitN(header, ",", 2)[0]

	// "fr-FR" → "fr-FR"  /  "fr;q=0.9" → "fr"
	lang := strings.SplitN(first, ";", 2)[0]

	//"fr-FR" → "fr"
	//lang = strings.SplitN(lang, "-", 2)[0]

	return strings.TrimSpace(strings.Replace(lang, "-", "_", 1))
}
