package pm

import (
	"testing"
	"time"
)

func TestNormalizePlanRequiresExactlyOneTarget(t *testing.T) {
	unitID := int64(1)
	modelID := int64(2)
	interval := 100.0
	base := PlanInput{
		Name:          "PM Unit",
		UnitID:        &unitID,
		EffectiveFrom: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC),
		IntervalHM:    &interval,
	}
	if _, err := normalizePlanInput(base); err != nil {
		t.Fatalf("unit target should be valid: %v", err)
	}
	base.UnitModelID = &modelID
	if _, err := normalizePlanInput(base); err == nil {
		t.Fatal("two targets must be rejected")
	}
}

func TestNormalizePlanRequiresInterval(t *testing.T) {
	unitID := int64(1)
	_, err := normalizePlanInput(PlanInput{
		Name:          "PM Unit",
		UnitID:        &unitID,
		EffectiveFrom: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("a plan without any interval must be rejected")
	}
}

func TestNormalizePlanDefaults(t *testing.T) {
	unitID := int64(1)
	interval := 50.0
	plan, err := normalizePlanInput(PlanInput{
		Name:          "PM Unit",
		UnitID:        &unitID,
		EffectiveFrom: time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC),
		IntervalKM:    &interval,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "draft" || plan.IntervalRule != "first_reached" || plan.DefaultPriority != "medium" {
		t.Fatalf("unexpected defaults: %+v", plan)
	}
}
