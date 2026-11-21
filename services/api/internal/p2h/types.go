package p2h

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrReportNotFound = errors.New("p2h: report not found")
	ErrSyncInProgress = errors.New("p2h: another sync is already running")
)

type Report struct {
	ID              int64
	OdooReportID    int64
	Name            string
	ReportDate      *time.Time
	UnitID          *int64
	OdooUnitID      *int64
	UnitCode        string
	UnitType        string
	Shift           string
	ForemanName     string
	EmployeeName    string
	Location        string
	State           string
	ApprovalStatus  string
	KMAkhir         *float64
	HMAwal          *float64
	Notes           string
	CheckedItems    int
	FindingsCount   int
	RusakCount      int
	DetailSyncedAt  *time.Time
	SyncedAt        *time.Time
	Raw             json.RawMessage
	Findings        []Finding
	CandidateID     *int64
	CandidateStatus string
}

type Finding struct {
	ID           int64
	OdooLineID   int64
	ReportID     int64
	QuestionID   *int64
	Component    string
	Category     string
	CategoryNorm string
	Condition    string
	Remark       string
}

type ReportFilter struct {
	Query  string
	State  string
	UnitID *int64
	Page   int
	Limit  int
	Sort   string
	Order  string
}

type SyncCounters struct {
	Reports    int
	Details    int
	Findings   int
	Candidates int
	WorkOrders int
}

type SettingsValues struct {
	AutoWO       bool
	BackfillDays int
	SLAHours     int
}

type CandidateCreated struct {
	ID          int64
	Archived    bool
	AutoWO      bool
	WorkOrderID *int64
}
