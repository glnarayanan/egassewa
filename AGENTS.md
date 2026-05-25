# GasMate — Agent Context

This file is the authoritative project context for AI agents (Claude Code, Codex, Cursor, etc.) working in this repository. `CLAUDE.md` is a symlink to this file.

---

## Repository Layout

```
egassewa/
├── AGENTS.md          ← this file
├── CLAUDE.md          ← symlink → AGENTS.md
├── gasmate/           ← Go rewrite (active development)
│   ├── main.go
│   ├── go.mod / go.sum
│   ├── internal/
│   │   ├── auth/session.go
│   │   ├── db/db.go + schema.sql + seed.sql
│   │   ├── email/email.go
│   │   ├── handler/
│   │   │   ├── handler.go    ← Handler struct, render helpers
│   │   │   ├── middleware.go ← RequireRole, LoadSession
│   │   │   ├── auth.go       ← login, logout, signup
│   │   │   ├── public.go     ← public pages + HTMX partials
│   │   │   ├── user.go       ← customer portal
│   │   │   ├── dealer.go     ← dealer portal
│   │   │   └── admin.go      ← admin portal
│   │   └── model/models.go
│   ├── templates/
│   │   ├── layout/base.html + dashboard.html
│   │   ├── public/
│   │   ├── user/
│   │   ├── dealer/
│   │   └── admin/
│   └── static/css/ + static/js/
├── controllers/       ← original 2013 Java Play 1.x source (read-only reference)
├── models/
└── views/
```

The `gasmate/` subdirectory is the only thing under active development. The Java code under `controllers/`, `models/`, `views/` is historical reference only — do not modify it.

---

## What GasMate Does

LPG gas agency management system. Three user roles:

| Role | Entry after login | Created by |
|------|-------------------|------------|
| `admin` | `/admin/cylinders` | Manual DB insert |
| `dealer` | `/dealer/orders` | Admin via `/admin/dealers` |
| `customer` | `/user/book` | Self-registration at `/signup` |

**Order flows:**
- Cylinder: `WAITING → CONFIRMED → DISPATCHED` (also `→ CANCELLED`). 20-day cooldown between bookings of the same cylinder type.
- Accessory: `PENDING → CONFIRMED → DISPATCHED`.
- Email notification on every status transition (logs to stdout if `SMTP_HOST` is unset).

---

## Tech Stack

| Concern | Choice |
|---------|--------|
| HTTP server | `net/http` stdlib |
| Templates | `html/template` stdlib |
| Database | `database/sql` + `modernc.org/sqlite` (pure Go, no CGo) |
| Password hashing | `golang.org/x/crypto/bcrypt` |
| Sessions | HMAC-SHA256 signed cookies (`crypto/hmac` stdlib) |
| Email | `net/smtp` stdlib |
| Frontend interactivity | Self-contained ~80-line HTMX-compatible JS (`static/js/htmx.min.js`) — no CDN |
| Styling | Custom CSS — no framework |

**External dependencies: 2** (`modernc.org/sqlite`, `golang.org/x/crypto`). Everything else is stdlib.

---

## How to Run

```bash
cd egassewa/gasmate
go run .          # dev server on :8080
go build -o gasmate .   # single self-contained binary
```

Environment variables (all optional):

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Listen port |
| `DB_PATH` | `gasmate.db` | SQLite file path |
| `SMTP_HOST` | _(empty)_ | If unset, emails log to stdout |
| `SMTP_PORT` | `587` | |
| `SMTP_USER` | _(empty)_ | |
| `SMTP_PASS` | _(empty)_ | |
| `SMTP_FROM` | `gasmate@example.com` | |
| `SESSION_KEY` | _(random)_ | Hex-encoded 32-byte HMAC key. If unset, a random key is generated at startup (invalidates sessions on restart). Set this in production. |

First boot seeds the database with 5 states, 13 cities, 6 cylinder types, 9 accessories. No user accounts are seeded — create an admin manually (see `gasmate/README.md`).

---

## Architecture Decisions

### Template System
Go's `html/template` has a "last `{{define}}` wins" problem when all templates share a single `*template.Template`. This app avoids it by creating a **`template.Clone()`** of the layout for each page at startup — each page's `{{define}}` blocks exist in isolation. This happens in `buildTemplateCache()` in `main.go`.

- Public pages clone `templates/layout/base.html` → executes as `"base"`.
- Portal pages (user/dealer/admin) clone `templates/layout/dashboard.html` → executes as `"dashboard"`.
- HTMX partials share a separate `*template.Template` (no layout).
- `entryPoint(name string) string` in `handler.go` determines which root template to execute based on filename prefix.

### Sessions
`internal/auth/session.go`. Format: `base64(json) + "." + base64(hmac-sha256)`. Cookie is `HttpOnly`, `SameSite=Strict`. Key is currently generated randomly at startup — set `SESSION_KEY` env var (hex-encoded 32 bytes) for persistence.

