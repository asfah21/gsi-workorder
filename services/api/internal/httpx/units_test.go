package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
	"gsi-workorder/internal/units"
)

type fakeUnitService struct {
	createUnitInput units.UnitInput
	createUnitErr   error
	readingErr      error
}

func (f *fakeUnitService) ListUnitModels(context.Context, string, bool, int, int) ([]units.UnitModel, int, error) {
	return []units.UnitModel{}, 0, nil
}
func (f *fakeUnitService) CreateUnitModel(context.Context, units.UnitModelInput) (units.UnitModel, error) {
	return units.UnitModel{ID: 1}, nil
}
func (f *fakeUnitService) UpdateUnitModel(context.Context, int64, units.UnitModelPatch) (units.UnitModel, error) {
	return units.UnitModel{ID: 1}, nil
}
func (f *fakeUnitService) ListEngineModels(context.Context, string, bool, int, int) ([]units.EngineModel, int, error) {
	return []units.EngineModel{}, 0, nil
}
func (f *fakeUnitService) CreateEngineModel(context.Context, units.EngineModelInput) (units.EngineModel, error) {
	return units.EngineModel{ID: 1}, nil
}
func (f *fakeUnitService) UpdateEngineModel(context.Context, int64, units.EngineModelPatch) (units.EngineModel, error) {
	return units.EngineModel{ID: 1}, nil
}
func (f *fakeUnitService) ListUnits(context.Context, units.UnitFilter) ([]units.Unit, int, error) {
	return []units.Unit{{ID: 1, Code: "LV-01"}}, 1, nil
}
func (f *fakeUnitService) GetUnit(context.Context, int64) (units.Unit, error) {
	return units.Unit{ID: 1, Code: "LV-01"}, nil
}
func (f *fakeUnitService) CreateUnit(_ context.Context, input units.UnitInput) (units.Unit, error) {
	f.createUnitInput = input
	if f.createUnitErr != nil {
		return units.Unit{}, f.createUnitErr
	}
	return units.Unit{ID: 1, Code: input.Code, IsActive: true}, nil
}
func (f *fakeUnitService) UpdateUnit(context.Context, int64, units.UnitPatch) (units.Unit, error) {
	return units.Unit{ID: 1, Code: "LV-01"}, nil
}
func (f *fakeUnitService) ListMeterReadings(context.Context, int64, int) ([]units.MeterReading, error) {
	return []units.MeterReading{}, nil
}
func (f *fakeUnitService) AddManualReading(context.Context, int64, units.MeterReadingInput, string) (units.MeterReading, error) {
	if f.readingErr != nil {
		return units.MeterReading{}, f.readingErr
	}
	return units.MeterReading{ID: 1, UnitID: 1, ReadingType: "hm", Value: 12.5}, nil
}
func (f *fakeUnitService) DashboardSummary(context.Context) (units.DashboardSummary, error) {
	return units.DashboardSummary{UnitsTotal: 1}, nil
}

func unitTestRouter(authUser auth.User, unitService UnitService) *gin.Engine {
	service := &fakeAuthService{authUser: authUser}
	return NewRouterWithUnits(testLogger(), CheckFunc(func(context.Context) error { return nil }), service, unitService)
}

func TestUnitsRequireAuthentication(t *testing.T) {
	router := NewRouterWithUnits(testLogger(), CheckFunc(func(context.Context) error { return nil }),
		&fakeAuthService{authErr: auth.ErrInvalidToken}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units", nil)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestUnitsRejectMechanicWrite(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 7, Role: auth.RoleMechanic, IsActive: true}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units", strings.NewReader(`{"code":"LV-01"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestUnitsAllowAdminCreate(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units", strings.NewReader(`{"code":"LV-01","unit_type":"light_vehicle"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if unitService.createUnitInput.Code != "LV-01" {
		t.Fatalf("input = %#v", unitService.createUnitInput)
	}
	var body struct {
		Data units.Unit `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.Code != "LV-01" {
		t.Fatalf("body = %#v", body)
	}
}

func TestUnitValidationMapsTo422(t *testing.T) {
	unitService := &fakeUnitService{
		createUnitErr: &units.ValidationError{Fields: map[string]string{"code": "wajib"}},
	}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units", strings.NewReader(`{"code":""}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestMechanicCanAddMeterReading(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 7, Role: auth.RoleMechanic, IsActive: true}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units/1/meter-readings",
		strings.NewReader(`{"reading_type":"hm","value":12.5,"recorded_at":"2026-11-20T10:00:00+08:00"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
