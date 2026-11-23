package units

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

const (
	componentClassMajor = "major"
	componentClassMinor = "minor"

	installationStatusActive      = "active"
	installationStatusRemoved     = "removed"
	installationStatusTransferred = "transferred"
	installationStatusScrapped    = "scrapped"
)

// Component is the component master row. Lifetime values are optional defaults;
// an installation can override them.
type Component struct {
	ID                  int64    `json:"id"`
	Code                string   `json:"code"`
	Name                string   `json:"name"`
	Category            string   `json:"category"`
	ComponentClass      string   `json:"component_class"`
	DefaultLifetimeHM   *float64 `json:"default_lifetime_hm,omitempty"`
	DefaultLifetimeDays *int     `json:"default_lifetime_days,omitempty"`
	DefaultLifetimeKM   *float64 `json:"default_lifetime_km,omitempty"`
	IsActive            bool     `json:"is_active"`
}

type ComponentInput struct {
	Code                string
	Name                string
	Category            string
	ComponentClass      string
	DefaultLifetimeHM   *float64
	DefaultLifetimeDays *int
	DefaultLifetimeKM   *float64
	IsActive            *bool
}

type ComponentPatch struct {
	Code                *string
	Name                *string
	Category            *string
	ComponentClass      *string
	DefaultLifetimeHM   *float64
	DefaultLifetimeDays *int
	DefaultLifetimeKM   *float64
	IsActive            *bool
}

