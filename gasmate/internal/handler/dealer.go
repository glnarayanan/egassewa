package handler

import (
	"fmt"
	"net/http"

	"github.com/glnarayanan/egassewa/gasmate/internal/model"
)

func (h *Handler) DealerOrders(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)

	var dealerID int64
	h.DB.QueryRowContext(r.Context(),
		`SELECT id FROM dealers WHERE account_id=?`, sess.UserID).Scan(&dealerID)

	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT oc.id, oc.order_number, oc.order_count, oc.ordered_at,
		       oc.next_order_date, oc.status, gc.unit_kg, gc.type, a.username, a.email
		FROM order_cylinders oc
		JOIN gas_cylinders gc ON gc.id=oc.cylinder_id
		JOIN accounts a ON a.id=oc.account_id
		WHERE oc.dealer_id=? AND oc.status NOT IN ('DISPATCHED','CANCELLED')
		ORDER BY oc.ordered_at DESC
	`, dealerID)
	var orders []model.OrderCylinder
	for rows != nil && rows.Next() {
		var o model.OrderCylinder
		rows.Scan(&o.ID, &o.OrderNumber, &o.OrderCount, &o.OrderedAt,
			&o.NextOrderDate, &o.Status, &o.CylinderUnit, &o.CylinderType, &o.Username, &o.Email)
		orders = append(orders, o)
	}
	rows.Close()

	h.render(w, r, "dealer-orders.html", orders)
}

func (h *Handler) DealerConfirmOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	h.updateCylinderOrderStatus(w, r, orderID, "CONFIRMED")
}

func (h *Handler) DealerDispatchOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	h.updateCylinderOrderStatus(w, r, orderID, "DISPATCHED")
}

func (h *Handler) DealerCancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	h.updateCylinderOrderStatus(w, r, orderID, "CANCELLED")
}

func (h *Handler) updateCylinderOrderStatus(w http.ResponseWriter, r *http.Request, orderID, status string) {
	var email, username string
	var orderNum int64
	h.DB.QueryRowContext(r.Context(), `
		SELECT a.email, a.username, oc.order_number
		FROM order_cylinders oc JOIN accounts a ON a.id=oc.account_id
		WHERE oc.id=?
	`, orderID).Scan(&email, &username, &orderNum)

	h.DB.ExecContext(r.Context(),
		`UPDATE order_cylinders SET status=? WHERE id=?`, status, orderID)

	go h.Email.Send(email, "GasMate — Order Update",
		fmt.Sprintf("Hi %s,\n\nYour cylinder order #%d status has been updated to: %s\n\nThank you,\nGasMate Team",
			username, orderNum, status))

	// Return updated row partial
	var o model.OrderCylinder
	h.DB.QueryRowContext(r.Context(), `
		SELECT oc.id, oc.order_number, oc.order_count, oc.ordered_at,
		       oc.next_order_date, oc.status, gc.unit_kg, gc.type, a.username, a.email
		FROM order_cylinders oc
		JOIN gas_cylinders gc ON gc.id=oc.cylinder_id
		JOIN accounts a ON a.id=oc.account_id
		WHERE oc.id=?
	`, orderID).Scan(&o.ID, &o.OrderNumber, &o.OrderCount, &o.OrderedAt,
		&o.NextOrderDate, &o.Status, &o.CylinderUnit, &o.CylinderType, &o.Username, &o.Email)

	h.renderPartial(w, "dealer-order-row.html", o)
}

func (h *Handler) DealerAccessories(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var dealerID int64
	h.DB.QueryRowContext(r.Context(),
		`SELECT id FROM dealers WHERE account_id=?`, sess.UserID).Scan(&dealerID)

	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT oa.id, oa.order_count, oa.ordered_at, oa.status,
		       acc.name, a.username, a.email
		FROM order_accessories oa
		JOIN accessories acc ON acc.id=oa.accessory_id
		JOIN accounts a ON a.id=oa.account_id
		WHERE oa.dealer_id=? AND oa.status NOT IN ('DISPATCHED','CANCELLED')
		ORDER BY oa.ordered_at DESC
	`, dealerID)
	var orders []model.OrderAccessory
	for rows != nil && rows.Next() {
		var o model.OrderAccessory
		rows.Scan(&o.ID, &o.OrderCount, &o.OrderedAt, &o.Status, &o.ItemName, &o.Username, &o.Email)
		orders = append(orders, o)
	}
	rows.Close()

	h.render(w, r, "dealer-accessories.html", orders)
}

func (h *Handler) DealerConfirmAccessory(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	h.updateAccessoryOrderStatus(w, r, orderID, "CONFIRMED")
}

func (h *Handler) DealerDispatchAccessory(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	h.updateAccessoryOrderStatus(w, r, orderID, "DISPATCHED")
}

func (h *Handler) updateAccessoryOrderStatus(w http.ResponseWriter, r *http.Request, orderID, status string) {
	var email, username, itemName string
	h.DB.QueryRowContext(r.Context(), `
		SELECT a.email, a.username, acc.name
		FROM order_accessories oa
		JOIN accounts a ON a.id=oa.account_id
		JOIN accessories acc ON acc.id=oa.accessory_id
		WHERE oa.id=?
	`, orderID).Scan(&email, &username, &itemName)

	h.DB.ExecContext(r.Context(),
		`UPDATE order_accessories SET status=? WHERE id=?`, status, orderID)

	go h.Email.Send(email, "GasMate — Accessory Order Update",
		fmt.Sprintf("Hi %s,\n\nYour order for %s has been updated to: %s\n\nThank you,\nGasMate Team",
			username, itemName, status))

	var o model.OrderAccessory
	h.DB.QueryRowContext(r.Context(), `
		SELECT oa.id, oa.order_count, oa.ordered_at, oa.status,
		       acc.name, a.username, a.email
		FROM order_accessories oa
		JOIN accessories acc ON acc.id=oa.accessory_id
		JOIN accounts a ON a.id=oa.account_id
		WHERE oa.id=?
	`, orderID).Scan(&o.ID, &o.OrderCount, &o.OrderedAt, &o.Status, &o.ItemName, &o.Username, &o.Email)

	h.renderPartial(w, "dealer-acc-row.html", o)
}

func (h *Handler) DealerCustomers(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var dealerID int64
	h.DB.QueryRowContext(r.Context(),
		`SELECT id FROM dealers WHERE account_id=?`, sess.UserID).Scan(&dealerID)

	rows, _ := h.DB.QueryContext(r.Context(), `
		SELECT c.id, c.gas_connection_number, c.address, c.phone,
		       a.username, a.email, ci.name
		FROM customers c
		JOIN accounts a ON a.id=c.account_id
		JOIN cities ci ON ci.id=c.city_id
		WHERE c.dealer_id=?
		ORDER BY a.username
	`, dealerID)
	var custs []model.Customer
	for rows != nil && rows.Next() {
		var c model.Customer
		rows.Scan(&c.ID, &c.GasConnectionNumber, &c.Address, &c.Phone,
			&c.Username, &c.Email, &c.CityName)
		custs = append(custs, c)
	}
	rows.Close()

	h.render(w, r, "dealer-customers.html", custs)
}
