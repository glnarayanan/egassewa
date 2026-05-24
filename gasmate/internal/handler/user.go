package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/glnarayanan/egassewa/gasmate/internal/model"
)

func (h *Handler) UserBook(w http.ResponseWriter, r *http.Request) {
	rows, _ := h.DB.QueryContext(r.Context(),
		`SELECT id,unit_kg,type,price,description FROM gas_cylinders ORDER BY type,unit_kg`)
	var cylinders []model.GasCylinder
	for rows != nil && rows.Next() {
		var c model.GasCylinder
		rows.Scan(&c.ID, &c.UnitKg, &c.Type, &c.Price, &c.Description)
		cylinders = append(cylinders, c)
	}
	rows.Close()
	h.render(w, r, "user-book.html", cylinders)
}

func (h *Handler) UserBookPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	cylinderID := r.FormValue("cylinder_id")

	// Get customer's dealer
	var dealerID int64
	var gasConnNum string
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT c.dealer_id, c.gas_connection_number FROM customers c
		 JOIN accounts a ON a.id=c.account_id WHERE a.id=?`, sess.UserID).
		Scan(&dealerID, &gasConnNum)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.renderPartial(w, "book-result.html", map[string]any{"Error": "Customer record not found"})
		return
	}

	// Enforce 20-day cooldown on same cylinder type
	var blocked int
	h.DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM order_cylinders
		WHERE account_id=? AND cylinder_id=?
		AND next_order_date > datetime('now')
		AND status != 'CANCELLED'
	`, sess.UserID, cylinderID).Scan(&blocked)

	if blocked > 0 {
		var nextDate string
		h.DB.QueryRowContext(r.Context(), `
			SELECT next_order_date FROM order_cylinders
			WHERE account_id=? AND cylinder_id=? AND status != 'CANCELLED'
			ORDER BY ordered_at DESC LIMIT 1
		`, sess.UserID, cylinderID).Scan(&nextDate)
		w.WriteHeader(http.StatusConflict)
		h.renderPartial(w, "book-result.html", map[string]any{
			"Error": fmt.Sprintf("You can book again after %s", nextDate[:10]),
		})
		return
	}

	// Get next order number
	var maxNum int64
	h.DB.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(order_number),10000) FROM order_cylinders`).Scan(&maxNum)

	nextOrder := time.Now().AddDate(0, 0, 20)
	_, err = h.DB.ExecContext(r.Context(), `
		INSERT INTO order_cylinders
		(order_number,account_id,cylinder_id,dealer_id,order_count,next_order_date,status)
		VALUES(?,?,?,?,1,?,'WAITING')
	`, maxNum+1, sess.UserID, cylinderID, dealerID, nextOrder)
	if err != nil {
		h.renderPartial(w, "book-result.html", map[string]any{"Error": "Booking failed"})
		return
	}

	// Send confirmation email (non-blocking)
	go h.Email.Send(sess.Email, "GasMate — Order Placed",
		fmt.Sprintf("Hi %s,\n\nYour cylinder order has been placed successfully.\nOrder #%d\nStatus: WAITING\n\nNext order available: %s\n\nThank you,\nGasMate Team",
			sess.Username, maxNum+1, nextOrder.Format("2006-01-02")))

	h.renderPartial(w, "book-result.html", map[string]any{"Success": true, "OrderNum": maxNum + 1})
}

func (h *Handler) UserOrders(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT oc.id, oc.order_number, oc.order_count, oc.ordered_at,
		       oc.next_order_date, oc.status, gc.unit_kg, gc.type
		FROM order_cylinders oc
		JOIN gas_cylinders gc ON gc.id=oc.cylinder_id
		WHERE oc.account_id=?
		ORDER BY oc.ordered_at DESC
	`, sess.UserID)
	var orders []model.OrderCylinder
	for rows != nil && rows.Next() {
		var o model.OrderCylinder
		rows.Scan(&o.ID, &o.OrderNumber, &o.OrderCount, &o.OrderedAt,
			&o.NextOrderDate, &o.Status, &o.CylinderUnit, &o.CylinderType)
		orders = append(orders, o)
	}
	rows.Close()

	rows2, _ := h.DB.QueryContext(r.Context(), `
		SELECT oa.id, oa.order_count, oa.ordered_at, oa.status, a.name
		FROM order_accessories oa
		JOIN accessories a ON a.id=oa.accessory_id
		WHERE oa.account_id=?
		ORDER BY oa.ordered_at DESC
	`, sess.UserID)
	var accOrders []model.OrderAccessory
	for rows2 != nil && rows2.Next() {
		var o model.OrderAccessory
		rows2.Scan(&o.ID, &o.OrderCount, &o.OrderedAt, &o.Status, &o.ItemName)
		accOrders = append(accOrders, o)
	}
	rows2.Close()

	h.render(w, r, "user-orders.html", map[string]any{
		"CylinderOrders":  orders,
		"AccessoryOrders": accOrders,
	})
}