// ComponentInstallation is one physical component installed on a unit. A
// transfer closes the old row and creates a new row so the full history remains
// visible.
type ComponentInstallation struct {
	ID             int64      `json:"id"`
	UnitID         int64      `json:"unit_id"`
	UnitCode       string     `json:"unit_code"`
	ComponentID    int64      `json:"component_id"`
	ComponentCode  string     `json:"component_code"`
	ComponentName  string     `json:"component_name"`
	ComponentClass string     `json:"component_class"`
	SerialNumber   string     `json:"serial_number"`
	InstalledAt    time.Time  `json:"installed_at"`
	InstalledHM    float64    `json:"installed_hm"`
	InstalledKM    float64    `json:"installed_km"`
	RemovedAt      *time.Time `json:"removed_at,omitempty"`
	RemovedHM      *float64   `json:"removed_hm,omitempty"`
	RemovedKM      *float64   `json:"removed_km,omitempty"`
	LifetimeHM     *float64   `json:"lifetime_hm,omitempty"`
	LifetimeDays   *int       `json:"lifetime_days,omitempty"`
	LifetimeKM     *float64   `json:"lifetime_km,omitempty"`
	Status         string     `json:"status"`
	SourceWOID     *int64     `json:"source_wo_id,omitempty"`
	Notes          string     `json:"notes"`
	CreatedBy      *int64     `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ComponentLifetime struct {
	InstallationID int64      `json:"installation_id"`
	UnitID         int64      `json:"unit_id"`
	UnitCode       string     `json:"unit_code"`
	UnitCurrentHM  float64    `json:"unit_current_hm"`
	UnitCurrentKM  float64    `json:"unit_current_km"`
	ComponentID    int64      `json:"component_id"`
	ComponentCode  string     `json:"component_code"`
	ComponentName  string     `json:"component_name"`
	Category       string     `json:"category"`
	ComponentClass string     `json:"component_class"`
	SerialNumber   string     `json:"serial_number"`
	InstalledAt    time.Time  `json:"installed_at"`
	InstalledHM    float64    `json:"installed_hm"`
	InstalledKM    float64    `json:"installed_km"`
	RemovedAt      *time.Time `json:"removed_at,omitempty"`
	RemovedHM      *float64   `json:"removed_hm,omitempty"`
	RemovedKM      *float64   `json:"removed_km,omitempty"`
	Status         string     `json:"status"`
	LifetimeHM     *float64   `json:"lifetime_hm,omitempty"`
	LifetimeDays   *int       `json:"lifetime_days,omitempty"`
	LifetimeKM     *float64   `json:"lifetime_km,omitempty"`
	UsedHM         float64    `json:"used_hm"`
	UsedKM         float64    `json:"used_km"`
	UsedDays       float64    `json:"used_days"`
	RemainingHM    *float64   `json:"remaining_hm,omitempty"`
	RemainingDays  *float64   `json:"remaining_days,omitempty"`
	RemainingKM    *float64   `json:"remaining_km,omitempty"`
	LifetimeState  string     `json:"lifetime_state"`
}

type ComponentInstallationInput struct {
	ComponentID  int64
	SerialNumber string
	InstalledAt  *time.Time
	InstalledHM  *float64
	InstalledKM  *float64
	LifetimeHM   *float64
	LifetimeDays *int
	LifetimeKM   *float64
	SourceWOID   *int64
	Notes        string
}

type ComponentReplacementInput struct {
	SerialNumber string
	InstalledAt  *time.Time
	InstalledHM  *float64
	InstalledKM  *float64
	LifetimeHM   *float64
	LifetimeDays *int
	LifetimeKM   *float64
	SourceWOID   *int64
	Notes        string
	RemovedAt    *time.Time
	RemovedHM    *float64
	RemovedKM    *float64
}

type ComponentTransferInput struct {
	ToUnitID    int64
	InstalledAt *time.Time
	InstalledHM *float64
	InstalledKM *float64
	SourceWOID  *int64
	Notes       string
	RemovedAt   *time.Time
	RemovedHM   *float64
	RemovedKM   *float64
}

// ComponentInstallationCreate is the normalized insert payload used by the
// repository. It contains only concrete values so the database layer does not
// repeat defaulting rules.
type ComponentInstallationCreate struct {
	UnitID       int64
	ComponentID  int64
	SerialNumber string
	InstalledAt  time.Time
	InstalledHM  float64
	InstalledKM  float64
	LifetimeHM   *float64
	LifetimeDays *int
	LifetimeKM   *float64
	SourceWOID   *int64
	Notes        string
	CreatedBy    *int64
}

type ComponentInstallationClose struct {
	InstallationID int64
	Status         string
	RemovedAt      time.Time
	RemovedHM      *float64
	RemovedKM      *float64
}

func (s *Service) GetComponent(ctx context.Context, id int64) (Component, error) {
	if id <= 0 {
		return Component{}, &ValidationError{Fields: map[string]string{"id": "ID komponen tidak valid"}}
	}
	return s.repo.GetComponent(ctx, id)
}

func (s *Service) CreateComponent(ctx context.Context, input ComponentInput) (Component, error) {
	component := Component{
		Code:                input.Code,
		Name:                input.Name,
		Category:            input.Category,
		ComponentClass:      input.ComponentClass,
		DefaultLifetimeHM:   input.DefaultLifetimeHM,
		DefaultLifetimeDays: input.DefaultLifetimeDays,
		DefaultLifetimeKM:   input.DefaultLifetimeKM,
		IsActive:            boolValue(input.IsActive, true),
	}
	if err := normalizeComponent(&component); err != nil {
		return Component{}, err
	}
	return s.repo.CreateComponent(ctx, component)
}

func (s *Service) UpdateComponent(ctx context.Context, id int64, patch ComponentPatch) (Component, error) {
	existing, err := s.GetComponent(ctx, id)
	if err != nil {
		return Component{}, err
	}
	if patch.Code != nil {
		existing.Code = *patch.Code
	}
	if patch.Name != nil {
		existing.Name = *patch.Name
	}
	if patch.Category != nil {
		existing.Category = *patch.Category
	}
	if patch.ComponentClass != nil {
		existing.ComponentClass = *patch.ComponentClass
	}
	if patch.DefaultLifetimeHM != nil {
		existing.DefaultLifetimeHM = patch.DefaultLifetimeHM
	}
	if patch.DefaultLifetimeDays != nil {
		existing.DefaultLifetimeDays = patch.DefaultLifetimeDays
	}
	if patch.DefaultLifetimeKM != nil {
		existing.DefaultLifetimeKM = patch.DefaultLifetimeKM
	}
	if patch.IsActive != nil {
		existing.IsActive = *patch.IsActive
	}
	if err := normalizeComponent(&existing); err != nil {
		return Component{}, err
	}
	return s.repo.UpdateComponent(ctx, existing)
}

func (s *Service) ListComponentInstallations(ctx context.Context, unitID int64, activeOnly bool) ([]ComponentInstallation, error) {
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return nil, err
	}
	return s.repo.ListComponentInstallations(ctx, unitID, activeOnly)
}

func (s *Service) GetComponentInstallation(ctx context.Context, unitID, installationID int64) (ComponentInstallation, error) {
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return ComponentInstallation{}, err
	}
	if installationID <= 0 {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{"installation_id": "ID instalasi komponen tidak valid"}}
	}
	item, err := s.repo.GetComponentInstallation(ctx, installationID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	if item.UnitID != unitID {
		return ComponentInstallation{}, ErrNotFound
	}
	return item, nil
}

func (s *Service) ListComponentLifetime(ctx context.Context, unitID int64, activeOnly bool) ([]ComponentLifetime, error) {
	if err := s.ensureUnit(ctx, unitID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListComponentLifetime(ctx, unitID, activeOnly)
	if err != nil {
		return nil, err
	}
	for index := range items {
		items[index].LifetimeState = componentLifetimeState(
			items[index].Status,
			items[index].UsedHM,
			items[index].UsedKM,
			items[index].UsedDays,
			items[index].LifetimeHM,
			items[index].LifetimeKM,
			items[index].LifetimeDays,
		)
	}
	return items, nil
}

func (s *Service) InstallComponent(ctx context.Context, unitID int64, input ComponentInstallationInput, actorID int64) (ComponentInstallation, error) {
	create, err := s.prepareComponentInstallation(ctx, unitID, input.ComponentID, input.SerialNumber,
		input.InstalledAt, input.InstalledHM, input.InstalledKM, input.LifetimeHM,
		input.LifetimeDays, input.LifetimeKM, input.SourceWOID, input.Notes, actorID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	return s.repo.CreateComponentInstallation(ctx, create)
}

func (s *Service) ReplaceComponent(ctx context.Context, unitID, installationID int64, input ComponentReplacementInput, actorID int64) (ComponentInstallation, error) {
	old, err := s.activeInstallation(ctx, unitID, installationID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	close, err := s.prepareComponentClose(ctx, unitID, old, installationStatusRemoved,
		input.RemovedAt, input.RemovedHM, input.RemovedKM)
	if err != nil {
		return ComponentInstallation{}, err
	}
	create, err := s.prepareComponentInstallation(ctx, unitID, old.ComponentID, input.SerialNumber,
		input.InstalledAt, input.InstalledHM, input.InstalledKM, input.LifetimeHM,
		input.LifetimeDays, input.LifetimeKM, input.SourceWOID, input.Notes, actorID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	if create.InstalledAt.Before(close.RemovedAt) {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"installed_at": "Tanggal pemasangan baru tidak boleh sebelum tanggal pelepasan",
		}}
	}
	return s.repo.CloseAndInstallComponent(ctx, old.ID, close, create)
}

func (s *Service) TransferComponent(ctx context.Context, unitID, installationID int64, input ComponentTransferInput, actorID int64) (ComponentInstallation, error) {
	old, err := s.activeInstallation(ctx, unitID, installationID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	if input.ToUnitID <= 0 {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{"to_unit_id": "Unit tujuan tidak valid"}}
	}
	if input.ToUnitID == unitID {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{"to_unit_id": "Unit tujuan harus berbeda dari unit asal"}}
	}
	close, err := s.prepareComponentClose(ctx, unitID, old, installationStatusTransferred,
		input.RemovedAt, input.RemovedHM, input.RemovedKM)
	if err != nil {
		return ComponentInstallation{}, err
	}
	create, err := s.prepareComponentInstallation(ctx, input.ToUnitID, old.ComponentID, old.SerialNumber,
		input.InstalledAt, input.InstalledHM, input.InstalledKM, old.LifetimeHM,
		old.LifetimeDays, old.LifetimeKM, input.SourceWOID, input.Notes, actorID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	if create.InstalledAt.Before(close.RemovedAt) {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"installed_at": "Tanggal pemasangan di unit tujuan tidak boleh sebelum tanggal pelepasan",
		}}
	}
	return s.repo.CloseAndInstallComponent(ctx, old.ID, close, create)
}

func (s *Service) activeInstallation(ctx context.Context, unitID, installationID int64) (ComponentInstallation, error) {
	if installationID <= 0 {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{"installation_id": "ID instalasi komponen tidak valid"}}
	}
	old, err := s.repo.GetComponentInstallation(ctx, installationID)
	if err != nil {
		return ComponentInstallation{}, err
	}
	if old.UnitID != unitID {
		return ComponentInstallation{}, ErrNotFound
	}
	if old.Status != installationStatusActive {
		return ComponentInstallation{}, &ValidationError{Fields: map[string]string{
			"installation_id": "Instalasi komponen tidak aktif",
		}}
	}
	return old, nil
}

func (s *Service) prepareComponentInstallation(
	ctx context.Context,
	unitID, componentID int64,
	serialNumber string,
	installedAt *time.Time,
	installedHM, installedKM, lifetimeHM *float64,
	lifetimeDays *int,
	lifetimeKM *float64,
	sourceWOID *int64,
	notes string,
	actorID int64,
) (ComponentInstallationCreate, error) {
	unit, err := s.repo.GetUnit(ctx, unitID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ComponentInstallationCreate{}, &ValidationError{Fields: map[string]string{"unit_id": "Unit tidak ditemukan"}}
		}
		return ComponentInstallationCreate{}, err
	}
	if !unit.IsActive {
		return ComponentInstallationCreate{}, ErrInactive
	}
	if componentID <= 0 {
		return ComponentInstallationCreate{}, &ValidationError{Fields: map[string]string{"component_id": "Komponen tidak valid"}}
	}
	component, err := s.repo.GetComponent(ctx, componentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ComponentInstallationCreate{}, &ValidationError{Fields: map[string]string{"component_id": "Komponen tidak ditemukan"}}
		}
		return ComponentInstallationCreate{}, err
	}
	if !component.IsActive {
		return ComponentInstallationCreate{}, &ValidationError{Fields: map[string]string{"component_id": "Komponen tidak aktif"}}
	}

	fields := map[string]string{}
	serialNumber = strings.TrimSpace(serialNumber)
	if serialNumber == "" {
		fields["serial_number"] = "Serial number komponen wajib diisi"
	} else if len(serialNumber) > 100 {
		fields["serial_number"] = "Serial number komponen maksimal 100 karakter"
	}
	installed := time.Now().UTC()
	if installedAt != nil {
		installed = installedAt.UTC()
	}
	if installed.IsZero() {
		fields["installed_at"] = "Tanggal pemasangan wajib diisi"
	} else if installed.After(time.Now().Add(5 * time.Minute)) {
		fields["installed_at"] = "Tanggal pemasangan tidak boleh di masa depan"
	}
	hm := unit.CurrentHM
	if installedHM != nil {
		hm = *installedHM
	}
	if !validNonNegative(hm) {
		fields["installed_hm"] = "HM pemasangan tidak valid"
	}
	km := unit.CurrentKM
	if installedKM != nil {
		km = *installedKM
	}
	if !validNonNegative(km) {
		fields["installed_km"] = "KM pemasangan tidak valid"
	}
	if lifetimeHM == nil {
		lifetimeHM = component.DefaultLifetimeHM
	}
	if lifetimeDays == nil {
		lifetimeDays = component.DefaultLifetimeDays
	}
	if lifetimeKM == nil {
		lifetimeKM = component.DefaultLifetimeKM
	}
	if lifetimeHM != nil && !validNonNegative(*lifetimeHM) {
		fields["lifetime_hm"] = "Lifetime HM tidak valid"
	}
	if lifetimeDays != nil && *lifetimeDays < 0 {
		fields["lifetime_days"] = "Lifetime hari tidak valid"
	}
	if lifetimeKM != nil && !validNonNegative(*lifetimeKM) {
		fields["lifetime_km"] = "Lifetime KM tidak valid"
	}
	if sourceWOID != nil && *sourceWOID <= 0 {
		fields["source_wo_id"] = "Work order sumber tidak valid"
	}
	notes = strings.TrimSpace(notes)
	if len(notes) > 2000 {
		fields["notes"] = "Catatan maksimal 2000 karakter"
	}
	if len(fields) > 0 {
		return ComponentInstallationCreate{}, &ValidationError{Fields: fields}
	}

	var createdBy *int64
	if actorID > 0 {
		createdBy = &actorID
	}
	return ComponentInstallationCreate{
		UnitID:       unitID,
		ComponentID:  componentID,
		SerialNumber: serialNumber,
		InstalledAt:  installed,
		InstalledHM:  roundOne(hm),
		InstalledKM:  roundOne(km),
		LifetimeHM:   roundOptionalOne(lifetimeHM),
		LifetimeDays: lifetimeDays,
		LifetimeKM:   roundOptionalOne(lifetimeKM),
		SourceWOID:   sourceWOID,
		Notes:        notes,
		CreatedBy:    createdBy,
	}, nil
}

func (s *Service) prepareComponentClose(
	ctx context.Context,
	unitID int64,
	old ComponentInstallation,
	status string,
	removedAt *time.Time,
	removedHM, removedKM *float64,
) (ComponentInstallationClose, error) {
	unit, err := s.repo.GetUnit(ctx, unitID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ComponentInstallationClose{}, &ValidationError{Fields: map[string]string{"unit_id": "Unit tidak ditemukan"}}
		}
		return ComponentInstallationClose{}, err
	}
	removed := time.Now().UTC()
	if removedAt != nil {
		removed = removedAt.UTC()
	}
	fields := map[string]string{}
	if removed.After(time.Now().Add(5 * time.Minute)) {
		fields["removed_at"] = "Tanggal pelepasan tidak boleh di masa depan"
	}
	if removed.Before(old.InstalledAt) {
		fields["removed_at"] = "Tanggal pelepasan tidak boleh sebelum tanggal pemasangan"
	}
	hm := unit.CurrentHM
	if removedHM != nil {
		hm = *removedHM
	}
	if !validNonNegative(hm) || hm < old.InstalledHM {
		fields["removed_hm"] = "HM pelepasan tidak valid atau lebih kecil dari HM pemasangan"
	}
	km := unit.CurrentKM
	if removedKM != nil {
		km = *removedKM
	}
	if !validNonNegative(km) || km < old.InstalledKM {
		fields["removed_km"] = "KM pelepasan tidak valid atau lebih kecil dari KM pemasangan"
	}
	if len(fields) > 0 {
		return ComponentInstallationClose{}, &ValidationError{Fields: fields}
	}
	return ComponentInstallationClose{
		InstallationID: old.ID,
		Status:         status,
		RemovedAt:      removed,
		RemovedHM:      floatPointer(roundOne(hm)),
		RemovedKM:      floatPointer(roundOne(km)),
	}, nil
}

func normalizeComponent(component *Component) error {
	fields := map[string]string{}
	component.Code = strings.TrimSpace(component.Code)
	component.Name = strings.TrimSpace(component.Name)
	component.Category = strings.TrimSpace(component.Category)
	component.ComponentClass = strings.ToLower(strings.TrimSpace(component.ComponentClass))
	if component.Code == "" {
		fields["code"] = "Kode komponen wajib diisi"
	} else if len(component.Code) > 100 {
		fields["code"] = "Kode komponen maksimal 100 karakter"
	}
	if component.Name == "" {
		fields["name"] = "Nama komponen wajib diisi"
	} else if len(component.Name) > 200 {
		fields["name"] = "Nama komponen maksimal 200 karakter"
	}
	if len(component.Category) > 100 {
		fields["category"] = "Kategori komponen maksimal 100 karakter"
	}
	if component.ComponentClass != "" && component.ComponentClass != componentClassMajor && component.ComponentClass != componentClassMinor {
		fields["component_class"] = "Kelas komponen harus major atau minor"
	}
	if component.DefaultLifetimeHM != nil {
		if !validNonNegative(*component.DefaultLifetimeHM) {
			fields["default_lifetime_hm"] = "Lifetime HM default tidak valid"
		} else {
			component.DefaultLifetimeHM = floatPointer(roundOne(*component.DefaultLifetimeHM))
		}
	}
	if component.DefaultLifetimeDays != nil && *component.DefaultLifetimeDays < 0 {
		fields["default_lifetime_days"] = "Lifetime hari default tidak valid"
	}
	if component.DefaultLifetimeKM != nil {
		if !validNonNegative(*component.DefaultLifetimeKM) {
			fields["default_lifetime_km"] = "Lifetime KM default tidak valid"
		} else {
			component.DefaultLifetimeKM = floatPointer(roundOne(*component.DefaultLifetimeKM))
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func componentLifetimeState(status string, usedHM, usedKM, usedDays float64, lifetimeHM, lifetimeKM *float64, lifetimeDays *int) string {
	if status != installationStatusActive {
		return "inactive"
	}
	if lifetimeHM == nil && lifetimeKM == nil && lifetimeDays == nil {
		return "unset"
	}
	// Default thresholds: due at 10% remaining or 7 days; upcoming at 20% or
	// 30 days. They are kept in Go so they can be adjusted without a schema
	// migration once PAM confirms its replacement policy.
	overdue := false
	due := false
	upcoming := false
	if lifetimeHM != nil {
		remaining := *lifetimeHM - usedHM
		switch {
		case remaining <= 0:
			overdue = true
		case remaining <= *lifetimeHM*0.1:
			due = true
		case remaining <= *lifetimeHM*0.2:
			upcoming = true
		}
	}
	if lifetimeKM != nil {
		remaining := *lifetimeKM - usedKM
		switch {
		case remaining <= 0:
			overdue = true
		case remaining <= *lifetimeKM*0.1:
			due = true
		case remaining <= *lifetimeKM*0.2:
			upcoming = true
		}
	}
	if lifetimeDays != nil {
		remaining := float64(*lifetimeDays) - usedDays
		switch {
		case remaining <= 0:
			overdue = true
		case remaining <= 7:
			due = true
		case remaining <= 30:
			upcoming = true
		}
	}
	switch {
	case overdue:
		return "overdue"
	case due:
		return "due"
	case upcoming:
		return "upcoming"
	default:
		return "safe"
	}
}

func validNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 999999999999.9
}

func roundOne(value float64) float64 {
	return math.Round(value*10) / 10
}

func roundOptionalOne(value *float64) *float64 {
	if value == nil {
		return nil
	}
	return floatPointer(roundOne(*value))
}

func floatPointer(value float64) *float64 {
	return &value
}
