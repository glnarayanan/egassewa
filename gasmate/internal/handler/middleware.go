package handler

import (
	"context"
	"net/http"

	"github.com/glnarayanan/egassewa/gasmate/internal/auth"
)

type contextKey string

const sessionKey contextKey = "session"

func withSession(r *http.Request, s *auth.Session) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), sessionKey, s))
}

func sessionFrom(r *http.Request) *auth.Session {
	s, _ := r.Context().Value(sessionKey).(*auth.Session)
	return s
}

func RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := auth.GetSession(r)
		if err != nil || s.Role != role {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, withSession(r, s))
	}
}

func LoadSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, _ := auth.GetSession(r)
		if s != nil {
			r = withSession(r, s)
		}
		next(w, r)
	}
}
