package pmcalc

import (
	"testing"
	"time"
)

func TestEvaluateCalendarBuckets(t *testing.T) {
	today := dateOnly(time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC))
	due := dateOnly(today.AddDate(0, 0, 10))
	result := Evaluate(Plan{}, Schedule{DueDate: &due}, 0, 0, today)
	if result.Bucket != BucketUpcoming || result.Due || result.Overdue {
		t.Fatalf("10 days should be upcoming, got %+v", result)
	}
	due = today.AddDate(0, 0, 3)
	result = Evaluate(Plan{}, Schedule{DueDate: &due}, 0, 0, today)
	if result.Bucket != BucketDueSoon {
		t.Fatalf("3 days should be due soon, got %+v", result)
	}
	due = today
	result = Evaluate(Plan{}, Schedule{DueDate: &due}, 0, 0, today)
	if result.Bucket != BucketDue || !result.Due {
		t.Fatalf("today should be due, got %+v", result)
	}
	due = today.AddDate(0, 0, -1)
	result = Evaluate(Plan{}, Schedule{DueDate: &due}, 0, 0, today)
	if result.Bucket != BucketOverdue || !result.Overdue {
		t.Fatalf("yesterday should be overdue, got %+v", result)
	}
}

func TestEvaluateFirstReached(t *testing.T) {
	today := dateOnly(time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC))
	dueHM := 100.0
	dueDate := today.AddDate(0, 0, 20)
	result := Evaluate(Plan{IntervalRule: "first_reached"}, Schedule{
		DueHM:   &dueHM,
		DueDate: &dueDate,
	}, 101, 0, today)
	if result.Bucket != BucketOverdue || !result.Due {
		t.Fatalf("HM threshold should make schedule overdue, got %+v", result)
	}
}

func TestEvaluateAllRequired(t *testing.T) {
	today := dateOnly(time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC))
	dueHM := 100.0
	dueDate := today.AddDate(0, 0, 60)
	result := Evaluate(Plan{IntervalRule: "all_required"}, Schedule{
		DueHM:   &dueHM,
		DueDate: &dueDate,
	}, 101, 0, today)
	if result.Due {
		t.Fatalf("all_required must wait for every threshold, got %+v", result)
	}
	if result.Bucket != BucketScheduled {
		t.Fatalf("bucket = %q, want scheduled", result.Bucket)
	}
}

func TestEvaluateMeterUpcomingThresholds(t *testing.T) {
	today := dateOnly(time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC))
	dueKM := 1000.0
	upcomingKM := 800.0
	dueSoonKM := 950.0
	result := Evaluate(Plan{
		UpcomingKM: &upcomingKM,
		DueSoonKM:  &dueSoonKM,
	}, Schedule{DueKM: &dueKM}, 0, 960, today)
	if result.Bucket != BucketDueSoon {
		t.Fatalf("bucket = %q, want due_soon", result.Bucket)
	}
}
