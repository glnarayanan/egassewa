package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/glnarayanan/egassewa/gasmate/internal/auth"
	"github.com/glnarayanan/egassewa/gasmate/internal/model"
)

func (h *Handler) LoginGet(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "login.html", nil)
}

func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	var acc model.Account
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT id, username, email, password_hash, role FROM accounts WHERE username = ?`,
		username).Scan(&acc.ID, &acc.Username, &acc.Email, &acc.PasswordHash, &acc.Role)
	if err == sql.ErrNoRows {
		h.renderError(w, r, "login.html", "Invalid username or password")
		return
	}
	if err != nil {
		h.renderError(w, r, "login.html", "Login failed")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password)); err != nil {
		h.renderError(w, r, "login.html", "Invalid username or password")
		return
	}

	if err := auth.SetSession(w, auth.Session{
		UserID:   acc.ID,
		Username: acc.Username,
		Email:    acc.Email,
		Role:     acc.Role,
	}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	switch acc.Role {
	case "dealer":
		h.redirect(w, r, "/dealer/orders")
	case "admin":
		h.redirect(w, r, "/admin/cylinders")
	default:
		h.redirect(w, r, "/user/book")
	}
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w)
	h.redirect(w, r, "/")
}

func (h *Handler) SignupGet(w http.ResponseWriter, r *http.Request) {
	cities, _ := h.allCities(r)
	dealers, _ := h.allDealers(r)
	states, _ := h.allStates(r)
	h.render(w, r, "signup.html", map[string]any{
		"Cities":  cities,
		"Dealers": dealers,
		"States":  states,
	})
}

func (h *Handler) SignupPost(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	phone := strings.TrimSpace(r.FormValue("phone"))
	address := strings.TrimSpace(r.FormValue("address"))
	cityID := r.FormValue("city_id")
	dealerID := r.FormValue("dealer_id")

	if username == "" || email == "" || password == "" {
		h.renderError(w, r, "signup.html", "All fields are required")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.renderError(w, r, "signup.html", "Registration failed")
		return
	}

	// Generate gas connection number
	var count int
	h.DB.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM customers").Scan(&count)
	connNumber := fmt.Sprintf("GAS%06d", count+1)

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		h.renderError(w, r, "signup.html", "Registration failed")
		return
	}
	defer tx.Rollback()

	var accID int64
	err = tx.QueryRowContext(r.Context(),
		`INSERT INTO accounts(username,email,password_hash,role,unique_number) VALUES(?,?,?,'customer',?) RETURNING id`,
		username, email, string(hash), connNumber).Scan(&accID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			h.renderError(w, r, "signup.html", "Username or email already taken")
			return
		}
		h.renderError(w, r, "signup.html", "Registration failed")
		return
	}

	_, err = tx.ExecContext(r.Context(),
		`INSERT INTO customers(account_id,gas_connection_number,dealer_id,city_id,address,phone)
		 VALUES(?,?,?,?,?,?)`,
		accID, connNumber, dealerID, cityID, address, phone)
	if err != nil {
		h.renderError(w, r, "signup.html", "Registration failed: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		h.renderError(w, r, "signup.html", "Registration failed")
		return
	}

	if err := auth.SetSession(w, auth.Session{
		UserID:   accID,
		Username: username,
		Email:    email,
		Role:     "customer",
	}); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}

	h.redirect(w, r, "/user/book")
}

