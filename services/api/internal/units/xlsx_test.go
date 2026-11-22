package units

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWriteUnitImportTemplateHasExpectedHeaders(t *testing.T) {
	var buffer bytes.Buffer
	err := WriteUnitImportTemplate(&buffer,
		[]UnitModel{{ID: 1, Manufacturer: "CAT", Model: "320"}},
		[]EngineModel{{ID: 2, Manufacturer: "Cummins", Model: "6BT"}},
	)
	if err != nil {
		t.Fatalf("WriteUnitImportTemplate() error = %v", err)
	}

	file, err := excelize.OpenReader(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	defer func() { _ = file.Close() }()

	rows, err := file.GetRows(UnitXLSXSheet)
	if err != nil {
		t.Fatalf("read template sheet: %v", err)
	}
	if len(rows) == 0 {
		t.Fatal("template has no header row")
	}
	if !headersMatch(rows[0], unitImportHeaders) {
		t.Fatalf("template headers = %v, want %v", rows[0], unitImportHeaders)
	}
	guideIndex, err := file.GetSheetIndex(UnitImportGuideSheet)
	if err != nil || guideIndex < 0 {
		t.Fatalf("template guide sheet index = %d, err = %v", guideIndex, err)
	}
}

func TestImportUnitsCreatesAllValidRows(t *testing.T) {
	rows := [][]string{
		unitImportHeaders,
		{"UNIT-1", "Excavator", "CAT", "Heavy", "B 1234 ABC", "Site A", "", "", "SNR-1", "ENR-1", "Tidak", "Aktif"},
		{"UNIT-2", "Dump truck", "Komatsu", "Heavy", "B 5678 XYZ", "Site B", "", "", "SNR-2", "ENR-2", "Ya", "Nonaktif"},
	}

	repo := &fakeRepository{}
	service := NewService(repo, nil)
	result, err := service.ImportUnits(context.Background(), buildUnitXLSX(t, rows))
	if err != nil {
		t.Fatalf("ImportUnits() error = %v", err)
	}
	if result.Total != 2 || result.Created != 2 || result.Failed != 0 {
		t.Fatalf("ImportUnits() result = %+v", result)
	}
	if len(repo.createdUnits) != 2 {
		t.Fatalf("created units = %d, want 2", len(repo.createdUnits))
	}
	if repo.createdUnits[0].Code != "UNIT-1" || repo.createdUnits[1].Code != "UNIT-2" {
		t.Fatalf("unexpected created codes: %+v", repo.createdUnits)
	}
	if repo.createdUnits[1].IsActive {
		t.Fatal("second unit should be inactive")
	}
	if !repo.createdUnits[1].IsBreakdown {
		t.Fatal("second unit should be marked breakdown")
	}
}

func TestImportUnitsRejectsInvalidHeader(t *testing.T) {
	rows := [][]string{
		{"Kode unit", "Tipe unit", "Merk", "Kelas unit", "No. polisi", "Lokasi", "Unit model ID", "Engine model ID", "Serial number unit", "Serial number engine", "Breakdown", "Status"},
		{"UNIT-1", "Excavator", "CAT", "Heavy", "B 1", "Site A", "", "", "", "", "Tidak", "Aktif"},
	}

	service := NewService(&fakeRepository{}, nil)
	_, err := service.ImportUnits(context.Background(), buildUnitXLSX(t, rows))
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("ImportUnits() error = %v, want ValidationError", err)
	}
}

func TestImportUnitsCancelsWholeFileOnDuplicateOrExistingCode(t *testing.T) {
	rows := [][]string{
		unitImportHeaders,
		{"UNIT-1", "Excavator", "CAT", "Heavy", "B 1", "Site A", "", "", "", "", "Tidak", "Aktif"},
		{"UNIT-1", "Excavator", "CAT", "Heavy", "B 2", "Site B", "", "", "", "", "Tidak", "Aktif"},
		{"UNIT-2", "Dump truck", "Komatsu", "Heavy", "B 3", "Site C", "", "", "", "", "Tidak", "Aktif"},
	}

	repo := &fakeRepository{existingCodes: map[string]struct{}{"UNIT-2": {}}}
	service := NewService(repo, nil)
	result, err := service.ImportUnits(context.Background(), buildUnitXLSX(t, rows))
	if err != nil {
		t.Fatalf("ImportUnits() error = %v", err)
	}
	if result.Created != 0 || result.Failed != 3 {
		t.Fatalf("ImportUnits() result = %+v", result)
	}
	if len(repo.createdUnits) != 0 {
		t.Fatalf("created units = %d, want 0", len(repo.createdUnits))
	}
}

func TestImportUnitsRejectsInvalidBooleanWithoutInsert(t *testing.T) {
	rows := [][]string{
		unitImportHeaders,
		{"UNIT-1", "Excavator", "CAT", "Heavy", "B 1", "Site A", "", "", "", "", "Mungkin", "Aktif"},
	}

	repo := &fakeRepository{}
	service := NewService(repo, nil)
	result, err := service.ImportUnits(context.Background(), buildUnitXLSX(t, rows))
	if err != nil {
		t.Fatalf("ImportUnits() error = %v", err)
	}
	if result.Created != 0 || result.Failed != 1 {
		t.Fatalf("ImportUnits() result = %+v", result)
	}
	if len(repo.createdUnits) != 0 {
		t.Fatalf("created units = %d, want 0", len(repo.createdUnits))
	}
}

func buildUnitXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	file := excelize.NewFile()
	if err := file.SetSheetName(file.GetSheetName(0), UnitXLSXSheet); err != nil {
		t.Fatalf("set sheet name: %v", err)
	}
	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				t.Fatalf("resolve cell: %v", err)
			}
			if err := file.SetCellStr(UnitXLSXSheet, cell, value); err != nil {
				t.Fatalf("write cell: %v", err)
			}
		}
	}
	var buffer bytes.Buffer
	if err := file.Write(&buffer); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close workbook: %v", err)
	}
	return buffer.Bytes()
}
