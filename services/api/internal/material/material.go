package material

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("material: not found")
	ErrConflict     = errors.New("material: conflict")
	ErrValidation   = errors.New("material: validation")
	ErrInvalidState = errors.New("material: invalid state")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "material: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Actor struct {
	ID   int64
	Role string
}

type Requirement struct {
	ID              int64      `json:"id"`
	Code            string     `json:"code"`
	WOID            int64      `json:"wo_id"`
	WOCode          string     `json:"wo_code"`
	WOTitle         string     `json:"wo_title"`
	UnitCode        string     `json:"unit_code"`
	Source          string     `json:"source"`
	Status          string     `json:"status"`
	StatusChangedAt time.Time  `json:"status_changed_at"`
	ShortageSince   *time.Time `json:"shortage_since,omitempty"`
	RequestedBy     *int64     `json:"requested_by,omitempty"`
	RequestedByName string     `json:"requested_by_name"`
	RequestedAt     *time.Time `json:"requested_at,omitempty"`
	Notes           string     `json:"notes"`
	ItemCount       int        `json:"item_count"`
	RequiredCount   int        `json:"required_count"`
	ShortageCount   int        `json:"shortage_count"`
	DerivedStatus   string     `json:"derived_status"`
	Items           []Item     `json:"items"`
	Usages          []Usage    `json:"usages"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Item struct {
	ID                     int64   `json:"id"`
	RequirementID          int64   `json:"material_requirement_id"`
	SparepartID            int64   `json:"sparepart_id"`
	SparepartCode          string  `json:"sparepart_code"`
	SparepartName          string  `json:"sparepart_name"`
	UOM                    string  `json:"uom"`
	QtyRequired            float64 `json:"qty_required"`
	QtyReserved            float64 `json:"qty_reserved"`
	QtyIssued              float64 `json:"qty_issued"`
	QtyReturned            float64 `json:"qty_returned"`
	NetIssued              float64 `json:"net_issued"`
	Available              float64 `json:"available"`
	Coverage               float64 `json:"coverage"`
	Shortage               float64 `json:"shortage"`
	IsOptional             bool    `json:"is_optional"`
	PreferredWarehouseID   *int64  `json:"preferred_warehouse_id,omitempty"`
	PreferredWarehouseCode string  `json:"preferred_warehouse_code"`
	Notes                  string  `json:"notes"`
}

type UpdateRow struct {
	ID            int64     `json:"id"`
	RequirementID int64     `json:"material_requirement_id"`
	UserID        *int64    `json:"user_id,omitempty"`
	UserName      string    `json:"user_name"`
	StatusFrom    string    `json:"status_from"`
	StatusTo      string    `json:"status_to"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
}

type Reservation struct {
	ID                int64      `json:"id"`
	RequirementItemID int64      `json:"material_requirement_item_id"`
	RequirementID     int64      `json:"material_requirement_id"`
	RequirementCode   string     `json:"requirement_code"`
	WOID              int64      `json:"wo_id"`
	WOCode            string     `json:"wo_code"`
	SparepartID       int64      `json:"sparepart_id"`
	SparepartCode     string     `json:"sparepart_code"`
	SparepartName     string     `json:"sparepart_name"`
	UOM               string     `json:"uom"`
	LocationID        int64      `json:"warehouse_location_id"`
	WarehouseCode     string     `json:"warehouse_code"`
	LocationCode      string     `json:"location_code"`
	LocationName      string     `json:"location_name"`
	Qty               float64    `json:"qty"`
	Status            string     `json:"status"`
	ReservedBy        *int64     `json:"reserved_by,omitempty"`
	ReservedByName    string     `json:"reserved_by_name"`
	ReservedAt        time.Time  `json:"reserved_at"`
	ReleasedAt        *time.Time `json:"released_at,omitempty"`
	ConsumedAt        *time.Time `json:"consumed_at,omitempty"`
	Notes             string     `json:"notes"`
}

