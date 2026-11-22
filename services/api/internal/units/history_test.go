package units

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDowntimeResultMergesOverlappingIntervals(t *testing.T) {
	base := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	events := []DowntimeEvent{
		{
			WOID: 1, DowntimeClass: "unplanned", WaitingMinutes: 30,
			DowntimeStart: timePtr(base), DowntimeEnd: base.Add(4 * time.Hour),
		},
		{
			WOID: 2, DowntimeClass: "unplanned", WaitingMinutes: 15,
			DowntimeStart: timePtr(base.Add(2 * time.Hour)), DowntimeEnd: base.Add(6 * time.Hour),
		},
		{
			WOID: 3, DowntimeClass: "planned",
			DowntimeStart: timePtr(base.Add(7 * time.Hour)), DowntimeEnd: base.Add(9 * time.Hour),
		},
	}
	result := buildDowntimeResult(events, DowntimeFilter{})
	if result.Summary.UnplannedMinutes != 360 {
		t.Fatalf("unplanned minutes = %v, want 360", result.Summary.UnplannedMinutes)
	}
	if result.Summary.PlannedMinutes != 120 || result.Summary.TotalMinutes != 480 {
		t.Fatalf("planned/total = %v/%v, want 120/480", result.Summary.PlannedMinutes, result.Summary.TotalMinutes)
	}
	if result.Summary.MTTRMinutes != 240 {
		t.Fatalf("mttr = %v, want 240", result.Summary.MTTRMinutes)
	}
	if result.Summary.WaitingMinutes != 45 {
		t.Fatalf("waiting = %v, want 45", result.Summary.WaitingMinutes)
	}
}

func TestDowntimeResultClipsToPeriod(t *testing.T) {
	base := time.Date(2026, 11, 10, 8, 0, 0, 0, time.UTC)
	from := base.Add(2 * time.Hour)
	to := base.Add(5 * time.Hour)
	events := []DowntimeEvent{{
		WOID: 1, DowntimeClass: "unplanned",
		DowntimeStart: timePtr(base), DowntimeEnd: base.Add(4 * time.Hour),
	}}
	result := buildDowntimeResult(events, DowntimeFilter{From: &from, To: &to})
	if result.Summary.TotalMinutes != 120 || result.Summary.UnplannedMinutes != 120 {
		t.Fatalf("summary = %#v", result.Summary)
	}
}

func TestListMaintenanceHistoryValidatesFilter(t *testing.T) {
	service := NewService(&fakeRepository{}, nil)
	_, _, err := service.ListMaintenanceHistory(context.Background(), 1, HistoryFilter{
		Page: 0, Limit: 500, Status: "unknown", MaintenanceType: "breakdown",
	})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["status"] == "" {
		t.Fatalf("error = %v, want status validation", err)
	}
}

func TestListMaintenanceHistoryNormalizesFilter(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	_, _, err := service.ListMaintenanceHistory(context.Background(), 1, HistoryFilter{
		Page: 0, Limit: 500, Status: " DONE ", MaintenanceType: " BREAKDOWN ",
	})
	if err != nil {
		t.Fatalf("ListMaintenanceHistory() error = %v", err)
	}
	if repo.historyFilter.Page != 1 || repo.historyFilter.Limit != MaxPageLimit {
		t.Fatalf("pagination = %#v", repo.historyFilter)
	}
	if repo.historyFilter.Status != "done" || repo.historyFilter.MaintenanceType != "breakdown" {
		t.Fatalf("filter = %#v", repo.historyFilter)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
