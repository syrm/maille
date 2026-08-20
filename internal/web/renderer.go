package web

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/CloudyKit/jet/v6"
)

type Renderer struct {
	Views *jet.Set
}

func (r *Renderer) Render(w http.ResponseWriter, templateName string, variables jet.VarMap) error {
	t, err := r.Views.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("get template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, variables, nil); err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = buf.WriteTo(w)
	return err
}
