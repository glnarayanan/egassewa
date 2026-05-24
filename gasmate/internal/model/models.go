package model

import "time"

type Account struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	Role         string // customer | dealer | admin
	UniqueNumber string
	CreatedAt    time.Time
}

type Country struct {
	ID   int64
	Name string
}

type State struct {
	ID        int64
	Name      string
	CountryID int64
}

type City struct {
	ID      int64
	Name    string
	StateID int64
}

type Dealer struct {
	ID           int64
	AccountID    int64
	DealershipID string
	CityID       int64
	CityName     string
	StateName    string
	Address      string
	Phone        string
	Username     string
	Email        string
}

type Customer struct {
	ID                  int64
	AccountID           int64
	GasConnectionNumber string
	DealerID            int64
	CityID              int64
	Address             string
	Phone               string
	Username            string
	Email               string
	CityName            string
	DealerName          string
}

type GasCylinder struct {
	ID          int64
	UnitKg      float64
	Type        string // domestic | commercial
	Price       float64
	Description string
}

type Accessory struct {
	ID          int64
	Name        string
	Price       float64
	Description string
}

type OrderCylinder struct {
	ID            int64
	OrderNumber   int64
	AccountID     int64
	CylinderID    int64
	DealerID      int64
	OrderCount    int
	OrderedAt     time.Time
	NextOrderDate time.Time
	Status        string // WAITING | CONFIRMED | DISPATCHED | CANCELLED
	CylinderUnit  float64
	CylinderType  string
	Username      string
	Email         string
}

type OrderAccessory struct {
	ID          int64
	AccountID   int64
	AccessoryID int64
	DealerID    int64
	OrderCount  int
	OrderedAt   time.Time
	Status      string // PENDING | CONFIRMED | DISPATCHED | CANCELLED
	ItemName    string
	Username    string
	Email       string
}

type NewConnectionRequest struct {
	ID            int64
	AccountID     int64
	Name          string
	Address       string
	Phone         string
	CityID        int64
	IDProofType   string
	IDProofNumber string
	RequestedAt   time.Time
	Status        string
	Username      string
	CityName      string
}

type ChangeLocation struct {
	ID          int64
	AccountID   int64
	NewCityID   int64
	NewDealerID int64
	RequestedAt time.Time
	Status      string
	Username    string
	CityName    string
	DealerName  string
}

type EndService struct {
	ID          int64
	AccountID   int64
	Reason      string
	RequestedAt time.Time
	Status      string
	Username    string
}