func (h *Handler) UserAccessoriesPage(w http.ResponseWriter, r *http.Request) {
	accs, _ := h.allAccessories(r)
	h.render(w, r, "user-accessories.html", accs)
}

func (h *Handler) UserAccessoriesPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	accessoryID := r.FormValue("accessory_id")

	var dealerID int64
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT dealer_id FROM customers WHERE account_id=?`, sess.UserID).Scan(&dealerID)
	if err != nil {
		h.renderPartial(w, "acc-order-result.html", map[string]any{"Error": "Customer record not found"})
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		INSERT INTO order_accessories(account_id,accessory_id,dealer_id,order_count,status)
		VALUES(?,?,?,1,'PENDING')
	`, sess.UserID, accessoryID, dealerID)
	if err != nil {
		h.renderPartial(w, "acc-order-result.html", map[string]any{"Error": "Order failed"})
		return
	}

	var itemName string
	h.DB.QueryRowContext(r.Context(), `SELECT name FROM accessories WHERE id=?`, accessoryID).Scan(&itemName)

	go h.Email.Send(sess.Email, "GasMate — Accessory Order Placed",
		fmt.Sprintf("Hi %s,\n\nYour order for %s has been placed.\nStatus: PENDING\n\nThank you,\nGasMate Team",
			sess.Username, itemName))

	h.renderPartial(w, "acc-order-result.html", map[string]any{"Success": true, "ItemName": itemName})
}

func (h *Handler) UserProfile(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var cust model.Customer
	h.DB.QueryRowContext(r.Context(), `
		SELECT c.id, c.gas_connection_number, c.address, c.phone,
		       a.username, a.email, ci.name
		FROM customers c
		JOIN accounts a ON a.id=c.account_id
		JOIN cities ci ON ci.id=c.city_id
		WHERE c.account_id=?
	`, sess.UserID).Scan(&cust.ID, &cust.GasConnectionNumber, &cust.Address, &cust.Phone,
		&cust.Username, &cust.Email, &cust.CityName)
	h.render(w, r, "user-profile.html", cust)
}

func (h *Handler) UserProfilePost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	h.DB.ExecContext(r.Context(),
		`UPDATE customers SET address=?, phone=? WHERE account_id=?`,
		r.FormValue("address"), r.FormValue("phone"), sess.UserID)
	h.redirect(w, r, "/user/profile")
}

func (h *Handler) UserNewConnection(w http.ResponseWriter, r *http.Request) {
	states, _ := h.allStates(r)
	h.render(w, r, "user-new-connection.html", map[string]any{"States": states})
}

func (h *Handler) UserNewConnectionPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	_, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO new_connection_requests
		(account_id,name,address,phone,city_id,id_proof_type,id_proof_number)
		VALUES(?,?,?,?,?,?,?)`,
		sess.UserID, r.FormValue("name"), r.FormValue("address"), r.FormValue("phone"),
		r.FormValue("city_id"), r.FormValue("id_proof_type"), r.FormValue("id_proof_number"),
	)
	if err != nil {
		h.render(w, r, "user-new-connection.html", map[string]any{"Error": "Submission failed"})
		return
	}
	h.render(w, r, "user-new-connection.html", map[string]any{"Success": true})
}

func (h *Handler) UserChangeLocation(w http.ResponseWriter, r *http.Request) {
	states, _ := h.allStates(r)
	h.render(w, r, "user-change-location.html", map[string]any{"States": states})
}

func (h *Handler) UserChangeLocationPost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	_, err := h.DB.ExecContext(r.Context(), `
		INSERT INTO change_locations(account_id,new_city_id,new_dealer_id) VALUES(?,?,?)`,
		sess.UserID, r.FormValue("city_id"), r.FormValue("dealer_id"),
	)
	if err != nil {
		h.render(w, r, "user-change-location.html", map[string]any{"Error": "Submission failed"})
		return
	}
	h.render(w, r, "user-change-location.html", map[string]any{"Success": true})
}

func (h *Handler) UserEndService(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, "user-end-service.html", nil)
}

func (h *Handler) UserEndServicePost(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)

	// Check if already has pending/active end-service request
	var count int
	h.DB.QueryRowContext(r.Context(),
		`SELECT COUNT(*) FROM end_services WHERE account_id=? AND status='PENDING'`,
		sess.UserID).Scan(&count)

	if count > 0 {
		h.render(w, r, "user-end-service.html", map[string]any{"Error": "You already have a pending end-service request"})
		return
	}

	reason := r.FormValue("reason")
	if _, err := h.DB.ExecContext(r.Context(),
		`INSERT INTO end_services(account_id,reason) VALUES(?,?)`,
		sess.UserID, reason); err != nil {
		h.render(w, r, "user-end-service.html", map[string]any{"Error": "Submission failed"})
		return
	}

	h.render(w, r, "user-end-service.html", map[string]any{"Success": true})
}
