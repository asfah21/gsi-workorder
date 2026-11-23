package units

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	unitModel                UnitModel
	unitModelErr             error
	unitModelFilter          ModelFilter
	engineModel              EngineModel
	engineModelErr           error
	engineModelFilter        ModelFilter
	unitFilter               UnitFilter
	createdUnit              Unit
	createdUnitErr           error
	createdUnits             []Unit
	createUnitsErr           error
	existingCodes            map[string]struct{}
	reading                  MeterReading
	readingErr               error
	historyFilter            HistoryFilter
	downtimeEvents           []DowntimeEvent
	component                Component
	componentErr             error
	installation             ComponentInstallation
	installationErr          error
	createdInstallation      ComponentInstallation
	createdInstallationInput ComponentInstallationCreate
	closedInstallationID     int64
	closedInstallation       ComponentInstallationClose
	unitsByID                map[int64]Unit
}

func (f *fakeRepository) ListUnitModels(_ context.Context, filter ModelFilter) ([]UnitModel, int, error) {
	f.unitModelFilter = filter
	return nil, 0, nil
}
func (f *fakeRepository) GetUnitModel(context.Context, int64) (UnitModel, error) {
	return f.unitModel, f.unitModelErr
}
func (f *fakeRepository) CreateUnitModel(context.Context, UnitModel) (UnitModel, error) {
	return UnitModel{ID: 1}, nil
}
func (f *fakeRepository) UpdateUnitModel(context.Context, UnitModel) (UnitModel, error) {
	return UnitModel{ID: 1}, nil
}
func (f *fakeRepository) ListEngineModels(_ context.Context, filter ModelFilter) ([]EngineModel, int, error) {
	f.engineModelFilter = filter
	return nil, 0, nil
}
func (f *fakeRepository) GetEngineModel(context.Context, int64) (EngineModel, error) {
	return f.engineModel, f.engineModelErr
}
func (f *fakeRepository) CreateEngineModel(context.Context, EngineModel) (EngineModel, error) {
	return EngineModel{ID: 1}, nil
}
func (f *fakeRepository) UpdateEngineModel(context.Context, EngineModel) (EngineModel, error) {
	return EngineModel{ID: 1}, nil
}
func (f *fakeRepository) ListUnits(_ context.Context, filter UnitFilter) ([]Unit, int, error) {
	f.unitFilter = filter
	return nil, 0, nil
}
func (f *fakeRepository) GetUnit(_ context.Context, id int64) (Unit, error) {
	if f.unitsByID != nil {
		unit, ok := f.unitsByID[id]
		if !ok {
			return Unit{}, ErrNotFound
		}
		return unit, nil
	}
	return Unit{ID: 1, Code: "LV-01", IsActive: true}, nil
}
func (f *fakeRepository) CreateUnit(_ context.Context, unit Unit) (Unit, error) {
	f.createdUnit = unit
	if f.createdUnitErr != nil {
		return Unit{}, f.createdUnitErr
	}
	unit.ID = 1
	return unit, nil
}
func (f *fakeRepository) CreateUnits(_ context.Context, items []Unit) ([]Unit, error) {
	f.createdUnits = items
	if f.createUnitsErr != nil {
		return nil, f.createUnitsErr
	}
	for index := range items {
		items[index].ID = int64(index + 1)
	}
	return items, nil
}
func (f *fakeRepository) ExistingUnitCodes(context.Context, []string) (map[string]struct{}, error) {
	if f.existingCodes == nil {
		return map[string]struct{}{}, nil
	}
	return f.existingCodes, nil
}
func (f *fakeRepository) UpdateUnit(context.Context, Unit) (Unit, error) {
	return Unit{ID: 1}, nil
}
func (f *fakeRepository) ListMeterReadings(context.Context, int64, int) ([]MeterReading, error) {
	return nil, nil
}
func (f *fakeRepository) AddManualReading(context.Context, int64, MeterReadingInput, string) (MeterReading, error) {
	return f.reading, f.readingErr
}
func (f *fakeRepository) DashboardSummary(context.Context) (DashboardSummary, error) {
	return DashboardSummary{}, nil
}
func (f *fakeRepository) ListMaintenanceHistory(_ context.Context, _ int64, filter HistoryFilter) ([]MaintenanceHistory, int, error) {
	f.historyFilter = filter
	return nil, 0, nil
}
func (f *fakeRepository) ListComponentHistory(context.Context, int64, ComponentHistoryFilter) ([]ComponentHistory, int, error) {
	return nil, 0, nil
}
func (f *fakeRepository) ListComponentIntervals(context.Context, int64) ([]ComponentInterval, error) {
	return nil, nil
}
func (f *fakeRepository) ListUnitDowntime(context.Context, int64, DowntimeFilter) ([]DowntimeEvent, error) {
	return f.downtimeEvents, nil
}
func (f *fakeRepository) ListComponents(context.Context, bool) ([]Component, error) {
	return nil, nil
}
func (f *fakeRepository) GetComponent(context.Context, int64) (Component, error) {
	return f.component, f.componentErr
}
func (f *fakeRepository) CreateComponent(_ context.Context, component Component) (Component, error) {
	if f.componentErr != nil {
		return Component{}, f.componentErr
	}
	if component.ID == 0 {
		component.ID = 1
	}
	return component, nil
}
func (f *fakeRepository) UpdateComponent(_ context.Context, component Component) (Component, error) {
	return component, f.componentErr
}
func (f *fakeRepository) ListComponentInstallations(context.Context, int64, bool) ([]ComponentInstallation, error) {
	return nil, nil
}
func (f *fakeRepository) GetComponentInstallation(context.Context, int64) (ComponentInstallation, error) {
	return f.installation, f.installationErr
}
func (f *fakeRepository) CreateComponentInstallation(_ context.Context, input ComponentInstallationCreate) (ComponentInstallation, error) {
	f.createdInstallationInput = input
	return f.createdInstallation, f.installationErr
}
func (f *fakeRepository) CloseAndInstallComponent(_ context.Context, oldID int64, close ComponentInstallationClose, create ComponentInstallationCreate) (ComponentInstallation, error) {
	f.closedInstallationID = oldID
	f.closedInstallation = close
	f.createdInstallationInput = create
	return f.createdInstallation, f.installationErr
}
func (f *fakeRepository) ListComponentLifetime(context.Context, int64, bool) ([]ComponentLifetime, error) {
	return nil, nil
}

