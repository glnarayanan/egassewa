package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	appdb "github.com/glnarayanan/egassewa/gasmate/internal/db"
	"github.com/glnarayanan/egassewa/gasmate/internal/email"
	"github.com/glnarayanan/egassewa/gasmate/internal/handler"
)

//go:embed static
var staticFS embed.FS

//go:embed templates
var templatesFS embed.FS

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "gasmate.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := appdb.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	pages, partials := buildTemplateCache(templatesFS)

	h := &handler.Handler{
		DB:       db,
		Pages:    pages,
		Partials: partials,
		Email:    email.FromEnv(),
	}

	mux := http.NewServeMux()

	// Static assets
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	// Public routes
	mux.HandleFunc("GET /{$}", handler.LoadSession(h.Home))
	mux.HandleFunc("GET /products", handler.LoadSession(h.Products))
	mux.HandleFunc("GET /accessories", handler.LoadSession(h.AccessoriesPage))
	mux.HandleFunc("GET /search-dealers", handler.LoadSession(h.SearchDealers))
	mux.HandleFunc("GET /search-dealers/results", handler.LoadSession(h.SearchDealersResults))
	mux.HandleFunc("GET /cities", h.CitiesForState)
	mux.HandleFunc("GET /dealers", h.DealersForCity)
	mux.HandleFunc("GET /apply", handler.LoadSession(h.Apply))
	mux.HandleFunc("POST /apply", handler.LoadSession(h.ApplyPost))
	mux.HandleFunc("GET /faq", handler.LoadSession(h.FAQ))
	mux.HandleFunc("GET /about", handler.LoadSession(h.About))
	mux.HandleFunc("GET /contact", handler.LoadSession(h.Contact))
	mux.HandleFunc("GET /rules", handler.LoadSession(h.Rules))

	// Auth routes
	mux.HandleFunc("GET /login", handler.LoadSession(h.LoginGet))
	mux.HandleFunc("POST /login", h.LoginPost)
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("GET /signup", handler.LoadSession(h.SignupGet))
	mux.HandleFunc("POST /signup", h.SignupPost)

	// Customer routes
	mux.HandleFunc("GET /user/book", handler.RequireRole("customer", h.UserBook))
	mux.HandleFunc("POST /user/book", handler.RequireRole("customer", h.UserBookPost))
	mux.HandleFunc("GET /user/orders", handler.RequireRole("customer", h.UserOrders))
	mux.HandleFunc("GET /user/accessories", handler.RequireRole("customer", h.UserAccessoriesPage))
	mux.HandleFunc("POST /user/accessories", handler.RequireRole("customer", h.UserAccessoriesPost))
	mux.HandleFunc("GET /user/profile", handler.RequireRole("customer", h.UserProfile))
	mux.HandleFunc("POST /user/profile", handler.RequireRole("customer", h.UserProfilePost))
	mux.HandleFunc("GET /user/new-connection", handler.RequireRole("customer", h.UserNewConnection))
	mux.HandleFunc("POST /user/new-connection", handler.RequireRole("customer", h.UserNewConnectionPost))
	mux.HandleFunc("GET /user/change-location", handler.RequireRole("customer", h.UserChangeLocation))
	mux.HandleFunc("POST /user/change-location", handler.RequireRole("customer", h.UserChangeLocationPost))
	mux.HandleFunc("GET /user/end-service", handler.RequireRole("customer", h.UserEndService))
	mux.HandleFunc("POST /user/end-service", handler.RequireRole("customer", h.UserEndServicePost))

	// Dealer routes
	mux.HandleFunc("GET /dealer/orders", handler.RequireRole("dealer", h.DealerOrders))
	mux.HandleFunc("POST /dealer/orders/{id}/confirm", handler.RequireRole("dealer", h.DealerConfirmOrder))
	mux.HandleFunc("POST /dealer/orders/{id}/dispatch", handler.RequireRole("dealer", h.DealerDispatchOrder))
	mux.HandleFunc("POST /dealer/orders/{id}/cancel", handler.RequireRole("dealer", h.DealerCancelOrder))
	mux.HandleFunc("GET /dealer/accessories", handler.RequireRole("dealer", h.DealerAccessories))
	mux.HandleFunc("POST /dealer/accessories/{id}/confirm", handler.RequireRole("dealer", h.DealerConfirmAccessory))
	mux.HandleFunc("POST /dealer/accessories/{id}/dispatch", handler.RequireRole("dealer", h.DealerDispatchAccessory))
	mux.HandleFunc("GET /dealer/customers", handler.RequireRole("dealer", h.DealerCustomers))

	// Admin routes
	mux.HandleFunc("GET /admin/cylinders", handler.RequireRole("admin", h.AdminCylinders))
	mux.HandleFunc("POST /admin/cylinders", handler.RequireRole("admin", h.AdminCreateCylinder))
	mux.HandleFunc("DELETE /admin/cylinders/{id}", handler.RequireRole("admin", h.AdminDeleteCylinder))
	mux.HandleFunc("GET /admin/accessories", handler.RequireRole("admin", h.AdminAccessories))
	mux.HandleFunc("POST /admin/accessories", handler.RequireRole("admin", h.AdminCreateAccessory))
	mux.HandleFunc("DELETE /admin/accessories/{id}", handler.RequireRole("admin", h.AdminDeleteAccessory))
	mux.HandleFunc("GET /admin/dealers", handler.RequireRole("admin", h.AdminDealers))
	mux.HandleFunc("POST /admin/dealers", handler.RequireRole("admin", h.AdminCreateDealer))
	mux.HandleFunc("GET /admin/customers", handler.RequireRole("admin", h.AdminCustomers))
	mux.HandleFunc("GET /admin/cities", handler.RequireRole("admin", h.AdminCities))
	mux.HandleFunc("POST /admin/cities", handler.RequireRole("admin", h.AdminCreateCity))
	mux.HandleFunc("GET /admin/requests", handler.RequireRole("admin", h.AdminRequests))
	mux.HandleFunc("POST /admin/requests/{table}/{id}/status", handler.RequireRole("admin", h.AdminUpdateRequestStatus))

	log.Printf("GasMate listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// buildTemplateCache creates per-page template clones to avoid {{define}} conflicts,
// plus a shared partials template set for HTMX fragments.
func buildTemplateCache(fsys embed.FS) (map[string]*template.Template, *template.Template) {
	baseLayout := template.Must(template.ParseFS(fsys, "templates/layout/base.html"))
	dashLayout := template.Must(template.ParseFS(fsys, "templates/layout/dashboard.html"))

	// Partial templates (HTMX fragments — no layout needed)
	partialFiles := []string{
		"templates/public/city-options.html",
		"templates/public/dealer-options.html",
		"templates/public/dealer-results.html",
		"templates/dealer/dealer-order-row.html",
		"templates/dealer/dealer-acc-row.html",
		"templates/user/book-result.html",
		"templates/user/acc-order-result.html",
	}
	partials := template.New("")
	for _, f := range partialFiles {
		template.Must(partials.ParseFS(fsys, f))
	}

	// Pages: each gets a clone of the appropriate layout + its own defines
	type pageSpec struct {
		name   string
		layout *template.Template
		file   string
	}
	specs := []pageSpec{
		// Public pages (base layout)
		{"home.html", baseLayout, "templates/public/home.html"},
		{"products.html", baseLayout, "templates/public/products.html"},
		{"accessories.html", baseLayout, "templates/public/accessories.html"},
		{"search.html", baseLayout, "templates/public/search.html"},
		{"apply.html", baseLayout, "templates/public/apply.html"},
		{"apply-success.html", baseLayout, "templates/public/apply-success.html"},
		{"faq.html", baseLayout, "templates/public/faq.html"},
		{"about.html", baseLayout, "templates/public/about.html"},
		{"contact.html", baseLayout, "templates/public/contact.html"},
		{"rules.html", baseLayout, "templates/public/rules.html"},
		{"login.html", baseLayout, "templates/public/login.html"},
		{"signup.html", baseLayout, "templates/public/signup.html"},
		// Customer portal (dashboard layout)
		{"user-book.html", dashLayout, "templates/user/user-book.html"},
		{"user-orders.html", dashLayout, "templates/user/user-orders.html"},
		{"user-accessories.html", dashLayout, "templates/user/user-accessories.html"},
		{"user-profile.html", dashLayout, "templates/user/user-profile.html"},
		{"user-new-connection.html", dashLayout, "templates/user/user-new-connection.html"},
		{"user-change-location.html", dashLayout, "templates/user/user-change-location.html"},
		{"user-end-service.html", dashLayout, "templates/user/user-end-service.html"},
		// Dealer portal
		{"dealer-orders.html", dashLayout, "templates/dealer/dealer-orders.html"},
		{"dealer-accessories.html", dashLayout, "templates/dealer/dealer-accessories.html"},
		{"dealer-customers.html", dashLayout, "templates/dealer/dealer-customers.html"},
		// Admin portal
		{"admin-cylinders.html", dashLayout, "templates/admin/admin-cylinders.html"},
		{"admin-accessories.html", dashLayout, "templates/admin/admin-accessories.html"},
		{"admin-dealers.html", dashLayout, "templates/admin/admin-dealers.html"},
		{"admin-customers.html", dashLayout, "templates/admin/admin-customers.html"},
		{"admin-cities.html", dashLayout, "templates/admin/admin-cities.html"},
		{"admin-requests.html", dashLayout, "templates/admin/admin-requests.html"},
	}

	// Dealer pages need the partial row templates included so {{template "dealer-order-row.html"}} works
	dealerPartialFiles := []string{
		"templates/dealer/dealer-order-row.html",
		"templates/dealer/dealer-acc-row.html",
	}

	cache := make(map[string]*template.Template, len(specs))
	for _, s := range specs {
		clone := template.Must(s.layout.Clone())
		template.Must(clone.ParseFS(fsys, s.file))
		// Dealer pages reference row partials via {{template}}
		if s.name == "dealer-orders.html" || s.name == "dealer-accessories.html" {
			for _, pf := range dealerPartialFiles {
				template.Must(clone.ParseFS(fsys, pf))
			}
		}
		cache[s.name] = clone
	}

	return cache, partials
}
