package inventory

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("inventory: not found")
	ErrConflict          = errors.New("inventory: conflict")
	ErrValidation        = errors.New("inventory: validation")
	ErrInsufficientStock = errors.New("inventory: insufficient stock")
	ErrInvalidState      = errors.New("inventory: invalid state")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "inventory: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Actor struct {
	ID   int64
	Role string
}

type Warehouse struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	SiteLocation  string    `json:"site_location"`
	IsActive      bool      `json:"is_active"`
	LocationCount int       `json:"location_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type WarehouseInput struct {
	Code         string
	Name         string
	SiteLocation string
	IsActive     *bool
}

type WarehousePatch struct {
	Code         *string
	Name         *string
	SiteLocation *string
	IsActive     *bool
}

type Location struct {
	ID            int64     `json:"id"`
	WarehouseID   int64     `json:"warehouse_id"`
	WarehouseCode string    `json:"warehouse_code"`
	WarehouseName string    `json:"warehouse_name"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	LocationType  string    `json:"location_type"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type LocationInput struct {
	WarehouseID  int64
	Code         string
	Name         string
	LocationType string
	IsActive     *bool
}

type LocationPatch struct {
	Code         *string
	Name         *string
	LocationType *string
	IsActive     *bool
}

type Policy struct {
	ID            int64     `json:"id"`
	SparepartID   int64     `json:"sparepart_id"`
	SparepartCode string    `json:"sparepart_code"`
	SparepartName string    `json:"sparepart_name"`
	UOM           string    `json:"uom"`
	LocationID    int64     `json:"warehouse_location_id"`
	WarehouseID   int64     `json:"warehouse_id"`
	WarehouseCode string    `json:"warehouse_code"`
	LocationCode  string    `json:"location_code"`
	LocationName  string    `json:"location_name"`
	MinQty        float64   `json:"min_qty"`
	CriticalQty   float64   `json:"critical_qty"`
	ReorderQty    *float64  `json:"reorder_qty,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type PolicyInput struct {
	SparepartID int64
	LocationID  int64
	MinQty      float64
	CriticalQty float64
	ReorderQty  *float64
	IsActive    *bool
}

type PolicyPatch struct {
	MinQty       *float64
	CriticalQty  *float64
	ReorderQty   *float64
	ClearReorder bool
	IsActive     *bool
}

type PolicyFilter struct {
	Query       string
	SparepartID *int64
	LocationID  *int64
	ActiveOnly  bool
	Page        int
	Limit       int
}

