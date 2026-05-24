package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/glnarayanan/egassewa/gasmate/internal/model"
)

// ── Gas Cylinders ─────────────────────────────────────────────────────────────

func (h *Handler) AdminCylinders(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(),
		`SELECT id,unit_kg,type,price,description FROM gas_cylinders ORDER BY type,unit_kg`)
	var cyls []model.GasCylinder
	for rows != nil && rows.Next() {
		var c model.GasCylinder
		rows.Scan(&c.ID, &c.UnitKg, &c.Type, &c.Price, &c.Description)
		cyls = append(cyls, c)
	}
	rows.Close()
	h.render(w, r, "admin-cylinders.html", cyls)
}

func (h *Handler) AdminCreateCylinder(w http.ResponseWriter, r *http.Request) {
	unit, _ := strconv.ParseFloat(r.FormValue("unit_kg"), 64)
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	_, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO gas_cylinders(unit_kg,type,price,description) VALUES(?,?,?,?)`,
		unit, r.FormValue("type"), price, r.FormValue("description"))
	if err != nil {
		http.Error(w, "create failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	h.redirect(w, r, "/admin/cylinders")
}

func (h *Handler) AdminDeleteCylinder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.DB.ExecContext(r.Context(), `DELETE FROM gas_cylinders WHERE id=?`, id)
	w.WriteHeader(http.StatusOK)
}

// ── Accessories ───────────────────────────────────────────────────────────────

func (h *Handler) AdminAccessories(w http.ResponseWriter, r *http.Request) {
	accs, _ := h.allAccessories(r)
	h.render(w, r, "admin-accessories.html", accs)
}

func (h *Handler) AdminCreateAccessory(w http.ResponseWriter, r *http.Request) {
	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)
	_, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO accessories(name,price,description) VALUES(?,?,?)`,
		strings.ToUpper(strings.TrimSpace(r.FormValue("name"))), price, r.FormValue("description"))
	if err != nil {
		http.Error(w, "create failed", http.StatusBadRequest)
		return
	}
	h.redirect(w, r, "/admin/accessories")
}

func (h *Handler) AdminDeleteAccessory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.DB.ExecContext(r.Context(), `DELETE FROM accessories WHERE id=?`, id)
	w.WriteHeader(http.StatusOK)
}

// ── Dealers ───────────────────────────────────────────────────────────────────

func (h *Handler) AdminDealers(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT d.id, d.dealership_id, d.address, d.phone,
		       a.username, a.email, ci.name, s.name
		FROM dealers d
		JOIN accounts a ON a.id=d.account_id
		JOIN cities ci ON ci.id=d.city_id
		JOIN states s ON s.id=ci.state_id
		ORDER BY a.username
	`)
	var dealers []model.Dealer
	for rows != nil && rows.Next() {
		var d model.Dealer
		rows.Scan(&d.ID, &d.DealershipID, &d.Address, &d.Phone,
			&d.Username, &d.Email, &d.CityName, &d.StateName)
		dealers = append(dealers, d)
	}
	rows.Close()

	cities, _ := h.allCities(r)
	states, _ := h.allStates(r)
	h.render(w, r, "admin-dealers.html", map[string]any{
		"Dealers": dealers,
		"Cities":  cities,
		"States":  states,
	})
}

func (h *Handler) AdminCreateDealer(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	cityID := r.FormValue("city_id")
	address := r.FormValue("address")
	phone := r.FormValue("phone")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hash failed", http.StatusInternalServerError)
		return
	}

	var dealerCount int
	h.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM dealers`).Scan(&dealerCount)
	dealershipID := fmt.Sprintf("DLR%05d", dealerCount+1)

	tx, _ := h.DB.BeginTx(r.Context(), nil)
	defer tx.Rollback()

	var accID int64
	tx.QueryRowContext(r.Context(),
		`INSERT INTO accounts(username,email,password_hash,role,unique_number) VALUES(?,?,?,'dealer',?) RETURNING id`,
		username, email, string(hash), dealershipID).Scan(&accID)

	tx.ExecContext(r.Context(),
		`INSERT INTO dealers(account_id,dealership_id,city_id,address,phone) VALUES(?,?,?,?,?)`,
		accID, dealershipID, cityID, address, phone)

	tx.Commit()
	h.redirect(w, r, "/admin/dealers")
}

// ── Customers ─────────────────────────────────────────────────────────────────