### HTMX
`static/js/htmx.min.js` is a self-contained ~80-line implementation (no CDN). Attributes supported: `hx-get`, `hx-post`, `hx-delete`, `hx-target`, `hx-swap`, `hx-vals`, `hx-include`, `hx-trigger`, `hx-confirm`. Do not replace this with a CDN link — network policy may block it.

### SQLite Concurrency
`db.SetMaxOpenConns(1)` enforces single-writer semantics. WAL mode is enabled for read concurrency.

### Routing
Go 1.22+ `http.ServeMux` with method+pattern syntax (`"POST /dealer/orders/{id}/confirm"`). No router library. Path values via `r.PathValue("id")`.

---

## Database Schema (Key Tables)

```
accounts       (id, username, email, password_hash, role, unique_number, created_at)
customers      (id, account_id, gas_connection_number, dealer_id, city_id, address, phone)
dealers        (id, account_id, dealership_id, city_id, address, phone)
order_cylinders    (id, order_number, account_id, cylinder_id, dealer_id,
                    ordered_at, next_order_date, status)
order_accessories  (id, account_id, accessory_id, dealer_id, ordered_at, status)
new_connection_requests / change_locations / end_services  (request tables)
countries / states / cities  (geography)
gas_cylinders / accessories  (catalog)
```

Full DDL: `internal/db/schema.sql`. Seed data: `internal/db/seed.sql`. Both are embedded at build time via `//go:embed`.

---

## Known Issues / Technical Debt

These are real bugs and gaps. Fix them before any production deployment:

### Functional Gaps
1. **Location-change approval is a no-op.** `AdminUpdateRequestStatus` sets `change_locations.status = 'APPROVED'` but never updates `customers.dealer_id` / `customers.city_id`. The customer's assignment never actually changes. Fix: add a follow-up `UPDATE customers SET dealer_id=?, city_id=? WHERE account_id=?` inside a transaction when the status becomes `APPROVED`.

2. **End-service approval doesn't disable the account.** Approving an end-service request leaves the account fully active. Fix: either add a `deactivated_at` column to `accounts` and block login, or hard-delete the customer record (cascading).

### Race Conditions
3. **Order number generation is a race.** `SELECT COALESCE(MAX(order_number), 10000)` + `INSERT order_number+1` is TOCTOU. Two concurrent bookings can collide on the UNIQUE constraint and fail silently. Fix: use `INSERT INTO order_cylinders ... RETURNING order_number` with a SQLite `AUTOINCREMENT` on `order_number`, or generate inside a `BEGIN EXCLUSIVE` transaction.

4. **Gas connection number / dealership ID generation race.** `SELECT COUNT(*) FROM customers` → `GAS{n+1}` has the same problem. Fix: same approach (let SQLite's AUTOINCREMENT drive the number).

### Security
5. **No CSRF protection.** All POST forms are vulnerable. Fix: implement double-submit cookie pattern or a server-side token per session.

6. **Request status value not validated.** `AdminUpdateRequestStatus` whitelists the table name but not the `status` value — arbitrary strings can be written. Fix: add a whitelist check on `status` before executing.

7. **No request body size limit.** Fix: add `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` at the top of every POST handler (or in a middleware).

### Robustness
8. **`rows.Err()` never checked.** Every query loop silently swallows mid-iteration DB errors. Fix: add `if err := rows.Err(); err != nil { ... }` after each `rows.Close()`.

9. **Missing DB indexes.** Hot paths query `order_cylinders` by `(account_id, cylinder_id)` and `(dealer_id, status)`, and `order_accessories` by `(dealer_id, status)`. None have indexes. Fix: add to `schema.sql`:
   ```sql
   CREATE INDEX IF NOT EXISTS idx_oc_account_cyl ON order_cylinders(account_id, cylinder_id);
   CREATE INDEX IF NOT EXISTS idx_oc_dealer_status ON order_cylinders(dealer_id, status);
   CREATE INDEX IF NOT EXISTS idx_oa_dealer_status ON order_accessories(dealer_id, status);
   ```

10. **No graceful shutdown.** `http.ListenAndServe` doesn't use a context. Fix: use `http.Server` with `Shutdown(ctx)` on SIGTERM/SIGINT.

### Operational
11. **Ephemeral session key.** The HMAC key is `rand.Read` at startup; all sessions are invalidated on every restart. Already noted in `SESSION_KEY` env var above — wire up the env var reading in `internal/auth/session.go`.

12. **No pagination.** Admin `/admin/customers` and `/admin/requests` load all rows. Add `LIMIT`/`OFFSET` pagination before the dataset grows.

---

## Development Workflow

```bash
# Compile check
go build ./...

# Vet
go vet ./...

# Run (dev)
go run .

# The binary embeds everything — no separate assets needed
go build -o gasmate . && ./gasmate
```

No test suite exists yet. When adding tests, use `database/sql` with an in-memory SQLite (`?mode=memory&cache=shared`) to avoid touching disk.

---

## Branch

Active development branch: `claude/modernize-egassewa-XRSKC`  
Repository: `glnarayanan/egassewa`
