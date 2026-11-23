package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
	"gsi-workorder/internal/units"
)

type fakeUnitService struct {
	createUnitInput   units.UnitInput
	createUnitErr     error
	readingErr        error
	listFilter        units.UnitFilter
	unitModelFilter   units.ModelFilter
	engineModelFilter units.ModelFilter
	historyFilter     units.HistoryFilter
	downtimeFilter    units.DowntimeFilter
	importFile        []byte
	importErr         error
	importResult      units.ImportResult
}

func (f *fakeUnitService) ListUnitModels(_ context.Context, filter units.ModelFilter) ([]units.UnitModel, int, error) {
	f.unitModelFilter = filter
	return []units.UnitModel{}, 0, nil
}
func (f *fakeUnitService) CreateUnitModel(context.Context, units.UnitModelInput) (units.UnitModel, error) {
	return units.UnitModel{ID: 1}, nil
}
func (f *fakeUnitService) UpdateUnitModel(context.Context, int64, units.UnitModelPatch) (units.UnitModel, error) {
	return units.UnitModel{ID: 1}, nil
}
func (f *fakeUnitService) ListEngineModels(_ context.Context, filter units.ModelFilter) ([]units.EngineModel, int, error) {
	f.engineModelFilter = filter
	return []units.EngineModel{}, 0, nil
}
func (f *fakeUnitService) CreateEngineModel(context.Context, units.EngineModelInput) (units.EngineModel, error) {
	return units.EngineModel{ID: 1}, nil
}
func (f *fakeUnitService) UpdateEngineModel(context.Context, int64, units.EngineModelPatch) (units.EngineModel, error) {
	return units.EngineModel{ID: 1}, nil
}
func (f *fakeUnitService) ListUnits(_ context.Context, filter units.UnitFilter) ([]units.Unit, int, error) {
	f.listFilter = filter
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
func (f *fakeUnitService) ImportUnits(_ context.Context, data []byte) (units.ImportResult, error) {
	f.importFile = data
	if f.importErr != nil {
		return units.ImportResult{}, f.importErr
	}
	return f.importResult, nil
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
func (f *fakeUnitService) ListMaintenanceHistory(_ context.Context, _ int64, filter units.HistoryFilter) ([]units.MaintenanceHistory, int, error) {
	f.historyFilter = filter
	return []units.MaintenanceHistory{}, 0, nil
}
func (f *fakeUnitService) ListComponentHistory(context.Context, int64, units.ComponentHistoryFilter) ([]units.ComponentHistory, int, error) {
	return []units.ComponentHistory{}, 0, nil
}
func (f *fakeUnitService) ListComponentIntervals(context.Context, int64) ([]units.ComponentInterval, error) {
	return []units.ComponentInterval{}, nil
}
func (f *fakeUnitService) UnitDowntime(_ context.Context, _ int64, filter units.DowntimeFilter) (units.UnitDowntime, error) {
	f.downtimeFilter = filter
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		return units.UnitDowntime{}, &units.ValidationError{Fields: map[string]string{"period": "Periode tidak valid"}}
	}
	return units.UnitDowntime{Events: []units.DowntimeEvent{}}, nil
}
func (f *fakeUnitService) ListComponents(context.Context, bool) ([]units.Component, error) {
	return []units.Component{}, nil
}
func (f *fakeUnitService) GetComponent(context.Context, int64) (units.Component, error) {
	return units.Component{ID: 1, Code: "CMP-1", Name: "Komponen"}, nil
}
func (f *fakeUnitService) CreateComponent(context.Context, units.ComponentInput) (units.Component, error) {
	return units.Component{ID: 1, Code: "CMP-1", Name: "Komponen"}, nil
}
func (f *fakeUnitService) UpdateComponent(context.Context, int64, units.ComponentPatch) (units.Component, error) {
	return units.Component{ID: 1, Code: "CMP-1", Name: "Komponen"}, nil
}
func (f *fakeUnitService) ListComponentInstallations(context.Context, int64, bool) ([]units.ComponentInstallation, error) {
	return []units.ComponentInstallation{}, nil
}
func (f *fakeUnitService) GetComponentInstallation(context.Context, int64, int64) (units.ComponentInstallation, error) {
	return units.ComponentInstallation{ID: 1, UnitID: 1}, nil
}
func (f *fakeUnitService) ListComponentLifetime(context.Context, int64, bool) ([]units.ComponentLifetime, error) {
	return []units.ComponentLifetime{}, nil
}
func (f *fakeUnitService) InstallComponent(context.Context, int64, units.ComponentInstallationInput, int64) (units.ComponentInstallation, error) {
	return units.ComponentInstallation{ID: 1, UnitID: 1}, nil
}
func (f *fakeUnitService) ReplaceComponent(context.Context, int64, int64, units.ComponentReplacementInput, int64) (units.ComponentInstallation, error) {
	return units.ComponentInstallation{ID: 2, UnitID: 1}, nil
}
func (f *fakeUnitService) TransferComponent(context.Context, int64, int64, units.ComponentTransferInput, int64) (units.ComponentInstallation, error) {
	return units.ComponentInstallation{ID: 2, UnitID: 2}, nil
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

func TestListUnitsInvalidPaginationFallsBack(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units?page=abc&limit=9999&sort=evil&order=sideways", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if unitService.listFilter.Page != 1 || unitService.listFilter.Limit != 100 {
		t.Fatalf("filter = %#v", unitService.listFilter)
	}
}

func TestListUnitModelsPassesSortAndPagination(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/unit-models?page=2&limit=10&sort=model&order=desc", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	filter := unitService.unitModelFilter
	if filter.Page != 2 || filter.Limit != 10 || filter.Sort != "model" || filter.Order != "desc" {
		t.Fatalf("filter = %#v", filter)
	}
}

func TestUnitHistoryParsesWitaPeriod(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleViewer, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units/1/history?from=2026-11-10&to=2026-11-10&status=done&maintenance_type=breakdown", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	filter := unitService.historyFilter
	if filter.From == nil || filter.To == nil {
		t.Fatalf("period = %#v", filter)
	}
	if filter.From.UTC().Format(time.RFC3339) != "2026-11-09T16:00:00Z" {
		t.Fatalf("from = %s", filter.From.UTC())
	}
	if filter.To.UTC().Format(time.RFC3339) != "2026-11-10T16:00:00Z" {
		t.Fatalf("to = %s", filter.To.UTC())
	}
	if filter.Status != "done" || filter.MaintenanceType != "breakdown" {
		t.Fatalf("filter = %#v", filter)
	}
}

func TestUnitDowntimeRejectsInvalidPeriod(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleViewer, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units/1/downtime?from=2026-11-10&to=2026-11-01", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestUnitsExportReturnsXLSX(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 9, Role: auth.RoleViewer, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units/export?q=LV-01", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "spreadsheetml.sheet") {
		t.Fatalf("content type = %q", contentType)
	}
	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("PK")) {
		t.Fatal("response body is not a zip-based workbook")
	}
	if unitService.listFilter.Query != "LV-01" {
		t.Fatalf("export filter = %+v", unitService.listFilter)
	}
}