func (h *Handler) AdminCustomers(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT c.id, c.gas_connection_number, c.address, c.phone,
		       a.username, a.email, ci.name, d_acc.username
		FROM customers c
		JOIN accounts a ON a.id=c.account_id
		JOIN cities ci ON ci.id=c.city_id
		JOIN dealers d ON d.id=c.dealer_id
		JOIN accounts d_acc ON d_acc.id=d.account_id
		ORDER BY a.username
	`)
	var custs []model.Customer
	for rows != nil && rows.Next() {
		var c model.Customer
		rows.Scan(&c.ID, &c.GasConnectionNumber, &c.Address, &c.Phone,
			&c.Username, &c.Email, &c.CityName, &c.DealerName)
		custs = append(custs, c)
	}
	rows.Close()
	h.render(w, r, "admin-customers.html", custs)
}

// ── Cities ────────────────────────────────────────────────────────────────────

func (h *Handler) AdminCities(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT ci.id, ci.name, s.name
		FROM cities ci JOIN states s ON s.id=ci.state_id
		ORDER BY s.name, ci.name
	`)
	type cityRow struct {
		ID        int64
		Name      string
		StateName string
	}
	var cities []cityRow
	for rows != nil && rows.Next() {
		var c cityRow
		rows.Scan(&c.ID, &c.Name, &c.StateName)
		cities = append(cities, c)
	}
	rows.Close()
	states, _ := h.allStates(r)
	h.render(w, r, "admin-cities.html", map[string]any{
		"Cities": cities,
		"States": states,
	})
}

func (h *Handler) AdminCreateCity(w http.ResponseWriter, r *http.Request) {
	_, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO cities(name,state_id) VALUES(?,?)`,
		strings.ToUpper(strings.TrimSpace(r.FormValue("name"))), r.FormValue("state_id"))
	if err != nil {
		http.Error(w, "create failed", http.StatusBadRequest)
		return
	}
	h.redirect(w, r, "/admin/cities")
}

// ── Admin requests ─────────────────────────────────────────────────────────────

func (h *Handler) AdminRequests(w http.ResponseWriter, r *http.Request) {
	// New connection requests
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT ncr.id, ncr.name, ncr.address, ncr.phone, ncr.id_proof_type,
		       ncr.id_proof_number, ncr.requested_at, ncr.status, ci.name
		FROM new_connection_requests ncr
		JOIN cities ci ON ci.id=ncr.city_id
		ORDER BY ncr.requested_at DESC
	`)
	var ncrs []model.NewConnectionRequest
	for rows != nil && rows.Next() {
		var n model.NewConnectionRequest
		rows.Scan(&n.ID, &n.Name, &n.Address, &n.Phone, &n.IDProofType,
			&n.IDProofNumber, &n.RequestedAt, &n.Status, &n.CityName)
		ncrs = append(ncrs, n)
	}
	rows.Close()

	// End service requests
	rows2, _ := h.DB.QueryContext(r.Context(), `
		SELECT es.id, a.username, es.reason, es.requested_at, es.status
		FROM end_services es JOIN accounts a ON a.id=es.account_id
		ORDER BY es.requested_at DESC
	`)
	var ess []model.EndService
	for rows2 != nil && rows2.Next() {
		var e model.EndService
		rows2.Scan(&e.ID, &e.Username, &e.Reason, &e.RequestedAt, &e.Status)
		ess = append(ess, e)
	}
	rows2.Close()

	// Location change requests
	rows3, _ := h.DB.QueryContext(r.Context(), `
		SELECT cl.id, a.username, ci.name, d_acc.username, cl.requested_at, cl.status
		FROM change_locations cl
		JOIN accounts a ON a.id=cl.account_id
		JOIN cities ci ON ci.id=cl.new_city_id
		JOIN dealers d ON d.id=cl.new_dealer_id
		JOIN accounts d_acc ON d_acc.id=d.account_id
		ORDER BY cl.requested_at DESC
	`)
	var cls []model.ChangeLocation
	for rows3 != nil && rows3.Next() {
		var c model.ChangeLocation
		rows3.Scan(&c.ID, &c.Username, &c.CityName, &c.DealerName, &c.RequestedAt, &c.Status)
		cls = append(cls, c)
	}
	rows3.Close()

	h.render(w, r, "admin-requests.html", map[string]any{
		"NewConnections":  ncrs,
		"EndServices":     ess,
		"ChangeLocations": cls,
	})
}

func (h *Handler) AdminUpdateRequestStatus(w http.ResponseWriter, r *http.Request) {
	table := r.PathValue("table")
	id := r.PathValue("id")
	status := r.FormValue("status")

	validTables := map[string]bool{
		"new_connection_requests": true,
		"end_services":            true,
		"change_locations":        true,
	}
	if !validTables[table] {
		http.Error(w, "invalid table", http.StatusBadRequest)
		return
	}

	h.DB.ExecContext(r.Context(),
		fmt.Sprintf(`UPDATE %s SET status=? WHERE id=?`, table), status, id)
	h.redirect(w, r, "/admin/requests")
}
