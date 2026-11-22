package reporting

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// WriteCSV renders a report table as UTF-8 CSV. A BOM is included so spreadsheet
// applications open Indonesian labels correctly, and timestamps are rendered in
// the configured application timezone.
func WriteCSV(w io.Writer, table Table, loc *time.Location) error {
	if _, err := io.WriteString(w, "\ufeff"); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	writer.UseCRLF = true

	header := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		header = append(header, column.Label)
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, row := range table.Rows {
		record := make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			record = append(record, csvValue(row[column.Key], loc))
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

// csvValue formats one cell and neutralises spreadsheet formula injection from
// user-supplied text (notes, titles, codes).
func csvValue(value any, loc *time.Location) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return escapeFormula(typed)
	case time.Time:
		if loc != nil {
			typed = typed.In(loc)
		}
		return typed.Format("2006-01-02 15:04:05")
	case *time.Time:
		if typed == nil {
			return ""
		}
		if loc != nil {
			return typed.In(loc).Format("2006-01-02 15:04:05")
		}
		return typed.Format("2006-01-02 15:04:05")
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case []byte:
		return escapeFormula(string(typed))
	default:
		return escapeFormula(fmt.Sprint(typed))
	}
}

func escapeFormula(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" && strings.ContainsRune("=+-@\t\r", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
