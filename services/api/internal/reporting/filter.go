package reporting

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ParseFilter converts HTTP query parameters into a reporting filter. Calendar
// dates are interpreted in the configured application timezone and converted
// to UTC instants; the upper bound is the exclusive start of the next day.
func ParseFilter(values url.Values, loc *time.Location) (Filter, error) {
	filter := Filter{
		Bucket:  strings.ToLower(strings.TrimSpace(values.Get("bucket"))),
		GroupBy: strings.ToLower(strings.TrimSpace(values.Get("group_by"))),
		Status:  strings.ToLower(strings.TrimSpace(values.Get("status"))),
		Format:  strings.ToLower(strings.TrimSpace(values.Get("format"))),
		Page:    positiveInt(values.Get("page"), 1),
		Limit:   positiveInt(values.Get("limit"), DefaultLimit),
	}
	fields := map[string]string{}

	from, err := parseDate(values.Get("from"), loc)
	if err != nil {
		fields["from"] = "Tanggal mulai tidak valid"
	} else {
		filter.From = from
	}
	to, err := parseDate(values.Get("to"), loc)
	if err != nil {
		fields["to"] = "Tanggal akhir tidak valid"
	} else if to != nil {
		next := to.AddDate(0, 0, 1)
		filter.To = &next
	}
	if filter.From != nil && filter.To != nil && !filter.From.Before(*filter.To) {
		fields["period"] = "Periode tidak valid"
	}

	filter.UnitID, err = parsePositiveID(values.Get("unit_id"))
	if err != nil {
		fields["unit_id"] = "Filter unit tidak valid"
	}
	filter.UnitModelID, err = parsePositiveID(values.Get("unit_model_id"))
	if err != nil {
		fields["unit_model_id"] = "Filter model unit tidak valid"
	}
	filter.LocationID, err = parsePositiveID(values.Get("location_id"))
	if err != nil {
		fields["location_id"] = "Filter lokasi tidak valid"
	}

	if filter.Bucket == "" {
		filter.Bucket = BucketDay
	}
	if filter.Bucket != BucketDay && filter.Bucket != BucketWeek && filter.Bucket != BucketMonth {
		fields["bucket"] = "Bucket harus day, week, atau month"
	}
	if filter.GroupBy == "" {
		filter.GroupBy = "plan"
	}
	if filter.GroupBy != "plan" && filter.GroupBy != "unit" && filter.GroupBy != "model" {
		fields["group_by"] = "Group by harus plan, unit, atau model"
	}
	if len(filter.Status) > 40 {
		fields["status"] = "Filter status terlalu panjang"
	}
	if filter.Format == "" {
		filter.Format = FormatJSON
	}
	if filter.Format != FormatJSON && filter.Format != FormatCSV {
		fields["format"] = "Format harus json atau csv"
	}
	if filter.Limit > MaxPageLimit {
		filter.Limit = MaxPageLimit
	}

	if len(fields) > 0 {
		return Filter{}, &ValidationError{Fields: fields}
	}
	return filter, nil
}

func parseDate(raw string, loc *time.Location) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.ParseInLocation("2006-01-02", raw, loc)
	if err != nil {
		return nil, err
	}
	utc := value.UTC()
	return &utc, nil
}

func parsePositiveID(raw string) (*int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return nil, ErrInvalidFilter
	}
	return &value, nil
}

func positiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

// bucketSQL builds the WITA bucket label expression for a timestamp column.
// The timezone name is always supplied as a bound parameter so no fixed offset
// is embedded in code or SQL.
func bucketSQL(bucket, column string, locArg int) (string, error) {
	zone := "$" + strconv.Itoa(locArg) + "::text"
	switch bucket {
	case BucketDay:
		return "to_char((" + column + " AT TIME ZONE " + zone + ")::date, 'YYYY-MM-DD')", nil
	case BucketWeek:
		return "to_char(date_trunc('week', (" + column + " AT TIME ZONE " + zone + ")), 'YYYY-MM-DD')", nil
	case BucketMonth:
		return "to_char(date_trunc('month', (" + column + " AT TIME ZONE " + zone + ")), 'YYYY-MM')", nil
	default:
		return "", ErrInvalidFilter
	}
}
