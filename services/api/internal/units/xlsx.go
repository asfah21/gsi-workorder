package units

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	UnitXLSXSheet          = "Unit"
	UnitImportGuideSheet   = "Panduan"
	UnitImportMaxRows      = 1000
	UnitImportMaxFileBytes = 5 << 20 // 5 MiB
)

var unitImportHeaders = []string{
	"Kode unit",
	"Tipe unit",
	"Merk",
	"Kelas unit",
	"No. polisi",
	"Lokasi",
	"Unit model ID",
	"Engine model ID",
	"Serial number unit",
	"Serial number engine",
	"Breakdown",
	"Aktif",
}

// ImportRowResult describes the outcome of one spreadsheet row.
type ImportRowResult struct {
	Row     int    `json:"row"`
	Code    string `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ImportResult is the summary returned by a unit import.
type ImportResult struct {
	Total   int               `json:"total"`
	Created int               `json:"created"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Rows    []ImportRowResult `json:"rows"`
}

type parsedImportRow struct {
	number  int
	code    string
	input   UnitInput
	message string
}

// ImportUnits parses an XLSX upload, validates every row, and inserts all
// valid rows in one database transaction. A single invalid row rejects the
// whole file so the master data is never left partially imported.
func (s *Service) ImportUnits(ctx context.Context, data []byte) (ImportResult, error) {
	result := ImportResult{Rows: make([]ImportRowResult, 0)}
	if len(data) == 0 {
		return result, &ValidationError{Fields: map[string]string{"file": "File XLSX kosong"}}
	}
	if len(data) > UnitImportMaxFileBytes {
		return result, &ValidationError{Fields: map[string]string{"file": "Ukuran file maksimal 5 MB"}}
	}

	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return result, &ValidationError{Fields: map[string]string{"file": "File bukan XLSX yang valid"}}
	}
	defer func() { _ = file.Close() }()

	sheet := file.GetSheetName(0)
	if sheet == "" {
		return result, &ValidationError{Fields: map[string]string{"file": "Workbook tidak memiliki sheet"}}
	}
	rows, err := file.GetRows(sheet)
	if err != nil {
		return result, &ValidationError{Fields: map[string]string{"file": "Sheet tidak dapat dibaca"}}
	}
	if len(rows) == 0 {
		return result, &ValidationError{Fields: map[string]string{"file": "Sheet kosong"}}
	}
	if !headersMatch(rows[0], unitImportHeaders) {
		return result, &ValidationError{Fields: map[string]string{
			"file": "Header tidak sesuai template. Gunakan tombol Unduh template XLSX.",
		}}
	}

	dataRows := rows[1:]
	for len(dataRows) > 0 && rowEmpty(dataRows[len(dataRows)-1]) {
		dataRows = dataRows[:len(dataRows)-1]
	}
	if len(dataRows) > UnitImportMaxRows {
		return result, &ValidationError{Fields: map[string]string{
			"file": fmt.Sprintf("Maksimal %d baris data per file", UnitImportMaxRows),
		}}
	}
	result.Total = len(dataRows)

	parsed := make([]parsedImportRow, 0, len(dataRows))
	for index, row := range dataRows {
		parsed = append(parsed, parseUnitImportRow(index+2, row))
	}

	seenCodes := make(map[string]int, len(parsed))
	for index := range parsed {
		item := &parsed[index]
		if rowEmpty(dataRows[index]) {
			item.message = "Baris kosong"
			continue
		}
		if item.message != "" {
			continue
		}
		key := strings.ToLower(item.code)
		if firstRow, exists := seenCodes[key]; exists {
			item.message = fmt.Sprintf("Kode unit duplikat dengan baris %d", firstRow)
			continue
		}
		seenCodes[key] = item.number
	}

	candidateUnits := make([]Unit, 0, len(parsed))
	candidateIndexes := make([]int, 0, len(parsed))
	codes := make([]string, 0, len(parsed))
	for index := range parsed {
		item := &parsed[index]
		if item.message != "" {
			continue
		}
		unit := unitFromInput(item.input)
		if err := s.validateUnitInput(ctx, unit); err != nil {
			item.message = validationMessage(err)
			continue
		}
		candidateUnits = append(candidateUnits, unit)
		candidateIndexes = append(candidateIndexes, index)
		codes = append(codes, unit.Code)
	}

	if len(codes) > 0 {
		existing, err := s.repo.ExistingUnitCodes(ctx, codes)
		if err != nil {
			return ImportResult{Rows: result.Rows}, fmt.Errorf("check existing unit codes: %w", err)
		}
		for position, index := range candidateIndexes {
			code := candidateUnits[position].Code
			if _, exists := existing[code]; exists {
				parsed[index].message = "Kode unit sudah ada di database"
			}
		}
	}

	hasFailure := false
	for _, item := range parsed {
		if item.message != "" && item.message != "Baris kosong" {
			hasFailure = true
			break
		}
	}
	if hasFailure {
		for index := range parsed {
			if parsed[index].message == "" {
				parsed[index].message = "Impor dibatalkan karena ada baris tidak valid"
			}
		}
	} else {
		insertUnits := make([]Unit, 0, len(candidateUnits))
		insertIndexes := make([]int, 0, len(candidateIndexes))
		for position, index := range candidateIndexes {
			insertUnits = append(insertUnits, candidateUnits[position])
			insertIndexes = append(insertIndexes, index)
		}
		created, err := s.repo.CreateUnits(ctx, insertUnits)
		if err != nil {
			return ImportResult{Rows: result.Rows}, err
		}
		if len(created) != len(insertIndexes) {
			return ImportResult{Rows: result.Rows}, fmt.Errorf("create units: unexpected result count")
		}
		for position, index := range insertIndexes {
			parsed[index].code = created[position].Code
		}
	}

	for _, item := range parsed {
		row := ImportRowResult{Row: item.number, Code: item.code}
		switch {
		case item.message == "Baris kosong":
			row.Status = "skipped"
			row.Message = item.message
			result.Skipped++
		case item.message != "":
			row.Status = "failed"
			row.Message = item.message
			result.Failed++
		default:
			row.Status = "created"
			result.Created++
		}
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func parseUnitImportRow(number int, row []string) parsedImportRow {
	item := parsedImportRow{number: number, code: strings.TrimSpace(rowValue(row, 0))}
	if item.code == "" {
		item.message = "Kode unit wajib diisi"
	}

	unitModelID, err := parseOptionalID(rowValue(row, 6))
	if err != nil {
		item.message = "Unit model ID tidak valid"
	}
	engineModelID, err := parseOptionalID(rowValue(row, 7))
	if err != nil {
		item.message = "Engine model ID tidak valid"
	}
	breakdown, err := parseImportBool(rowValue(row, 10), false)
	if err != nil {
		item.message = "Nilai Breakdown harus Ya atau Tidak"
	}
	active, err := parseImportBool(rowValue(row, 11), true)
	if err != nil {
		item.message = "Nilai Aktif harus Aktif atau Nonaktif"
	}

	item.input = UnitInput{
		Code:               item.code,
		UnitType:           strings.TrimSpace(rowValue(row, 1)),
		Merk:               strings.TrimSpace(rowValue(row, 2)),
		ClassUnit:          strings.TrimSpace(rowValue(row, 3)),
		LicensePlate:       strings.TrimSpace(rowValue(row, 4)),
		SiteLocation:       strings.TrimSpace(rowValue(row, 5)),
		UnitModelID:        unitModelID,
		EngineModelID:      engineModelID,
		SerialNumber:       strings.TrimSpace(rowValue(row, 8)),
		EngineSerialNumber: strings.TrimSpace(rowValue(row, 9)),
		IsBreakdown:        &breakdown,
		IsActive:           &active,
	}
	return item
}

func rowValue(row []string, index int) string {
	if index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func rowEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func headersMatch(row, expected []string) bool {
	if len(row) < len(expected) {
		return false
	}
	for index, header := range expected {
		if normalizeHeader(row[index]) != header {
			return false
		}
	}
	return true
}

func normalizeHeader(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(value, "\ufeff"))
}

func parseOptionalID(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, fmt.Errorf("invalid id")
	}
	return &value, nil
}

