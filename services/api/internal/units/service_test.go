package units

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	unitModel      UnitModel
	unitModelErr   error
	engineModel    EngineModel
	engineModelErr error
	createdUnit    Unit
	createdUnitErr error
	reading        MeterReading
	readingErr     error
}

func (f *fakeRepository) ListUnitModels(context.Context, string, bool, int, int) ([]UnitModel, int, error) {
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
func (f *fakeRepository) ListEngineModels(context.Context, string, bool, int, int) ([]EngineModel, int, error) {
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
func (f *fakeRepository) ListUnits(context.Context, UnitFilter) ([]Unit, int, error) {
	return nil, 0, nil
}
func (f *fakeRepository) GetUnit(context.Context, int64) (Unit, error) {
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