func TestUnitImportTemplateIsAvailableToViewer(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 9, Role: auth.RoleViewer, IsActive: true}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units/import-template", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("PK")) {
		t.Fatal("template body is not a zip-based workbook")
	}
}

func TestUnitsImportRejectsMechanic(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 7, Role: auth.RoleMechanic, IsActive: true}, &fakeUnitService{})
	body, contentType := unitImportRequest(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units/import", body)
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", contentType)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestInstallComponentRequiresPlanner(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 9, Role: auth.RoleViewer, IsActive: true}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units/1/component-installations", strings.NewReader(`{"component_id":1,"serial_number":"SN-1"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestInstallComponentCreatesInstallation(t *testing.T) {
	unitService := &fakeUnitService{}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units/1/component-installations", strings.NewReader(`{"component_id":1,"serial_number":"SN-1"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data units.ComponentInstallation `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if response.Data.ID != 1 || response.Data.UnitID != 1 {
		t.Fatalf("body = %#v", response.Data)
	}
}

func TestComponentLifetimeIsAvailableToViewer(t *testing.T) {
	router := unitTestRouter(auth.User{ID: 9, Role: auth.RoleViewer, IsActive: true}, &fakeUnitService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/units/1/component-lifetime?active_only=true", nil)
	request.Header.Set("Authorization", "Bearer good")
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data []units.ComponentLifetime `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if response.Data == nil {
		t.Fatal("data is nil")
	}
}

func TestUnitsAllowAdminImport(t *testing.T) {
	unitService := &fakeUnitService{
		importResult: units.ImportResult{Total: 1, Created: 1, Rows: []units.ImportRowResult{
			{Row: 2, Code: "UNIT-1", Status: "created"},
		}},
	}
	router := unitTestRouter(auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}, unitService)
	body, contentType := unitImportRequest(t)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/units/import", body)
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", contentType)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if string(unitService.importFile) != "xlsx-bytes" {
		t.Fatalf("import file = %q", unitService.importFile)
	}
	var response struct {
		Data units.ImportResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if response.Data.Created != 1 {
		t.Fatalf("body = %#v", response.Data)
	}
}

func unitImportRequest(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "unit.xlsx")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("xlsx-bytes")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}