func parseImportBool(raw string, fallback bool) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return fallback, nil
	case "1", "true", "ya", "y", "aktif", "yes", "checked":
		return true, nil
	case "0", "false", "tidak", "t", "nonaktif", "no", "unchecked":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean")
	}
}

func validationMessage(err error) string {
	var validation *ValidationError
	if errors.As(err, &validation) {
		fields := make([]string, 0, len(validation.Fields))
		for key, message := range validation.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", key, message))
		}
		if len(fields) > 0 {
			return strings.Join(fields, "; ")
		}
	}
	return err.Error()
}

// WriteUnitImportTemplate renders the canonical upload template.
func WriteUnitImportTemplate(w io.Writer, unitModels []UnitModel, engineModels []EngineModel) error {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	sheet := UnitXLSXSheet
	if err := file.SetSheetName(file.GetSheetName(0), sheet); err != nil {
		return fmt.Errorf("set unit sheet name: %w", err)
	}
	if err := writeXLSXHeader(file, sheet, unitImportHeaders); err != nil {
		return err
	}
	example := []string{
		"CONTOH-01", "Excavator", "CAT", "Heavy", "B 1234 ABC", "Site A",
		"", "", "SNR-0001", "ENR-0001", "Tidak", "Aktif",
	}
	if err := writeXLSXRow(file, sheet, 2, example); err != nil {
		return err
	}
	if err := finishXLSXSheet(file, sheet, len(unitImportHeaders)); err != nil {
		return err
	}
	if err := writeXLSXGuide(file, unitModels, engineModels); err != nil {
		return err
	}
	file.SetActiveSheet(0)
	if err := file.Write(w); err != nil {
		return fmt.Errorf("write import template: %w", err)
	}
	return nil
}

