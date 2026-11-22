package reporting

import (
	"context"
	"io"
	"time"
)

// Store is the read-only persistence surface used by the reporting service.
type Store interface {
	Summary(ctx context.Context) (map[string]any, error)
	PM(ctx context.Context, filter Filter) (DashboardPM, error)
	Material(ctx context.Context, filter Filter) (DashboardMaterial, error)
	Inventory(ctx context.Context, filter Filter) (DashboardInventory, error)
	Maintenance(ctx context.Context, filter Filter) (DashboardMaintenance, error)
	Report(ctx context.Context, name string, filter Filter) (Table, error)
}

var reportFiles = map[string]string{
	"pm-compliance":         "pm-compliance.csv",
	"shortage-aging":        "shortage-aging.csv",
	"sparepart-consumption": "sparepart-consumption.csv",
	"inventory-movement":    "inventory-movement.csv",
	"component-replacement": "component-replacement.csv",
}

// ReportNames lists the supported report identifiers for route validation.
func ReportNames() []string {
	names := make([]string, 0, len(reportFiles))
	for name := range reportFiles {
		names = append(names, name)
	}
	return names
}

type Service struct {
	repo Store
	loc  *time.Location
}

func NewService(repo Store, loc *time.Location) *Service {
	if loc == nil {
		loc = time.UTC
	}
	return &Service{repo: repo, loc: loc}
}

func (s *Service) Summary(ctx context.Context) (map[string]any, error) {
	return s.repo.Summary(ctx)
}

func (s *Service) PM(ctx context.Context, filter Filter) (DashboardPM, error) {
	return s.repo.PM(ctx, filter)
}

func (s *Service) Material(ctx context.Context, filter Filter) (DashboardMaterial, error) {
	return s.repo.Material(ctx, filter)
}

func (s *Service) Inventory(ctx context.Context, filter Filter) (DashboardInventory, error) {
	return s.repo.Inventory(ctx, filter)
}

func (s *Service) Maintenance(ctx context.Context, filter Filter) (DashboardMaintenance, error) {
	return s.repo.Maintenance(ctx, filter)
}

// Report returns a table plus the safe download filename for the report.
func (s *Service) Report(ctx context.Context, name string, filter Filter) (Table, string, error) {
	filename, ok := reportFiles[name]
	if !ok {
		return Table{}, "", ErrInvalidFilter
	}
	table, err := s.repo.Report(ctx, name, filter)
	if err != nil {
		return Table{}, "", err
	}
	return table, filename, nil
}

// RenderCSV renders a table using the configured application timezone.
func (s *Service) RenderCSV(w io.Writer, table Table) error {
	return WriteCSV(w, table, s.loc)
}