type Balance struct {
	SparepartID     int64      `json:"sparepart_id"`
	SparepartCode   string     `json:"sparepart_code"`
	SparepartName   string     `json:"sparepart_name"`
	Category        string     `json:"category"`
	UOM             string     `json:"uom"`
	SparepartActive bool       `json:"sparepart_active"`
	LocationID      int64      `json:"warehouse_location_id"`
	WarehouseID     int64      `json:"warehouse_id"`
	WarehouseCode   string     `json:"warehouse_code"`
	WarehouseName   string     `json:"warehouse_name"`
	LocationCode    string     `json:"location_code"`
	LocationName    string     `json:"location_name"`
	LocationType    string     `json:"location_type"`
	OnHand          float64    `json:"on_hand"`
	Reserved        float64    `json:"reserved"`
	Available       float64    `json:"available"`
	PolicyID        *int64     `json:"policy_id,omitempty"`
	MinQty          float64    `json:"min_qty"`
	CriticalQty     float64    `json:"critical_qty"`
	ReorderQty      *float64   `json:"reorder_qty,omitempty"`
	StockStatus     string     `json:"stock_status"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
}

type BalanceFilter struct {
	Query       string
	WarehouseID *int64
	LocationID  *int64
	Status      string
	Page        int
	Limit       int
	Sort        string
	Order       string
}

type PostItem struct {
	SparepartID    int64
	FromLocationID *int64
	ToLocationID   *int64
	ReservationID  *int64
	Qty            float64
	Notes          string
}

// PostedTransaction is returned by PostTx so callers that compose inventory
// with material usage can attach usage rows to the inserted ledger lines.
type PostedTransaction struct {
	ID      int64
	Code    string
	ItemIDs []int64
}

type PostInput struct {
	Type           string
	ReferenceType  string
	ReferenceID    *int64
	ReferenceNo    string
	Notes          string
	IdempotencyKey string
	Items          []PostItem
}

type TransactionItem struct {
	ID             int64   `json:"id"`
	TransactionID  int64   `json:"transaction_id"`
	SparepartID    int64   `json:"sparepart_id"`
	SparepartCode  string  `json:"sparepart_code"`
	SparepartName  string  `json:"sparepart_name"`
	UOM            string  `json:"uom"`
	FromLocationID *int64  `json:"from_location_id,omitempty"`
	FromLocation   string  `json:"from_location"`
	ToLocationID   *int64  `json:"to_location_id,omitempty"`
	ToLocation     string  `json:"to_location"`
	Qty            float64 `json:"qty"`
	Notes          string  `json:"notes"`
}

type Transaction struct {
	ID             int64             `json:"id"`
	Code           string            `json:"code"`
	Type           string            `json:"type"`
	Status         string            `json:"status"`
	ReferenceType  string            `json:"reference_type"`
	ReferenceID    *int64            `json:"reference_id,omitempty"`
	ReferenceNo    string            `json:"reference_no"`
	ReversalOfID   *int64            `json:"reversal_of_id,omitempty"`
	IdempotencyKey string            `json:"idempotency_key"`
	PostedBy       *int64            `json:"posted_by,omitempty"`
	PostedByName   string            `json:"posted_by_name"`
	PostedAt       time.Time         `json:"posted_at"`
	Notes          string            `json:"notes"`
	ItemCount      int               `json:"item_count"`
	Items          []TransactionItem `json:"items"`
	CreatedAt      time.Time         `json:"created_at"`
}

type TransactionFilter struct {
	Type  string
	Query string
	From  *time.Time
	To    *time.Time
	Page  int
	Limit int
}

type Movement struct {
	ID              int64     `json:"id"`
	TransactionID   int64     `json:"transaction_id"`
	TransactionCode string    `json:"transaction_code"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	ReferenceNo     string    `json:"reference_no"`
	PostedAt        time.Time `json:"posted_at"`
	SparepartID     int64     `json:"sparepart_id"`
	SparepartCode   string    `json:"sparepart_code"`
	SparepartName   string    `json:"sparepart_name"`
	UOM             string    `json:"uom"`
	FromLocationID  *int64    `json:"from_location_id,omitempty"`
	FromLocation    string    `json:"from_location"`
	ToLocationID    *int64    `json:"to_location_id,omitempty"`
	ToLocation      string    `json:"to_location"`
	Qty             float64   `json:"qty"`
	Notes           string    `json:"notes"`
}

type MovementFilter struct {
	Query       string
	Type        string
	SparepartID *int64
	LocationID  *int64
	From        *time.Time
	To          *time.Time
	Page        int
	Limit       int
}

type OpnameItem struct {
	ID            int64    `json:"id"`
	SessionID     int64    `json:"session_id"`
	SparepartID   int64    `json:"sparepart_id"`
	SparepartCode string   `json:"sparepart_code"`
	SparepartName string   `json:"sparepart_name"`
	UOM           string   `json:"uom"`
	LocationID    int64    `json:"location_id"`
	WarehouseCode string   `json:"warehouse_code"`
	LocationCode  string   `json:"location_code"`
	LocationName  string   `json:"location_name"`
	SystemQty     float64  `json:"system_qty"`
	CountedQty    *float64 `json:"counted_qty,omitempty"`
	Difference    *float64 `json:"difference,omitempty"`
	Notes         string   `json:"notes"`
}