// WriteUnitsXLSX renders the current unit list as an Excel workbook.
func WriteUnitsXLSX(w io.Writer, items []Unit, unitModels []UnitModel, engineModels []EngineModel) error {
	file := excelize.NewFile()
	defer func() { _ = file.Close() }()

	sheet := UnitXLSXSheet
	if err := file.SetSheetName(file.GetSheetName(0), sheet); err != nil {
		return fmt.Errorf("set unit sheet name: %w", err)
	}
	if err := writeXLSXHeader(file, sheet, unitImportHeaders); err != nil {
		return err
	}
	for index, item := range items {
		row := []string{
			item.Code,
			item.UnitType,
			item.Merk,
			item.ClassUnit,
			item.LicensePlate,
			item.SiteLocation,
			idString(item.UnitModelID),
			idString(item.EngineModelID),
			item.SerialNumber,
			item.EngineSerialNumber,
			yesNo(item.IsBreakdown),
			activeLabel(item.IsActive),
		}
		if err := writeXLSXRow(file, sheet, index+2, row); err != nil {
			return err
		}
	}
	if err := finishXLSXSheet(file, sheet, len(unitImportHeaders)); err != nil {
		return err
	}
	if err := writeXLSXGuide(file, unitModels, engineModels); err != nil {
		return err
	}
	file.SetActiveSheet(0)
	if err := file.Write(w); err != nil {
		return fmt.Errorf("write units xlsx: %w", err)
	}
	return nil
}

func writeXLSXHeader(file *excelize.File, sheet string, headers []string) error {
	style, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return fmt.Errorf("create xlsx header style: %w", err)
	}
	for index, header := range headers {
		cell, err := excelize.CoordinatesToCellName(index+1, 1)
		if err != nil {
			return fmt.Errorf("resolve xlsx header cell: %w", err)
		}
		if err := file.SetCellStr(sheet, cell, header); err != nil {
			return fmt.Errorf("write xlsx header: %w", err)
		}
		if err := file.SetCellStyle(sheet, cell, cell, style); err != nil {
			return fmt.Errorf("style xlsx header: %w", err)
		}
	}
	return nil
}