type Usage struct {
	ID                int64     `json:"id"`
	WOID              int64     `json:"wo_id"`
	RequirementItemID *int64    `json:"material_requirement_item_id,omitempty"`
	SparepartID       int64     `json:"sparepart_id"`
	SparepartCode     string    `json:"sparepart_code"`
	SparepartName     string    `json:"sparepart_name"`
	ComponentID       *int64    `json:"component_id,omitempty"`
	ComponentName     string    `json:"component_name"`
	LocationID        int64     `json:"warehouse_location_id"`
	WarehouseCode     string    `json:"warehouse_code"`
	LocationCode      string    `json:"location_code"`
	Qty               float64   `json:"qty"`
	HM                *float64  `json:"hm,omitempty"`
	KM                *float64  `json:"km,omitempty"`
	MovementType      string    `json:"movement_type"`
	InventoryItemID   int64     `json:"inventory_transaction_item_id"`
	ReturnOfUsageID   *int64    `json:"return_of_usage_id,omitempty"`
	UsedBy            *int64    `json:"used_by,omitempty"`
	UsedByName        string    `json:"used_by_name"`
	UsedAt            time.Time `json:"used_at"`
	Notes             string    `json:"notes"`
}

type Filter struct {
	WOID   *int64
	Status string
	Query  string
	Page   int
	Limit  int
}

type ReservationFilter struct {
	Status        string
	SparepartID   *int64
	RequirementID *int64
	WOID          *int64
	Page          int
	Limit         int
}

type ItemInput struct {
	SparepartID          int64
	QtyRequired          float64
	IsOptional           bool
	PreferredWarehouseID *int64
	Notes                string
}

type CreateInput struct {
	WOID   int64
	Source string
	Notes  string
	Items  []ItemInput
}

type PatchInput struct {
	Status *string
	Notes  *string
	Items  *[]ItemInput
}

type ReserveInput struct {
	IdempotencyKey string
	Notes          string
}

type IssueItemInput struct {
	MaterialItemID int64
	Qty            float64
	ComponentID    *int64
	Notes          string
}

type IssueInput struct {
	IdempotencyKey string
	Notes          string
	Items          []IssueItemInput
}

type ReturnItemInput struct {
	UsageID        *int64
	MaterialItemID *int64
	LocationID     *int64
	Qty            float64
	Notes          string
}

type ReturnInput struct {
	IdempotencyKey string
	Notes          string
	Items          []ReturnItemInput
}

type CreateReservationInput struct {
	MaterialItemID int64
	LocationID     int64
	Qty            float64
	Notes          string
}

type RecomputeResult struct {
	Updated int
}

type Service interface {
	List(ctx context.Context, filter Filter) ([]Requirement, int, error)
	Get(ctx context.Context, id int64) (Requirement, error)
	Create(ctx context.Context, actor Actor, input CreateInput) (Requirement, error)
	Update(ctx context.Context, id int64, actor Actor, patch PatchInput) (Requirement, error)
	ListUpdates(ctx context.Context, id int64) ([]UpdateRow, error)
	ListUsages(ctx context.Context, id int64) ([]Usage, error)
	CheckStock(ctx context.Context, id int64, actor Actor) (Requirement, error)
	Reserve(ctx context.Context, id int64, actor Actor, input ReserveInput) (Requirement, error)
	Issue(ctx context.Context, id int64, actor Actor, input IssueInput) (Requirement, error)
	Return(ctx context.Context, id int64, actor Actor, input ReturnInput) (Requirement, error)
	Cancel(ctx context.Context, id int64, actor Actor, reason string) (Requirement, error)

	ListReservations(ctx context.Context, filter ReservationFilter) ([]Reservation, int, error)
	CreateReservation(ctx context.Context, actor Actor, input CreateReservationInput) (Reservation, error)
	ReleaseReservation(ctx context.Context, id int64, actor Actor, reason string) (Reservation, error)

	RecomputeForSpareparts(ctx context.Context, sparepartIDs []int64) (RecomputeResult, error)
	RecomputeAll(ctx context.Context) (RecomputeResult, error)
}