func TestCreateUnitNormalizesAndDefaults(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	unit, err := service.CreateUnit(context.Background(), UnitInput{
		Code: "  LV-01  ",
		Merk: "  Toyota ",
	})
	if err != nil {
		t.Fatalf("CreateUnit() error = %v", err)
	}
	if unit.Code != "LV-01" || unit.Merk != "Toyota" {
		t.Fatalf("unit = %#v", unit)
	}
	if !unit.IsActive {
		t.Fatal("new unit should be active by default")
	}
}

func TestCreateUnitRequiresCode(t *testing.T) {
	service := NewService(&fakeRepository{}, nil)
	_, err := service.CreateUnit(context.Background(), UnitInput{})
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	if validation.Fields["code"] == "" {
		t.Fatalf("fields = %#v", validation.Fields)
	}
}

func TestCreateUnitRejectsMissingModel(t *testing.T) {
	repo := &fakeRepository{unitModelErr: ErrNotFound}
	service := NewService(repo, nil)
	modelID := int64(99)
	_, err := service.CreateUnit(context.Background(), UnitInput{
		Code:        "LV-01",
		UnitModelID: &modelID,
	})
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["unit_model_id"] == "" {
		t.Fatalf("error = %v, want unit_model_id validation", err)
	}
}

func TestAddManualReadingValidatesInput(t *testing.T) {
	service := NewService(&fakeRepository{}, nil)
	_, err := service.AddManualReading(context.Background(), 1, MeterReadingInput{
		ReadingType: "rpm",
		Value:       10,
		RecordedAt:  time.Now(),
	}, "mechanic")
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["reading_type"] == "" {
		t.Fatalf("error = %v, want reading_type validation", err)
	}
}

func TestAddManualReadingRejectsFutureTimestamp(t *testing.T) {
	service := NewService(&fakeRepository{}, nil)
	_, err := service.AddManualReading(context.Background(), 1, MeterReadingInput{
		ReadingType: "hm",
		Value:       10,
		RecordedAt:  time.Now().Add(time.Hour),
	}, "mechanic")
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Fields["recorded_at"] == "" {
		t.Fatalf("error = %v, want recorded_at validation", err)
	}
}

func TestAddManualReadingPassesLowerValueToRepository(t *testing.T) {
	repo := &fakeRepository{
		reading: MeterReading{ID: 5, UnitID: 1, ReadingType: "hm", Value: 90},
	}
	service := NewService(repo, nil)
	reading, err := service.AddManualReading(context.Background(), 1, MeterReadingInput{
		ReadingType: "hm",
		Value:       90,
		RecordedAt:  time.Now(),
		Notes:       "koreksi",
	}, "admin")
	if err != nil {
		t.Fatalf("AddManualReading() error = %v", err)
	}
	if reading.ID != 5 {
		t.Fatalf("reading = %#v", reading)
	}
}

func TestListUnitsNormalizesInvalidListParams(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	_, _, err := service.ListUnits(context.Background(), UnitFilter{
		Page:  0,
		Limit: 999,
		Sort:  "'; drop table units; --",
		Order: "sideways",
	})
	if err != nil {
		t.Fatalf("ListUnits() error = %v", err)
	}
	if repo.unitFilter.Page != 1 || repo.unitFilter.Limit != MaxPageLimit {
		t.Fatalf("pagination = %#v", repo.unitFilter)
	}
	if repo.unitFilter.Sort != "code" || repo.unitFilter.Order != "asc" {
		t.Fatalf("sort = %#v", repo.unitFilter)
	}
}

func TestListUnitModelsNormalizesInvalidListParams(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	_, _, err := service.ListUnitModels(context.Background(), ModelFilter{
		Page:  0,
		Limit: 999,
		Sort:  "not-a-column",
		Order: "sideways",
	})
	if err != nil {
		t.Fatalf("ListUnitModels() error = %v", err)
	}
	if repo.unitModelFilter.Page != 1 || repo.unitModelFilter.Limit != MaxPageLimit {
		t.Fatalf("pagination = %#v", repo.unitModelFilter)
	}
	if repo.unitModelFilter.Sort != "manufacturer" || repo.unitModelFilter.Order != "asc" {
		t.Fatalf("sort = %#v", repo.unitModelFilter)
	}
}

func TestListEngineModelsNormalizesInvalidListParams(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, nil)
	_, _, err := service.ListEngineModels(context.Background(), ModelFilter{
		Page:  0,
		Limit: 999,
		Sort:  "not-a-column",
		Order: "sideways",
	})
	if err != nil {
		t.Fatalf("ListEngineModels() error = %v", err)
	}
	if repo.engineModelFilter.Page != 1 || repo.engineModelFilter.Limit != MaxPageLimit {
		t.Fatalf("pagination = %#v", repo.engineModelFilter)
	}
	if repo.engineModelFilter.Sort != "manufacturer" || repo.engineModelFilter.Order != "asc" {
		t.Fatalf("sort = %#v", repo.engineModelFilter)
	}
}
