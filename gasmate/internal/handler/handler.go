package handler

import (
	"database/sql"
	"html/template"
	"net/http"

	"github.com/glnarayanan/egassewa/gasmate/internal/auth"
	"github.com/glnarayanan/egassewa/gasmate/internal/email"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB       *sql.DB
	Pages    map[string]*template.Template // page name → layout+page clone
	Partials *template.Template            // all HTMX partial templates
	Email    email.Config
}

type templateData struct {
	Session *auth.Session
	Data    any
	Flash   string
	Error   string
}

// render executes a full-page template (layout + content).
// The entry point is either "base" (public pages) or "dashboard" (role portals).
func (h *Handler) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	tmpl, ok := h.Pages[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	td := templateData{
		Session: sessionFrom(r),
		Data:    data,
	}
	entry := entryPoint(name)
	if err := tmpl.ExecuteTemplate(w, entry, td); err != nil {
		// Header likely already partially written; log only
		_ = err
	}
}

// renderPartial executes an HTMX partial (no layout).
func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	if err := h.Partials.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "partial error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, url string) {
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, tmpl, msg string) {
	pageT, ok := h.Pages[tmpl]
	if !ok {
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	td := templateData{Session: sessionFrom(r), Error: msg}
	pageT.ExecuteTemplate(w, entryPoint(tmpl), td)
}

// entryPoint returns the root template name to execute for a given page file name.
func entryPoint(name string) string {
	if len(name) >= 5 && name[:5] == "user-" {
		return "dashboard"
	}
	if len(name) >= 7 && name[:7] == "dealer-" {
		return "dashboard"
	}
	if len(name) >= 6 && name[:6] == "admin-" {
		return "dashboard"
	}
	return "base"
}
