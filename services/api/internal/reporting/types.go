package reporting

import (
	"errors"
	"time"
)

const (
	BucketDay   = "day"
	BucketWeek  = "week"
	BucketMonth = "month"

	FormatJSON = "json"
	FormatCSV  = "csv"

	DefaultLimit = 25
	MaxPageLimit = 100
)

var ErrInvalidFilter = errors.New("reporting: invalid filter")

// ValidationError carries field-level messages to the HTTP layer.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return ErrInvalidFilter.Error()
}

func (e *ValidationError) Unwrap() error {
	return ErrInvalidFilter
}

// Filter is shared by dashboard and report queries. From/To are UTC instants
// derived from inclusive WITA calendar dates; To is exclusive.
type Filter struct {
	From        *time.Time
	To          *time.Time
	UnitID      *int64
	UnitModelID *int64
	LocationID  *int64
	Bucket      string
	GroupBy     string
	Status      string
	Format      string
	Page        int
	Limit       int
}

// Column describes one report column for both JSON and CSV output.
type Column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Pagination struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

// Table is the common read-only shape for every report and dashboard list.
type Table struct {
	Columns []Column         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Meta    Pagination       `json:"meta"`
}

type DashboardPM struct {
	Summary map[string]any `json:"summary"`
	Due     Table          `json:"due"`
}

type DashboardMaterial struct {
	Summary      map[string]any `json:"summary"`
	Requirements Table          `json:"requirements"`
	Aging        Table          `json:"aging"`
}

type DashboardInventory struct {
	Summary   map[string]any `json:"summary"`
	LowStock  Table          `json:"low_stock"`
	Movements Table          `json:"movements"`
}

type DashboardMaintenance struct {
	Summary  map[string]any `json:"summary"`
	Activity Table          `json:"activity"`
	Downtime Table          `json:"downtime"`
	Waiting  Table          `json:"waiting"`
}