func writeXLSXRow(file *excelize.File, sheet string, rowNumber int, values []string) error {
	for index, value := range values {
		cell, err := excelize.CoordinatesToCellName(index+1, rowNumber)
		if err != nil {
			return fmt.Errorf("resolve xlsx cell: %w", err)
		}
		if err := file.SetCellStr(sheet, cell, value); err != nil {
			return fmt.Errorf("write xlsx cell: %w", err)
		}
	}
	return nil
}

func finishXLSXSheet(file *excelize.File, sheet string, columns int) error {
	if err := file.SetColWidth(sheet, "A", "L", 18); err != nil {
		return fmt.Errorf("set xlsx column width: %w", err)
	}
	if columns > 0 {
		lastCell, err := excelize.CoordinatesToCellName(columns, 1)
		if err != nil {
			return fmt.Errorf("resolve xlsx filter range: %w", err)
		}
		if err := file.AutoFilter(sheet, "A1:"+lastCell, []excelize.AutoFilterOptions{}); err != nil {
			return fmt.Errorf("set xlsx autofilter: %w", err)
		}
	}
	_ = file.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection:   []excelize.Selection{{Pane: "bottomLeft"}},
	})
	return nil
}

func writeXLSXGuide(file *excelize.File, unitModels []UnitModel, engineModels []EngineModel) error {
	if _, err := file.NewSheet(UnitImportGuideSheet); err != nil {
		return fmt.Errorf("create guide sheet: %w", err)
	}
	rows := [][]string{
		{"Kolom", "Keterangan"},
		{"Kode unit", "Wajib, maksimal 100 karakter, dan harus unik."},
		{"Tipe unit / Merk / Kelas unit / No. polisi / Lokasi", "Opsional. Isi sesuai master unit."},
		{"Unit model ID / Engine model ID", "Opsional. Isi ID numerik dari daftar di bawah atau kosongkan."},
		{"Serial number unit / Serial number engine", "Opsional, maksimal 100 karakter."},
		{"Breakdown", "Ya atau Tidak. Kosongkan untuk Tidak."},
		{"Aktif", "Aktif atau Nonaktif. Kosongkan untuk Aktif."},
		{"Batas baris", fmt.Sprintf("Maksimal %d baris data per file.", UnitImportMaxRows)},
		{"Catatan", "Jika satu baris tidak valid, seluruh file tidak akan diimpor."},
	}
	rowNumber := 1
	for _, row := range rows {
		if err := writeXLSXRow(file, UnitImportGuideSheet, rowNumber, row); err != nil {
			return err
		}
		rowNumber++
	}

	rowNumber++
	modelRows := [][]string{{"Unit model ID", "Nama model"}}
	for _, model := range unitModels {
		modelRows = append(modelRows, []string{strconv.FormatInt(model.ID, 10), model.Manufacturer + " " + model.Model})
	}
	for _, row := range modelRows {
		if err := writeXLSXRow(file, UnitImportGuideSheet, rowNumber, row); err != nil {
			return err
		}
		rowNumber++
	}

	rowNumber++
	engineRows := [][]string{{"Engine model ID", "Nama model"}}
	for _, model := range engineModels {
		engineRows = append(engineRows, []string{strconv.FormatInt(model.ID, 10), model.Manufacturer + " " + model.Model})
	}
	for _, row := range engineRows {
		if err := writeXLSXRow(file, UnitImportGuideSheet, rowNumber, row); err != nil {
			return err
		}
		rowNumber++
	}
	return nil
}

func idString(value *int64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(*value, 10)
}

func yesNo(value bool) string {
	if value {
		return "Ya"
	}
	return "Tidak"
}

func activeLabel(value bool) string {
	if value {
		return "Aktif"
	}
	return "Nonaktif"
}
