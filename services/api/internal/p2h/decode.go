package p2h

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type envelope struct {
	Code    int             `json:"code"`
	Status  string          `json:"status"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
	Errors  json.RawMessage `json:"errors"`
}

type reportDTO struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Tanggal        string    `json:"tanggal"`
	Unit           nameRef   `json:"unit"`
	UnitType       string    `json:"unit_type"`
	Shift          nameRef   `json:"shift"`
	Foreman        nameRef   `json:"foreman"`
	Employee       nameRef   `json:"employee"`
	Lokasi         string    `json:"lokasi"`
	State          string    `json:"state"`
	ApprovalStatus string    `json:"approval_status"`
	KMAkhir        *float64  `json:"km_akhir"`
	HMAwal         *float64  `json:"hm_awal"`
	Notes          string    `json:"notes"`
	ChecklistLines []lineDTO `json:"checklist_lines"`
}

type lineDTO struct {
	ID         int64  `json:"id"`
	QuestionID *int64 `json:"question_id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Sequence   int    `json:"sequence"`
	State      string `json:"state"`
	Remark     string `json:"remark"`
}

type nameRef struct {
	ID   *int64
	Name string
}

func (n *nameRef) UnmarshalJSON(raw []byte) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "false" {
		return nil
	}
	switch trimmed[0] {
	case '{':
		var object struct {
			ID   *int64 `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &object); err != nil {
			return err
		}
		n.ID = object.ID
		n.Name = strings.TrimSpace(object.Name)
		return nil
	case '[':
		var pair []json.RawMessage
		if err := json.Unmarshal(raw, &pair); err != nil {
			return err
		}
		if len(pair) > 0 {
			n.ID = decodeOptionalInt(pair[0])
		}
		if len(pair) > 1 {
			_ = json.Unmarshal(pair[1], &n.Name)
			n.Name = strings.TrimSpace(n.Name)
		}
		return nil
	case '"':
		return json.Unmarshal(raw, &n.Name)
	default:
		n.ID = decodeOptionalInt(raw)
		return nil
	}
}

func decodeOptionalInt(raw json.RawMessage) *int64 {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" || value == "false" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func decodeList(data json.RawMessage) ([]reportDTO, int, error) {
	var list []reportDTO
	if err := json.Unmarshal(data, &list); err == nil {
		return list, -1, nil
	}

	var object struct {
		Data       []reportDTO `json:"data"`
		Items      []reportDTO `json:"items"`
		Reports    []reportDTO `json:"reports"`
		Rows       []reportDTO `json:"rows"`
		Total      *int        `json:"total"`
		Count      *int        `json:"count"`
		TotalCount *int        `json:"total_count"`
		Meta       struct {
			Total *int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, 0, fmt.Errorf("%w: decode report list", ErrUnexpectedResponse)
	}
	items := object.Data
	if items == nil {
		items = object.Items
	}
	if items == nil {
		items = object.Reports
	}
	if items == nil {
		items = object.Rows
	}
	if items == nil {
		return nil, 0, fmt.Errorf("%w: report list is missing", ErrUnexpectedResponse)
	}
	total := -1
	for _, candidate := range []*int{object.Total, object.Count, object.TotalCount, object.Meta.Total} {
		if candidate != nil {
			total = *candidate
			break
		}
	}
	return items, total, nil
}

func decodeDetail(data json.RawMessage) (reportDTO, json.RawMessage, error) {
	var direct reportDTO
	if err := json.Unmarshal(data, &direct); err == nil && direct.ID > 0 {
		return direct, data, nil
	}

	var object struct {
		Data   reportDTO `json:"data"`
		Report reportDTO `json:"report"`
	}
	if err := json.Unmarshal(data, &object); err == nil {
		if object.Data.ID > 0 {
			raw, _ := json.Marshal(object.Data)
			return object.Data, raw, nil
		}
		if object.Report.ID > 0 {
			raw, _ := json.Marshal(object.Report)
			return object.Report, raw, nil
		}
	}
	return reportDTO{}, nil, fmt.Errorf("%w: decode report detail", ErrUnexpectedResponse)
}

func parseReport(dto reportDTO) (Report, error) {
	if dto.ID <= 0 {
		return Report{}, fmt.Errorf("%w: report id is missing", ErrUnexpectedResponse)
	}
	reportDate, err := parseDate(dto.Tanggal)
	if err != nil {
		return Report{}, fmt.Errorf("report %d: %w", dto.ID, err)
	}
	return Report{
		OdooReportID:   dto.ID,
		Name:           strings.TrimSpace(dto.Name),
		ReportDate:     reportDate,
		OdooUnitID:     dto.Unit.ID,
		UnitCode:       strings.TrimSpace(dto.Unit.Name),
		UnitType:       strings.TrimSpace(dto.UnitType),
		Shift:          strings.TrimSpace(dto.Shift.Name),
		ForemanName:    strings.TrimSpace(dto.Foreman.Name),
		EmployeeName:   strings.TrimSpace(dto.Employee.Name),
		Location:       strings.TrimSpace(dto.Lokasi),
		State:          strings.TrimSpace(dto.State),
		ApprovalStatus: strings.TrimSpace(dto.ApprovalStatus),
		KMAkhir:        dto.KMAkhir,
		HMAwal:         dto.HMAwal,
		Notes:          strings.TrimSpace(dto.Notes),
		CheckedItems:   len(dto.ChecklistLines),
	}, nil
}

func parseFindings(lines []lineDTO) []Finding {
	findings := make([]Finding, 0)
	for _, line := range lines {
		condition := strings.ToLower(strings.TrimSpace(line.State))
		if condition != "rusak" && condition != "tidak_ada" {
			continue
		}
		if line.ID <= 0 {
			continue
		}
		category := strings.TrimSpace(line.Category)
		findings = append(findings, Finding{
			OdooLineID:   line.ID,
			QuestionID:   line.QuestionID,
			Component:    strings.TrimSpace(line.Name),
			Category:     category,
			CategoryNorm: strings.ToLower(category),
			Condition:    condition,
			Remark:       strings.TrimSpace(line.Remark),
		})
	}
	return findings
}

func parseDate(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			date := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			return &date, nil
		}
	}
	return nil, fmt.Errorf("invalid report date %q", raw)
}
