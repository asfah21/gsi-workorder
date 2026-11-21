package pm

import (
	"strings"
	"time"
)

func normalizePlanInput(input PlanInput) (PlanInput, error) {
	fields := map[string]string{}
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.IntervalRule = strings.TrimSpace(input.IntervalRule)
	input.DefaultPriority = strings.TrimSpace(input.DefaultPriority)
	input.Status = strings.TrimSpace(input.Status)

	if len(input.Name) < 3 || len(input.Name) > 200 {
		fields["name"] = "Nama plan wajib 3-200 karakter."
	}
	if len(input.Description) > 2000 {
		fields["description"] = "Deskripsi maksimal 2000 karakter."
	}
	targets := 0
	if input.UnitID != nil && *input.UnitID > 0 {
		targets++
	}
	if input.UnitModelID != nil && *input.UnitModelID > 0 {
		targets++
	}
	if targets != 1 {
		fields["target"] = "Pilih tepat satu target: unit atau model."
	}
	if input.EffectiveFrom.IsZero() {
		fields["effective_from"] = "Tanggal efektif wajib diisi."
	}
	if input.EffectiveTo != nil && !input.EffectiveTo.IsZero() && input.EffectiveTo.Before(input.EffectiveFrom) {
		fields["effective_to"] = "Tanggal berakhir tidak boleh sebelum tanggal efektif."
	}
	if input.IntervalHM != nil && *input.IntervalHM <= 0 {
		fields["interval_hm"] = "Interval HM harus lebih dari 0."
	}
	if input.IntervalKM != nil && *input.IntervalKM <= 0 {
		fields["interval_km"] = "Interval KM harus lebih dari 0."
	}
	if input.IntervalDays != nil && *input.IntervalDays <= 0 {
		fields["interval_days"] = "Interval hari harus lebih dari 0."
	}
	if input.IntervalHM == nil && input.IntervalKM == nil && input.IntervalDays == nil {
		fields["interval"] = "Minimal satu interval HM, KM, atau hari wajib diisi."
	}
	if input.IntervalRule == "" {
		input.IntervalRule = "first_reached"
	}
	if input.IntervalRule != "first_reached" && input.IntervalRule != "all_required" {
		fields["interval_rule"] = "Aturan interval tidak valid."
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if input.Status != "draft" && input.Status != "active" && input.Status != "inactive" {
		fields["status"] = "Status plan tidak valid."
	}
	if input.DefaultPriority == "" {
		input.DefaultPriority = "medium"
	}
	if !validPriority(input.DefaultPriority) {
		fields["default_priority"] = "Prioritas default tidak valid."
	}
	if err := validateThresholds(input.UpcomingDays, input.DueSoonDays,
		input.UpcomingHM, input.DueSoonHM, "upcoming_days", "due_soon_days", "upcoming_hm", "due_soon_hm", fields); err != nil {
		return PlanInput{}, err
	}
	validateThresholds(input.UpcomingDays, input.DueSoonDays,
		input.UpcomingKM, input.DueSoonKM, "upcoming_days", "due_soon_days", "upcoming_km", "due_soon_km", fields)
	if len(fields) > 0 {
		return PlanInput{}, &ValidationError{Fields: fields}
	}

	checklist := make([]ChecklistInput, 0, len(input.Checklist))
	for index, item := range input.Checklist {
		item.Task = strings.TrimSpace(item.Task)
		item.Standard = strings.TrimSpace(item.Standard)
		if item.Sequence <= 0 {
			item.Sequence = index + 1
		}
		if item.Task == "" || len(item.Task) > 500 {
			fields["checklist"] = "Task checklist wajib 1-500 karakter."
		}
		if len(item.Standard) > 500 {
			fields["checklist"] = "Standard checklist maksimal 500 karakter."
		}
		checklist = append(checklist, item)
	}
	if len(fields) > 0 {
		return PlanInput{}, &ValidationError{Fields: fields}
	}
	input.Checklist = checklist
	return input, nil
}

func validateThresholds(upcomingDays, dueSoonDays *int, upcoming, dueSoon *float64, upcomingDaysKey, dueSoonDaysKey, upcomingKey, dueSoonKey string, fields map[string]string) error {
	if upcomingDays != nil && (*upcomingDays < 0 || *upcomingDays > 365) {
		fields[upcomingDaysKey] = "Harus antara 0 dan 365 hari."
	}
	if dueSoonDays != nil && (*dueSoonDays < 0 || *dueSoonDays > 365) {
		fields[dueSoonDaysKey] = "Harus antara 0 dan 365 hari."
	}
	if upcomingDays != nil && dueSoonDays != nil && *dueSoonDays > *upcomingDays {
		fields[dueSoonDaysKey] = "Due soon tidak boleh lebih besar dari upcoming."
	}
	if upcoming != nil && *upcoming < 0 {
		fields[upcomingKey] = "Ambang upcoming tidak valid."
	}
	if dueSoon != nil && *dueSoon < 0 {
		fields[dueSoonKey] = "Ambang due soon tidak valid."
	}
	if upcoming != nil && dueSoon != nil && *dueSoon > *upcoming {
		fields[dueSoonKey] = "Due soon tidak boleh lebih besar dari upcoming."
	}
	return nil
}

func validPriority(value string) bool {
	switch value {
	case "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func validStatus(value string) bool {
	return value == "draft" || value == "active" || value == "inactive"
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func dateOnlyIn(value time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	in := value.In(loc)
	return dateOnly(in)
}
