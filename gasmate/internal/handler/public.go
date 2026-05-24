package handler

import (
	"net/http"

	"github.com/glnarayanan/egassewa/gasmate/internal/model"
)

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	rows, _ := h.DB.QueryContext(r.Context(), `SELECT id,unit_kg,type,price,description FROM gas_cylinders ORDER BY unit_kg`)
	var cylinders []model.GasCylinder
	for rows != nil && rows.Next() {
		var c model.GasCylinder
		rows.Scan(&c.ID, &c.UnitKg, &c.Type, &c.Price, &c.Description)
		cylinders = append(cylinders, c)
	}
	rows.Close()

	accs, _ := h.allAccessories(r)
	h.render(w, r, "home.html", map[string]any{
		"Cylinders":   cylinders,
		"Accessories": accs,
	})
}

func (h *Handler) Products(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(),
		`SELECT id,unit_kg,type,price,description FROM gas_cylinders ORDER BY type,unit_kg`)
	var cylinders []model.GasCylinder
	for rows != nil && rows.Next() {
		var c model.GasCylinder
		rows.Scan(&c.ID, &c.UnitKg, &c.Type, &c.Price, &c.Description)
		cylinders = append(cylinders, c)
	}
	rows.Close()
	h.render(w, r, "products.html", cylinders)
}

func (h *Handler) AccessoriesPage(w http.ResponseWriter, r *http.Request) {
	accs, _ := h.allAccessories(r)
	h.render(w, r, "accessories.html", accs)
}

func (h *Handler) SearchDealers(w http.ResponseWriter, r *http.Request) {
	states, _ := h.allStates(r)
	h.render(w, r, "search.html", map[string]any{"States": states})
}

func (h *Handler) SearchDealersResults(w http.ResponseWriter, r *http.Request) {
	stateID := r.URL.Query().Get("state_id")
	cityID := r.URL.Query().Get("city_id")

	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT d.id, d.dealership_id, d.address, d.phone,
		       a.username, a.email, ci.name, s.name
		FROM dealers d
		JOIN accounts a ON a.id=d.account_id
		JOIN cities ci ON ci.id=d.city_id
		JOIN states s ON s.id=ci.state_id
		WHERE ci.state_id=? AND d.city_id=?
	`, stateID, cityID)
	var dealers []model.Dealer
	for rows != nil && rows.Next() {
		var d model.Dealer
		rows.Scan(&d.ID, &d.DealershipID, &d.Address, &d.Phone,
			&d.Username, &d.Email, &d.CityName, &d.StateName)
		dealers = append(dealers, d)
	}
	rows.Close()
	h.renderPartial(w, "dealer-results.html", dealers)
}

func (h *Handler) CitiesForState(w http.ResponseWriter, r *http.Request) {
	stateID := r.URL.Query().Get("state_id")
	rows, _ := h.DB.QueryContext(r.Context(),
		`SELECT id, name FROM cities WHERE state_id=? ORDER BY name`, stateID)
	var cities []model.City
	for rows != nil && rows.Next() {
		var c model.City
		rows.Scan(&c.ID, &c.Name)
		cities = append(cities, c)
	}
	rows.Close()
	h.renderPartial(w, "city-options.html", cities)
}

func (h *Handler) DealersForCity(w http.ResponseWriter, r *http.Request) {
	cityID := r.URL.Query().Get("city_id")
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT d.id, d.dealership_id, a.username
		FROM dealers d JOIN accounts a ON a.id=d.account_id
		WHERE d.city_id=? ORDER BY a.username
	`, cityID)
	var dealers []model.Dealer
	for rows != nil && rows.Next() {
		var d model.Dealer
		rows.Scan(&d.ID, &d.DealershipID, &d.Username)
		dealers = append(dealers, d)
	}
	rows.Close()
	h.renderPartial(w, "dealer-options.html", dealers)
}

func (h *Handler) Apply(w http.ResponseWriter, r *http.Request) {
	states, _ := h.allStates(r)
	h.render(w, r, "apply.html", map[string]any{"States": states})
}

func (h *Handler) ApplyPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var accountID any
	if sess != nil {
		accountID = sess.UserID
	}
	_, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO new_connection_requests
		(account_id,name,address,phone,city_id,id_proof_type,id_proof_number)
		VALUES(?,?,?,?,?,?,?)`,
		accountID,
		r.FormValue("name"), r.FormValue("address"), r.FormValue("phone"),
		r.FormValue("city_id"), r.FormValue("id_proof_type"), r.FormValue("id_proof_number"),
	)
	if err != nil {
		h.render(w, r, "apply.html", map[string]any{"Error": "Submission failed: " + err.Error()})
		return
	}
	h.render(w, r, "apply-success.html", nil)
}

func (h *Handler) FAQ(w http.ResponseWriter, r *http.Request) { h.render(w, r, "faq.html", nil) }
func (h *Handler) About(w http.ResponseWriter, r *http.Request) { h.render(w, r, "about.html", nil) }
func (h *Handler) Contact(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "contact.html", nil)
}
func (h *Handler) Rules(w http.ResponseWriter, r *http.Request) { h.render(w, r, "rules.html", nil) }

// helpers

func (h *Handler) allStates(r *http.Request) ([]model.State, error) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id,name FROM states ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.State
	for rows.Next() {
		var s model.State
		rows.Scan(&s.ID, &s.Name)
		out = append(out, s)
	}
	return out, nil
}

func (h *Handler) allCities(r *http.Request) ([]model.City, error) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id,name FROM cities ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.City
	for rows.Next() {
		var c model.City
		rows.Scan(&c.ID, &c.Name)
		out = append(out, c)
	}
	return out, nil
}

func (h *Handler) allDealers(r *http.Request) ([]model.Dealer, error) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT d.id, d.dealership_id, a.username, ci.name
		FROM dealers d
		JOIN accounts a ON a.id=d.account_id
		JOIN cities ci ON ci.id=d.city_id
		ORDER BY a.username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Dealer
	for rows.Next() {
		var d model.Dealer
		rows.Scan(&d.ID, &d.DealershipID, &d.Username, &d.CityName)
		out = append(out, d)
	}
	return out, nil
}

func (h *Handler) allAccessories(r *http.Request) ([]model.Accessory, error) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id,name,price,description FROM accessories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Accessory
	for rows.Next() {
		var a model.Accessory
		rows.Scan(&a.ID, &a.Name, &a.Price, &a.Description)
		out = append(out, a)
	}
	return out, nil
}
