package spareparts

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound   = errors.New("spareparts: not found")
	ErrConflict   = errors.New("spareparts: conflict")
	ErrValidation = errors.New("spareparts: validation")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "spareparts: validation failed" }
func (e *ValidationError) Unwrap() error { return ErrValidation }

type Actor struct {
	ID   int64
	Role string
}

type Number struct {
	ID          int64     `json:"id"`
	SparepartID int64     `json:"sparepart_id"`
	PartNumber  string    `json:"part_number"`
	Brand       string    `json:"brand"`
	NumberType  string    `json:"number_type"`
	IsPrimary   bool      `json:"is_primary"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type Alternative struct {
	ID                       int64      `json:"id"`
	SparepartID              int64      `json:"sparepart_id"`
	SparepartCode            string     `json:"sparepart_code"`
	SparepartName            string     `json:"sparepart_name"`
	AlternativeSparepartID   int64      `json:"alternative_sparepart_id"`
	AlternativeSparepartCode string     `json:"alternative_sparepart_code"`
	AlternativeSparepartName string     `json:"alternative_sparepart_name"`
	RelationType             string     `json:"relation_type"`
	VerificationStatus       string     `json:"verification_status"`
	Notes                    string     `json:"notes"`
	VerifiedBy               *int64     `json:"verified_by,omitempty"`
	VerifiedByName           string     `json:"verified_by_name"`
	VerifiedAt               *time.Time `json:"verified_at,omitempty"`
	Direction                string     `json:"direction"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type Compatibility struct {
	ID              int64      `json:"id"`
	SparepartID     int64      `json:"sparepart_id"`
	UnitModelID     *int64     `json:"unit_model_id,omitempty"`
	UnitModelName   string     `json:"unit_model_name"`
	EngineModelID   *int64     `json:"engine_model_id,omitempty"`
	EngineModelName string     `json:"engine_model_name"`
	Configuration   string     `json:"configuration"`
	Status          string     `json:"status"`
	ConditionNote   string     `json:"condition_note"`
	Source          string     `json:"source"`
	Notes           string     `json:"notes"`
	VerifiedBy      *int64     `json:"verified_by,omitempty"`
	VerifiedByName  string     `json:"verified_by_name"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Sparepart struct {
	ID                    int64           `json:"id"`
	Code                  string          `json:"code"`
	Name                  string          `json:"name"`
	Category              string          `json:"category"`
	UOM                   string          `json:"uom"`
	Description           string          `json:"description"`
	IsActive              bool            `json:"is_active"`
	PrimaryPartNumber     string          `json:"primary_part_number"`
	CompatibilityCount    int             `json:"compatibility_count"`
	ApprovedCompatibility int             `json:"approved_compatibility_count"`
	Numbers               []Number        `json:"numbers"`
	Alternatives          []Alternative   `json:"alternatives"`
	Compatibility         []Compatibility `json:"compatibility"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type Input struct {
	Code        string
	Name        string
	Category    string
	UOM         string
	Description string
	IsActive    *bool
}

type Patch struct {
	Code        *string
	Name        *string
	Category    *string
	UOM         *string
	Description *string
	IsActive    *bool
}

type NumberInput struct {
	PartNumber string
	Brand      string
	NumberType string
	IsPrimary  bool
	Notes      string
}

type NumberPatch struct {
	PartNumber *string
	Brand      *string
	NumberType *string
	IsPrimary  *bool
	Notes      *string
}

type AlternativeInput struct {
	AlternativeSparepartID int64
	RelationType           string
	Notes                  string
}

type CompatibilityInput struct {
	UnitModelID   *int64
	EngineModelID *int64
	Configuration string
	Source        string
	Notes         string
}

type VerifyInput struct {
	Status        string
	ConditionNote string
	Notes         string
	Source        string
}

type Filter struct {
	Query               string
	Category            string
	ActiveOnly          *bool
	UnitModelID         *int64
	EngineModelID       *int64
	CompatibilityStatus string
	Page                int
	Limit               int
	Sort                string
	Order               string
}

type Service interface {
	List(ctx context.Context, filter Filter) ([]Sparepart, int, error)
	Get(ctx context.Context, id int64) (Sparepart, error)
	Create(ctx context.Context, actor Actor, input Input) (Sparepart, error)
	Update(ctx context.Context, id int64, actor Actor, patch Patch) (Sparepart, error)
	ListNumbers(ctx context.Context, sparepartID int64) ([]Number, error)
	CreateNumber(ctx context.Context, sparepartID int64, actor Actor, input NumberInput) (Number, error)
	UpdateNumber(ctx context.Context, sparepartID, numberID int64, actor Actor, patch NumberPatch) (Number, error)
	ListAlternatives(ctx context.Context, sparepartID int64) ([]Alternative, error)
	CreateAlternative(ctx context.Context, sparepartID int64, actor Actor, input AlternativeInput) (Alternative, error)
	VerifyAlternative(ctx context.Context, sparepartID, alternativeID int64, actor Actor, input VerifyInput) (Alternative, error)
	ListCompatibility(ctx context.Context, sparepartID int64) ([]Compatibility, error)
	CreateCompatibility(ctx context.Context, sparepartID int64, actor Actor, input CompatibilityInput) (Compatibility, error)
	VerifyCompatibility(ctx context.Context, sparepartID, compatibilityID int64, actor Actor, input VerifyInput) (Compatibility, error)
}
