package units

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
)

const (
	MaxPageLimit = 100
	DefaultLimit = 25
)

var (
	ErrNotFound          = errors.New("units: not found")
	ErrConflict          = errors.New("units: conflict")
	ErrValidation        = errors.New("units: validation failed")
	ErrInactive          = errors.New("units: unit is inactive")
	ErrReadingRegression = errors.New("units: reading below current value")
)

// ValidationError carries field-level messages to the HTTP layer.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "units: validation failed"
}

func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

type UnitModel struct {
	ID           int64     `json:"id"`
	Manufacturer string    `json:"manufacturer"`
	Model        string    `json:"model"`
	Category     string    `json:"category"`
	Notes        string    `json:"notes"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type EngineModel struct {
	ID             int64     `json:"id"`
	Manufacturer   string    `json:"manufacturer"`
	Model          string    `json:"model"`
	FuelType       string    `json:"fuel_type"`
	DisplacementCC *int      `json:"displacement_cc"`
	Notes          string    `json:"notes"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UnitModelSummary struct {
	ID           int64  `json:"id"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
}

type EngineModelSummary struct {
	ID           int64  `json:"id"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
}

type Unit struct {
	ID                 int64               `json:"id"`
	OdooUnitID         *int64              `json:"odoo_unit_id,omitempty"`
	Code               string              `json:"code"`
	UnitType           string              `json:"unit_type"`
	Merk               string              `json:"merk"`
	ClassUnit          string              `json:"class_unit"`
	LicensePlate       string              `json:"license_plate"`
	SiteLocation       string              `json:"site_location"`
	UnitModelID        *int64              `json:"unit_model_id,omitempty"`
	EngineModelID      *int64              `json:"engine_model_id,omitempty"`
	UnitModel          *UnitModelSummary   `json:"unit_model,omitempty"`
	EngineModel        *EngineModelSummary `json:"engine_model,omitempty"`
	SerialNumber       string              `json:"serial_number"`
	EngineSerialNumber string              `json:"engine_serial_number"`
	CurrentHM          float64             `json:"current_hm"`
	CurrentKM          float64             `json:"current_km"`
	HMUpdatedAt        *time.Time          `json:"hm_updated_at,omitempty"`
	KMUpdatedAt        *time.Time          `json:"km_updated_at,omitempty"`
	IsBreakdown        bool                `json:"is_breakdown"`
	IsActive           bool                `json:"is_active"`
	SyncedAt           *time.Time          `json:"synced_at,omitempty"`
}

type MeterReading struct {
	ID          int64     `json:"id"`
	UnitID      int64     `json:"unit_id"`
	ReadingType string    `json:"reading_type"`
	Value       float64   `json:"value"`
	RecordedAt  time.Time `json:"recorded_at"`
	SourceType  string    `json:"source_type"`
	SourceID    *int64    `json:"source_id,omitempty"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
}

type DashboardSummary struct {
	UnitsTotal         int64 `json:"units_total"`
	UnitsActive        int64 `json:"units_active"`
	UnitsBreakdown     int64 `json:"units_breakdown"`
	UnitModels         int64 `json:"unit_models"`
	EngineModels       int64 `json:"engine_models"`
	ReadingsLast30Days int64 `json:"readings_last_30_days"`
}

type UnitFilter struct {
	Query        string
	IsActive     *bool
	UnitModelID  *int64
	SiteLocation string
	Page         int
	Limit        int
	Sort         string
	Order        string
}

type ModelFilter struct {
	Query      string
	ActiveOnly bool
	Page       int
	Limit      int
	Sort       string
	Order      string
}

type UnitModelInput struct {
	Manufacturer string
	Model        string
	Category     string
	Notes        string
	IsActive     *bool
}

type UnitModelPatch struct {
	Manufacturer *string
	Model        *string
	Category     *string
	Notes        *string
	IsActive     *bool
}

type EngineModelInput struct {
	Manufacturer   string
	Model          string
	FuelType       string
	DisplacementCC *int
	Notes          string
	IsActive       *bool
}

type EngineModelPatch struct {
	Manufacturer   *string
	Model          *string
	FuelType       *string
	DisplacementCC *int
	Notes          *string
	IsActive       *bool
}

type UnitInput struct {
	Code               string
	UnitType           string
	Merk               string
	ClassUnit          string
	LicensePlate       string
	SiteLocation       string
	UnitModelID        *int64
	EngineModelID      *int64
	SerialNumber       string
	EngineSerialNumber string
	IsBreakdown        *bool
	IsActive           *bool
}

type UnitPatch struct {
	Code               *string
	UnitType           *string
	Merk               *string
	ClassUnit          *string
	LicensePlate       *string
	SiteLocation       *string
	UnitModelID        *int64
	EngineModelID      *int64
	SerialNumber       *string
	EngineSerialNumber *string
	IsBreakdown        *bool
	IsActive           *bool
}

type MeterReadingInput struct {
	ReadingType string
	Value       float64
	RecordedAt  time.Time
	Notes       string
}

type Repository interface {
	ListUnitModels(ctx context.Context, filter ModelFilter) ([]UnitModel, int, error)
	GetUnitModel(ctx context.Context, id int64) (UnitModel, error)
	CreateUnitModel(ctx context.Context, model UnitModel) (UnitModel, error)
	UpdateUnitModel(ctx context.Context, model UnitModel) (UnitModel, error)

	ListEngineModels(ctx context.Context, filter ModelFilter) ([]EngineModel, int, error)
	GetEngineModel(ctx context.Context, id int64) (EngineModel, error)
	CreateEngineModel(ctx context.Context, model EngineModel) (EngineModel, error)
	UpdateEngineModel(ctx context.Context, model EngineModel) (EngineModel, error)

	ListUnits(ctx context.Context, filter UnitFilter) ([]Unit, int, error)
	GetUnit(ctx context.Context, id int64) (Unit, error)
	CreateUnit(ctx context.Context, unit Unit) (Unit, error)
	UpdateUnit(ctx context.Context, unit Unit) (Unit, error)

	ListMeterReadings(ctx context.Context, unitID int64, limit int) ([]MeterReading, error)
	AddManualReading(ctx context.Context, unitID int64, input MeterReadingInput, actorRole string) (MeterReading, error)
	DashboardSummary(ctx context.Context) (DashboardSummary, error)

	ListMaintenanceHistory(ctx context.Context, unitID int64, filter HistoryFilter) ([]MaintenanceHistory, int, error)
	ListComponentHistory(ctx context.Context, unitID int64, filter ComponentHistoryFilter) ([]ComponentHistory, int, error)
	ListComponentIntervals(ctx context.Context, unitID int64) ([]ComponentInterval, error)
	ListUnitDowntime(ctx context.Context, unitID int64, filter DowntimeFilter) ([]DowntimeEvent, error)
	ListComponents(ctx context.Context, activeOnly bool) ([]Component, error)
}

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repo: repo, logger: logger}
}

func (s *Service) ListUnitModels(ctx context.Context, filter ModelFilter) ([]UnitModel, int, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	filter.Sort = strings.ToLower(strings.TrimSpace(filter.Sort))
	if !allowedUnitModelSort(filter.Sort) {
		filter.Sort = "manufacturer"
	}
	filter.Order = normalizeOrder(filter.Order)
	return s.repo.ListUnitModels(ctx, filter)
}

func (s *Service) CreateUnitModel(ctx context.Context, input UnitModelInput) (UnitModel, error) {
	manufacturer := strings.TrimSpace(input.Manufacturer)
	modelName := strings.TrimSpace(input.Model)
	category := strings.TrimSpace(input.Category)
	notes := strings.TrimSpace(input.Notes)
	if err := validateUnitModel(manufacturer, modelName, category, notes); err != nil {
		return UnitModel{}, err
	}
	return s.repo.CreateUnitModel(ctx, UnitModel{
		Manufacturer: manufacturer,
		Model:        modelName,
		Category:     category,
		Notes:        notes,
		IsActive:     boolValue(input.IsActive, true),
	})
}

func (s *Service) UpdateUnitModel(ctx context.Context, id int64, patch UnitModelPatch) (UnitModel, error) {
	existing, err := s.repo.GetUnitModel(ctx, id)
	if err != nil {
		return UnitModel{}, err
	}
	if patch.Manufacturer != nil {
		existing.Manufacturer = strings.TrimSpace(*patch.Manufacturer)
	}
	if patch.Model != nil {
		existing.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.Category != nil {
		existing.Category = strings.TrimSpace(*patch.Category)
	}
	if patch.Notes != nil {
		existing.Notes = strings.TrimSpace(*patch.Notes)
	}
	if patch.IsActive != nil {
		existing.IsActive = *patch.IsActive
	}
	if err := validateUnitModel(existing.Manufacturer, existing.Model, existing.Category, existing.Notes); err != nil {
		return UnitModel{}, err
	}
	return s.repo.UpdateUnitModel(ctx, existing)
}

func (s *Service) ListEngineModels(ctx context.Context, filter ModelFilter) ([]EngineModel, int, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	filter.Sort = strings.ToLower(strings.TrimSpace(filter.Sort))
	if !allowedEngineModelSort(filter.Sort) {
		filter.Sort = "manufacturer"
	}
	filter.Order = normalizeOrder(filter.Order)
	return s.repo.ListEngineModels(ctx, filter)
}

func (s *Service) CreateEngineModel(ctx context.Context, input EngineModelInput) (EngineModel, error) {
	manufacturer := strings.TrimSpace(input.Manufacturer)
	modelName := strings.TrimSpace(input.Model)
	fuelType := strings.TrimSpace(input.FuelType)
	notes := strings.TrimSpace(input.Notes)
	if err := validateEngineModel(manufacturer, modelName, fuelType, notes, input.DisplacementCC); err != nil {
		return EngineModel{}, err
	}
	return s.repo.CreateEngineModel(ctx, EngineModel{
		Manufacturer:   manufacturer,
		Model:          modelName,
		FuelType:       fuelType,
		DisplacementCC: input.DisplacementCC,
		Notes:          notes,
		IsActive:       boolValue(input.IsActive, true),
	})
}

func (s *Service) UpdateEngineModel(ctx context.Context, id int64, patch EngineModelPatch) (EngineModel, error) {
	existing, err := s.repo.GetEngineModel(ctx, id)
	if err != nil {
		return EngineModel{}, err
	}
	if patch.Manufacturer != nil {
		existing.Manufacturer = strings.TrimSpace(*patch.Manufacturer)
	}
	if patch.Model != nil {
		existing.Model = strings.TrimSpace(*patch.Model)
	}
	if patch.FuelType != nil {
		existing.FuelType = strings.TrimSpace(*patch.FuelType)
	}
	if patch.DisplacementCC != nil {
		existing.DisplacementCC = patch.DisplacementCC
	}
	if patch.Notes != nil {
		existing.Notes = strings.TrimSpace(*patch.Notes)
	}
	if patch.IsActive != nil {
		existing.IsActive = *patch.IsActive
	}
	if err := validateEngineModel(existing.Manufacturer, existing.Model, existing.FuelType, existing.Notes, existing.DisplacementCC); err != nil {
		return EngineModel{}, err
	}
	return s.repo.UpdateEngineModel(ctx, existing)
}

func (s *Service) ListUnits(ctx context.Context, filter UnitFilter) ([]Unit, int, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.SiteLocation = strings.TrimSpace(filter.SiteLocation)
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	if filter.Sort == "" {
		filter.Sort = "code"
	}
	if filter.Order == "" {
		filter.Order = "asc"
	}
	filter.Sort = strings.ToLower(filter.Sort)
	if !allowedSort(filter.Sort) {
		filter.Sort = "code"
	}
	filter.Order = normalizeOrder(filter.Order)
	return s.repo.ListUnits(ctx, filter)
}

func (s *Service) GetUnit(ctx context.Context, id int64) (Unit, error) {
	if id <= 0 {
		return Unit{}, &ValidationError{Fields: map[string]string{"id": "ID unit tidak valid"}}
	}
	return s.repo.GetUnit(ctx, id)
}

func (s *Service) CreateUnit(ctx context.Context, input UnitInput) (Unit, error) {
	unit := Unit{
		Code:               strings.TrimSpace(input.Code),
		UnitType:           strings.TrimSpace(input.UnitType),
		Merk:               strings.TrimSpace(input.Merk),
		ClassUnit:          strings.TrimSpace(input.ClassUnit),
		LicensePlate:       strings.TrimSpace(input.LicensePlate),
		SiteLocation:       strings.TrimSpace(input.SiteLocation),
		UnitModelID:        input.UnitModelID,
		EngineModelID:      input.EngineModelID,
		SerialNumber:       strings.TrimSpace(input.SerialNumber),
		EngineSerialNumber: strings.TrimSpace(input.EngineSerialNumber),
		IsBreakdown:        boolValue(input.IsBreakdown, false),
		IsActive:           boolValue(input.IsActive, true),
	}
	if err := s.validateUnitInput(ctx, unit); err != nil {
		return Unit{}, err
	}
	return s.repo.CreateUnit(ctx, unit)
}

func (s *Service) UpdateUnit(ctx context.Context, id int64, patch UnitPatch) (Unit, error) {
	unit, err := s.repo.GetUnit(ctx, id)
	if err != nil {
		return Unit{}, err
	}
	if patch.Code != nil {
		unit.Code = strings.TrimSpace(*patch.Code)
	}
	if patch.UnitType != nil {
		unit.UnitType = strings.TrimSpace(*patch.UnitType)
	}
	if patch.Merk != nil {
		unit.Merk = strings.TrimSpace(*patch.Merk)
	}
	if patch.ClassUnit != nil {
		unit.ClassUnit = strings.TrimSpace(*patch.ClassUnit)
	}
	if patch.LicensePlate != nil {
		unit.LicensePlate = strings.TrimSpace(*patch.LicensePlate)
	}
	if patch.SiteLocation != nil {
		unit.SiteLocation = strings.TrimSpace(*patch.SiteLocation)
	}
	if patch.UnitModelID != nil {
		if *patch.UnitModelID == 0 {
			unit.UnitModelID = nil
		} else {
			unit.UnitModelID = patch.UnitModelID
		}
	}
	if patch.EngineModelID != nil {
		if *patch.EngineModelID == 0 {
			unit.EngineModelID = nil
		} else {
			unit.EngineModelID = patch.EngineModelID
		}
	}
	if patch.SerialNumber != nil {
		unit.SerialNumber = strings.TrimSpace(*patch.SerialNumber)
	}
	if patch.EngineSerialNumber != nil {
		unit.EngineSerialNumber = strings.TrimSpace(*patch.EngineSerialNumber)
	}
	if patch.IsBreakdown != nil {
		unit.IsBreakdown = *patch.IsBreakdown
	}
	if patch.IsActive != nil {
		unit.IsActive = *patch.IsActive
	}
	if err := s.validateUnitInput(ctx, unit); err != nil {
		return Unit{}, err
	}
	return s.repo.UpdateUnit(ctx, unit)
}

func (s *Service) ListMeterReadings(ctx context.Context, unitID int64, limit int) ([]MeterReading, error) {
	if unitID <= 0 {
		return nil, &ValidationError{Fields: map[string]string{"unit_id": "ID unit tidak valid"}}
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	return s.repo.ListMeterReadings(ctx, unitID, limit)
}

func (s *Service) AddManualReading(ctx context.Context, unitID int64, input MeterReadingInput, actorRole string) (MeterReading, error) {
	if unitID <= 0 {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"unit_id": "ID unit tidak valid"}}
	}
	readingType := strings.ToLower(strings.TrimSpace(input.ReadingType))
	if readingType != "hm" && readingType != "km" {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"reading_type": "Tipe reading harus hm atau km"}}
	}
	if math.IsNaN(input.Value) || math.IsInf(input.Value, 0) || input.Value < 0 || input.Value > 999999999999.9 {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"value": "Nilai HM/KM tidak valid"}}
	}
	if input.RecordedAt.IsZero() {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"recorded_at": "Waktu reading wajib diisi"}}
	}
	if input.RecordedAt.After(time.Now().Add(5 * time.Minute)) {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"recorded_at": "Waktu reading tidak boleh di masa depan"}}
	}
	notes := strings.TrimSpace(input.Notes)
	if len(notes) > 2000 {
		return MeterReading{}, &ValidationError{Fields: map[string]string{"notes": "Catatan maksimal 2000 karakter"}}
	}

	return s.repo.AddManualReading(ctx, unitID, MeterReadingInput{
		ReadingType: readingType,
		Value:       math.Round(input.Value*10) / 10,
		RecordedAt:  input.RecordedAt.UTC(),
		Notes:       notes,
	}, actorRole)
}

func (s *Service) DashboardSummary(ctx context.Context) (DashboardSummary, error) {
	return s.repo.DashboardSummary(ctx)
}

func (s *Service) validateUnitInput(ctx context.Context, unit Unit) error {
	fields := map[string]string{}
	if unit.Code == "" {
		fields["code"] = "Kode unit wajib diisi"
	} else if len(unit.Code) > 100 {
		fields["code"] = "Kode unit maksimal 100 karakter"
	}
	if len(unit.UnitType) > 100 {
		fields["unit_type"] = "Tipe unit maksimal 100 karakter"
	}
	if len(unit.Merk) > 100 {
		fields["merk"] = "Merk maksimal 100 karakter"
	}
	if len(unit.ClassUnit) > 100 {
		fields["class_unit"] = "Kelas unit maksimal 100 karakter"
	}
	if len(unit.LicensePlate) > 50 {
		fields["license_plate"] = "Nomor polisi maksimal 50 karakter"
	}
	if len(unit.SiteLocation) > 200 {
		fields["site_location"] = "Lokasi maksimal 200 karakter"
	}
	if len(unit.SerialNumber) > 100 {
		fields["serial_number"] = "Serial number maksimal 100 karakter"
	}
	if len(unit.EngineSerialNumber) > 100 {
		fields["engine_serial_number"] = "Serial engine maksimal 100 karakter"
	}
	if unit.UnitModelID != nil {
		if *unit.UnitModelID <= 0 {
			fields["unit_model_id"] = "Unit model tidak valid"
		} else if _, err := s.repo.GetUnitModel(ctx, *unit.UnitModelID); err != nil {
			if errors.Is(err, ErrNotFound) {
				fields["unit_model_id"] = "Unit model tidak ditemukan"
			} else {
				return fmt.Errorf("validate unit model: %w", err)
			}
		}
	}
	if unit.EngineModelID != nil {
		if *unit.EngineModelID <= 0 {
			fields["engine_model_id"] = "Engine model tidak valid"
		} else if _, err := s.repo.GetEngineModel(ctx, *unit.EngineModelID); err != nil {
			if errors.Is(err, ErrNotFound) {
				fields["engine_model_id"] = "Engine model tidak ditemukan"
			} else {
				return fmt.Errorf("validate engine model: %w", err)
			}
		}
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateUnitModel(manufacturer, modelName, category, notes string) error {
	fields := map[string]string{}
	if manufacturer == "" {
		fields["manufacturer"] = "Manufacturer wajib diisi"
	} else if len(manufacturer) > 100 {
		fields["manufacturer"] = "Manufacturer maksimal 100 karakter"
	}
	if modelName == "" {
		fields["model"] = "Model wajib diisi"
	} else if len(modelName) > 100 {
		fields["model"] = "Model maksimal 100 karakter"
	}
	if len(category) > 100 {
		fields["category"] = "Kategori maksimal 100 karakter"
	}
	if len(notes) > 2000 {
		fields["notes"] = "Catatan maksimal 2000 karakter"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func validateEngineModel(manufacturer, modelName, fuelType, notes string, displacementCC *int) error {
	fields := map[string]string{}
	if manufacturer == "" {
		fields["manufacturer"] = "Manufacturer wajib diisi"
	} else if len(manufacturer) > 100 {
		fields["manufacturer"] = "Manufacturer maksimal 100 karakter"
	}
	if modelName == "" {
		fields["model"] = "Model wajib diisi"
	} else if len(modelName) > 100 {
		fields["model"] = "Model maksimal 100 karakter"
	}
	if len(fuelType) > 50 {
		fields["fuel_type"] = "Tipe bahan bakar maksimal 50 karakter"
	}
	if displacementCC != nil && (*displacementCC <= 0 || *displacementCC > 1000000) {
		fields["displacement_cc"] = "Displacement harus antara 1 dan 1.000.000 cc"
	}
	if len(notes) > 2000 {
		fields["notes"] = "Catatan maksimal 2000 karakter"
	}
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	return page, limit
}

func allowedSort(value string) bool {
	switch value {
	case "code", "unit_type", "model", "site_location", "current_hm", "current_km", "is_active", "is_breakdown":
		return true
	default:
		return false
	}
}

func allowedUnitModelSort(value string) bool {
	switch value {
	case "manufacturer", "model", "category", "is_active", "updated_at":
		return true
	default:
		return false
	}
}

func allowedEngineModelSort(value string) bool {
	switch value {
	case "manufacturer", "model", "fuel_type", "displacement_cc", "is_active", "updated_at":
		return true
	default:
		return false
	}
}

func normalizeOrder(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == "desc" {
		return "desc"
	}
	return "asc"
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
