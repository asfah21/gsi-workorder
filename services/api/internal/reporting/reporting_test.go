package reporting

import (
	"bytes"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func loadLocation(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestParseFilterWITABoundaries(t *testing.T) {
	loc := loadLocation(t)
	values := url.Values{
		"from":    {"2026-11-01"},
		"to":      {"2026-11-30"},
		"bucket":  {"week"},
		"format":  {"csv"},
		"limit":   {"500"},
		"unit_id": {"7"},
	}
	filter, err := ParseFilter(values, loc)
	if err != nil {
		t.Fatal(err)
	}
	wantFrom := time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 11, 30, 16, 0, 0, 0, time.UTC)
	if filter.From == nil || !filter.From.Equal(wantFrom) {
		t.Fatalf("from = %v, want %v", filter.From, wantFrom)
	}
	if filter.To == nil || !filter.To.Equal(wantTo) {
		t.Fatalf("to = %v, want %v", filter.To, wantTo)
	}
	if filter.Bucket != BucketWeek || filter.Format != FormatCSV {
		t.Fatalf("bucket/format = %s/%s", filter.Bucket, filter.Format)
	}
	if filter.Limit != MaxPageLimit {
		t.Fatalf("limit = %d, want %d", filter.Limit, MaxPageLimit)
	}
	if filter.UnitID == nil || *filter.UnitID != 7 {
		t.Fatalf("unit_id = %v", filter.UnitID)
	}
}

func TestParseFilterDefaultsAndValidation(t *testing.T) {
	loc := loadLocation(t)
	filter, err := ParseFilter(url.Values{}, loc)
	if err != nil {
		t.Fatal(err)
	}
	if filter.Bucket != BucketDay || filter.GroupBy != "plan" || filter.Format != FormatJSON {
		t.Fatalf("defaults = %#v", filter)
	}
	if filter.Page != 1 || filter.Limit != DefaultLimit {
		t.Fatalf("pagination = %d/%d", filter.Page, filter.Limit)
	}

	_, err = ParseFilter(url.Values{
		"from":   {"2026-99-99"},
		"bucket": {"year"},
		"format": {"pdf"},
	}, loc)
	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want ValidationError", err)
	}
	for _, key := range []string{"from", "bucket", "format"} {
		if validation.Fields[key] == "" {
			t.Fatalf("missing field error for %s: %#v", key, validation.Fields)
		}
	}

	_, err = ParseFilter(url.Values{
		"from": {"2026-11-30"},
		"to":   {"2026-11-01"},
	}, loc)
	if err == nil {
		t.Fatal("expected invalid period error")
	}
}

func TestBucketSQLUsesBoundTimezone(t *testing.T) {
	for _, bucket := range []string{BucketDay, BucketWeek, BucketMonth} {
		expression, err := bucketSQL(bucket, "posted_at", 1)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(expression, "AT TIME ZONE $1") {
			t.Fatalf("bucket %s expression = %s", bucket, expression)
		}
		if strings.Contains(expression, "+08") || strings.Contains(expression, "Asia/Makassar") {
			t.Fatalf("bucket %s embeds a fixed timezone: %s", bucket, expression)
		}
	}
	if _, err := bucketSQL("year", "posted_at", 1); err == nil {
		t.Fatal("expected error for unsupported bucket")
	}
}

func TestWriteCSV(t *testing.T) {
	loc := loadLocation(t)
	table := Table{
		Columns: []Column{
			{Key: "name", Label: "Nama"},
			{Key: "at", Label: "Waktu"},
			{Key: "note", Label: "Catatan"},
		},
		Rows: []map[string]any{
			{
				"name": "Bearing",
				"at":   time.Date(2026, 11, 9, 23, 0, 0, 0, time.UTC),
				"note": "=SUM(1)",
			},
			{
				"name": "Filter",
				"at":   time.Date(2026, 11, 10, 1, 0, 0, 0, time.UTC),
				"note": "-SUM(1)",
			},
		},
	}
	service := NewService(nil, loc)
	var buffer bytes.Buffer
	if err := service.RenderCSV(&buffer, table); err != nil {
		t.Fatal(err)
	}
	output := buffer.String()
	if !strings.HasPrefix(output, "\ufeff") {
		t.Fatal("CSV missing UTF-8 BOM")
	}
	for _, want := range []string{
		"Nama,Waktu,Catatan",
		"Bearing,2026-11-10 07:00:00,'=SUM(1)",
		"Filter,2026-11-10 09:00:00,'-SUM(1)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("CSV output %q missing %q", output, want)
		}
	}
}

func TestReportNames(t *testing.T) {
	names := ReportNames()
	if len(names) != 5 {
		t.Fatalf("report names = %v", names)
	}
	want := map[string]bool{
		"pm-compliance": false, "shortage-aging": false,
		"sparepart-consumption": false, "inventory-movement": false,
		"component-replacement": false,
	}
	for _, name := range names {
		if _, ok := want[name]; !ok {
			t.Fatalf("unexpected report %s", name)
		}
		want[name] = true
	}
	for name, found := range want {
		if !found {
			t.Fatalf("missing report %s", name)
		}
	}
}