type OpnameSession struct {
	ID            int64        `json:"id"`
	Code          string       `json:"code"`
	WarehouseID   int64        `json:"warehouse_id"`
	WarehouseCode string       `json:"warehouse_code"`
	WarehouseName string       `json:"warehouse_name"`
	Status        string       `json:"status"`
	StartedAt     *time.Time   `json:"started_at,omitempty"`
	PostedAt      *time.Time   `json:"posted_at,omitempty"`
	PostedBy      *int64       `json:"posted_by,omitempty"`
	PostedByName  string       `json:"posted_by_name"`
	Notes         string       `json:"notes"`
	CancelReason  string       `json:"cancel_reason"`
	CancelledAt   *time.Time   `json:"cancelled_at,omitempty"`
	CreatedBy     *int64       `json:"created_by,omitempty"`
	ItemCount     int          `json:"item_count"`
	CountedCount  int          `json:"counted_count"`
	Items         []OpnameItem `json:"items"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type OpnameFilter struct {
	Status      string
	WarehouseID *int64
	Page        int
	Limit       int
}

type OpnameCreateInput struct {
	WarehouseID int64
	Notes       string
}

type OpnameCountItem struct {
	ItemID     int64
	CountedQty float64
	Notes      string
}

type OpnameCountInput struct {
	Items []OpnameCountItem
}

type ReconciliationRow struct {
	SparepartID         int64   `json:"sparepart_id"`
	SparepartCode       string  `json:"sparepart_code"`
	SparepartName       string  `json:"sparepart_name"`
	LocationID          int64   `json:"warehouse_location_id"`
	WarehouseCode       string  `json:"warehouse_code"`
	LocationCode        string  `json:"location_code"`
	BalanceOnHand       float64 `json:"balance_on_hand"`
	LedgerOnHand        float64 `json:"ledger_on_hand"`
	Difference          float64 `json:"difference"`
	BalanceReserved     float64 `json:"balance_reserved"`
	ReservationReserved float64 `json:"reservation_reserved"`
	ReservedDifference  float64 `json:"reserved_difference"`
}

type Reconciliation struct {
	Rows      []ReconciliationRow `json:"rows"`
	Mismatch  int                 `json:"mismatch"`
	CheckedAt time.Time           `json:"checked_at"`
}

type StockAlertResult struct {
	Low      int `json:"low"`
	Critical int `json:"critical"`
	Notified int `json:"notified"`
}

type Service interface {
	ListWarehouses(ctx context.Context, query string, activeOnly bool, page, limit int) ([]Warehouse, int, error)
	CreateWarehouse(ctx context.Context, actor Actor, input WarehouseInput) (Warehouse, error)
	UpdateWarehouse(ctx context.Context, id int64, actor Actor, patch WarehousePatch) (Warehouse, error)
	ListLocations(ctx context.Context, warehouseID *int64, query string, activeOnly bool) ([]Location, error)
	CreateLocation(ctx context.Context, actor Actor, input LocationInput) (Location, error)
	UpdateLocation(ctx context.Context, id int64, actor Actor, patch LocationPatch) (Location, error)

	ListPolicies(ctx context.Context, filter PolicyFilter) ([]Policy, int, error)
	UpsertPolicy(ctx context.Context, actor Actor, input PolicyInput) (Policy, error)
	UpdatePolicy(ctx context.Context, id int64, actor Actor, patch PolicyPatch) (Policy, error)

	ListBalances(ctx context.Context, filter BalanceFilter) ([]Balance, int, error)
	ListTransactions(ctx context.Context, filter TransactionFilter) ([]Transaction, int, error)
	GetTransaction(ctx context.Context, id int64) (Transaction, error)
	Post(ctx context.Context, actor Actor, input PostInput) (Transaction, error)
	Reverse(ctx context.Context, id int64, actor Actor, reason string) (Transaction, error)
	ListMovements(ctx context.Context, filter MovementFilter) ([]Movement, int, error)
	Reconcile(ctx context.Context) (Reconciliation, error)

	ListOpnameSessions(ctx context.Context, filter OpnameFilter) ([]OpnameSession, int, error)
	GetOpnameSession(ctx context.Context, id int64) (OpnameSession, error)
	CreateOpname(ctx context.Context, actor Actor, input OpnameCreateInput) (OpnameSession, error)
	CountOpname(ctx context.Context, id int64, actor Actor, input OpnameCountInput) (OpnameSession, error)
	PostOpname(ctx context.Context, id int64, actor Actor) (OpnameSession, error)
	CancelOpname(ctx context.Context, id int64, actor Actor, reason string) (OpnameSession, error)

	SendLowStockAlerts(ctx context.Context, now time.Time) (StockAlertResult, error)
}
