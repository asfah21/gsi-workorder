// Package pmcalc contains the pure PM interval/threshold rules shared by the
// PM API, worker, and work-order completion guard.
package pmcalc

import "time"

const (
	BucketScheduled = "scheduled"
	BucketUpcoming  = "upcoming"
	BucketDueSoon   = "due_soon"
	BucketDue       = "due"
	BucketOverdue   = "overdue"
)

type Plan struct {
	IntervalRule        string
	UpcomingDays        *int
	DueSoonDays         *int
	DefaultUpcomingDays *int
	DefaultDueSoonDays  *int
	UpcomingHM          *float64
	UpcomingKM          *float64
	DueSoonHM           *float64
	DueSoonKM           *float64
}

type Schedule struct {
	DueHM   *float64
	DueKM   *float64
	DueDate *time.Time
}

type Result struct {
	Bucket    string
	Due       bool
	Overdue   bool
	DaysToDue *int
}

// Evaluate computes the current PM bucket from a schedule's due thresholds and
// the unit's current meter readings. today must be a date-only value (UTC
// midnight) in the application timezone.
func Evaluate(plan Plan, schedule Schedule, currentHM, currentKM float64, today time.Time) Result {
	upcomingDays := 30
	if plan.DefaultUpcomingDays != nil {
		upcomingDays = *plan.DefaultUpcomingDays
	}
	if plan.UpcomingDays != nil {
		upcomingDays = *plan.UpcomingDays
	}
	dueSoonDays := 7
	if plan.DefaultDueSoonDays != nil {
		dueSoonDays = *plan.DefaultDueSoonDays
	}
	if plan.DueSoonDays != nil {
		dueSoonDays = *plan.DueSoonDays
	}

	states := make([]string, 0, 3)
	var daysToDue *int
	if schedule.DueDate != nil {
		due := dateOnly(*schedule.DueDate)
		today = dateOnly(today)
		switch {
		case today.After(due):
			states = append(states, BucketOverdue)
		case today.Equal(due):
			states = append(states, BucketDue)
		default:
			days := int(due.Sub(today).Hours() / 24)
			daysToDue = &days
			switch {
			case days <= dueSoonDays:
				states = append(states, BucketDueSoon)
			case days <= upcomingDays:
				states = append(states, BucketUpcoming)
			default:
				states = append(states, BucketScheduled)
			}
		}
	}

	if schedule.DueHM != nil {
		states = append(states, meterState(currentHM, *schedule.DueHM, plan.UpcomingHM, plan.DueSoonHM))
	}
	if schedule.DueKM != nil {
		states = append(states, meterState(currentKM, *schedule.DueKM, plan.UpcomingKM, plan.DueSoonKM))
	}

	return combine(plan.IntervalRule, states, daysToDue)
}

func meterState(current, due float64, upcoming, dueSoon *float64) string {
	switch {
	case current > due:
		return BucketOverdue
	case current >= due:
		return BucketDue
	case dueSoon != nil && current >= *dueSoon:
		return BucketDueSoon
	case upcoming != nil && current >= *upcoming:
		return BucketUpcoming
	default:
		return BucketScheduled
	}
}

func combine(rule string, states []string, daysToDue *int) Result {
	if len(states) == 0 {
		return Result{Bucket: BucketScheduled, DaysToDue: daysToDue}
	}

	selected := BucketScheduled
	if rule == "all_required" {
		allDue := true
		allOverdue := true
		allSoon := true
		allUpcoming := true
		for _, state := range states {
			if state != BucketDue && state != BucketOverdue {
				allDue = false
			}
			if state != BucketOverdue {
				allOverdue = false
			}
			if state != BucketDueSoon && state != BucketDue && state != BucketOverdue {
				allSoon = false
			}
			if state != BucketUpcoming && state != BucketDueSoon && state != BucketDue && state != BucketOverdue {
				allUpcoming = false
			}
		}
		switch {
		case allOverdue:
			selected = BucketOverdue
		case allDue:
			selected = BucketDue
		case allSoon:
			selected = BucketDueSoon
		case allUpcoming:
			selected = BucketUpcoming
		}
	} else {
		rank := map[string]int{
			BucketScheduled: 0,
			BucketUpcoming:  1,
			BucketDueSoon:   2,
			BucketDue:       3,
			BucketOverdue:   4,
		}
		for _, state := range states {
			if rank[state] > rank[selected] {
				selected = state
			}
		}
	}

	return Result{
		Bucket:    selected,
		Due:       selected == BucketDue || selected == BucketOverdue,
		Overdue:   selected == BucketOverdue,
		DaysToDue: daysToDue,
	}
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
