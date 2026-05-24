PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS accounts (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    role          TEXT    NOT NULL CHECK(role IN ('customer','dealer','admin')),
    unique_number TEXT    UNIQUE,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS countries (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS states (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    country_id INTEGER NOT NULL REFERENCES countries(id)
);

CREATE TABLE IF NOT EXISTS cities (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    name     TEXT    NOT NULL,
    state_id INTEGER NOT NULL REFERENCES states(id)
);

CREATE TABLE IF NOT EXISTS dealers (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL REFERENCES accounts(id),
    dealership_id TEXT    NOT NULL UNIQUE,
    city_id       INTEGER NOT NULL REFERENCES cities(id),
    address       TEXT,
    phone         TEXT
);

CREATE TABLE IF NOT EXISTS customers (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id           INTEGER NOT NULL REFERENCES accounts(id),
    gas_connection_number TEXT   NOT NULL UNIQUE,
    dealer_id            INTEGER NOT NULL REFERENCES dealers(id),
    city_id              INTEGER NOT NULL REFERENCES cities(id),
    address              TEXT,
    phone                TEXT
);

CREATE TABLE IF NOT EXISTS gas_cylinders (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    unit_kg     REAL    NOT NULL UNIQUE,
    type        TEXT    NOT NULL CHECK(type IN ('domestic','commercial')),
    price       REAL    NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS accessories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    price       REAL    NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS order_cylinders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    order_number    INTEGER NOT NULL UNIQUE,
    account_id      INTEGER NOT NULL REFERENCES accounts(id),
    cylinder_id     INTEGER NOT NULL REFERENCES gas_cylinders(id),
    dealer_id       INTEGER NOT NULL REFERENCES dealers(id),
    order_count     INTEGER NOT NULL DEFAULT 1,
    ordered_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    next_order_date DATETIME NOT NULL,
    status          TEXT     NOT NULL DEFAULT 'WAITING'
                    CHECK(status IN ('WAITING','CONFIRMED','DISPATCHED','CANCELLED'))
);

CREATE TABLE IF NOT EXISTS order_accessories (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   INTEGER NOT NULL REFERENCES accounts(id),
    accessory_id INTEGER NOT NULL REFERENCES accessories(id),
    dealer_id    INTEGER NOT NULL REFERENCES dealers(id),
    order_count  INTEGER NOT NULL DEFAULT 1,
    ordered_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    status       TEXT     NOT NULL DEFAULT 'PENDING'
                 CHECK(status IN ('PENDING','CONFIRMED','DISPATCHED','CANCELLED'))
);

CREATE TABLE IF NOT EXISTS new_connection_requests (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id      INTEGER REFERENCES accounts(id),
    name            TEXT    NOT NULL,
    address         TEXT    NOT NULL,
    phone           TEXT    NOT NULL,
    city_id         INTEGER NOT NULL REFERENCES cities(id),
    id_proof_type   TEXT    NOT NULL,
    id_proof_number TEXT    NOT NULL,
    requested_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    status          TEXT     NOT NULL DEFAULT 'PENDING'
);

CREATE TABLE IF NOT EXISTS change_locations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id    INTEGER NOT NULL REFERENCES accounts(id),
    new_city_id   INTEGER NOT NULL REFERENCES cities(id),
    new_dealer_id INTEGER NOT NULL REFERENCES dealers(id),
    requested_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    status        TEXT     NOT NULL DEFAULT 'PENDING'
);

CREATE TABLE IF NOT EXISTS end_services (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id   INTEGER NOT NULL REFERENCES accounts(id),
    reason       TEXT,
    requested_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status       TEXT     NOT NULL DEFAULT 'PENDING'
);
