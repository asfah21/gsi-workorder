package p2h

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	jsonRPCDetailBatchSize  = 100
	jsonRPCLinePageSize     = 5000
	jsonRPCMaxResponseBytes = 32 << 20
)

var jsonRPCBackoff = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

var p2hReportFields = []string{
	"id",
	"name",
	"tanggal",
	"unit_id",
	"unit_type",
	"shift_id",
	"foreman_id",
	"employee_id",
	"state",
	"approval_status",
	"km_akhir",
	"hm_awal",
	"lokasi",
	"catatan",
}

var p2hLineFields = []string{
	"id",
	"report_id",
	"question_id",
	"name",
	"category",
	"sequence",
	"state",
	"remark",
}

// JSONRPCClient loads P2H detail through Odoo's JSON-RPC API. It deliberately
// requests only the fields stored locally and never requests binary photo
// fields such as foto_kendaraan.
type JSONRPCClient struct {
	baseURL  string
	db       string
	login    string
	password string
	http     *http.Client

	mu  sync.Mutex
	uid int64
}

func NewJSONRPCClient(baseURL, db, login, password string, timeout time.Duration) *JSONRPCClient {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxConnsPerHost:       2,
		MaxIdleConns:          2,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
	return &JSONRPCClient{
		baseURL:  strings.TrimRight(baseURL, "/"),
		db:       strings.TrimSpace(db),
		login:    strings.TrimSpace(login),
		password: password,
		http:     &http.Client{Timeout: timeout, Transport: transport},
	}
}

func (c *JSONRPCClient) Configured() bool {
	return c.baseURL != "" && c.db != "" && c.login != "" && c.password != ""
}

// LoadDetails fetches report headers and checklist lines for the supplied
// Odoo report IDs. Reports with no checklist lines are returned too, so the
// caller can mark their detail as synced.
func (c *JSONRPCClient) LoadDetails(ctx context.Context, ids []int64) ([]Report, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	if len(ids) == 0 {
		return []Report{}, nil
	}

	unique := uniqueSortedIDs(ids)
	uid, err := c.serviceUID(ctx)
	if err != nil {
		return nil, err
	}

	reports := make([]Report, 0, len(unique))
	for start := 0; start < len(unique); start += jsonRPCDetailBatchSize {
		end := start + jsonRPCDetailBatchSize
		if end > len(unique) {
			end = len(unique)
		}
		batch := unique[start:end]

		reportRows, err := c.searchRead(ctx, uid, "p2h.report", []any{[]any{"id", "in", batch}}, p2hReportFields, len(batch), 0, "id asc")
		if err != nil {
			return nil, fmt.Errorf("read P2H report headers: %w", err)
		}
		lineRows, err := c.searchLines(ctx, uid, batch)
		if err != nil {
			return nil, fmt.Errorf("read P2H checklist lines: %w", err)
		}

		batchReports, err := buildReportsFromRPC(batch, reportRows, lineRows)
		if err != nil {
			return nil, err
		}
		reports = append(reports, batchReports...)
	}
	return reports, nil
}

func (c *JSONRPCClient) searchLines(ctx context.Context, uid int64, ids []int64) ([]map[string]json.RawMessage, error) {
	var rows []map[string]json.RawMessage
	for offset := 0; ; {
		page, err := c.searchRead(
			ctx,
			uid,
			"p2h.report.line",
			[]any{[]any{"report_id", "in", ids}},
			p2hLineFields,
			jsonRPCLinePageSize,
			offset,
			"id asc",
		)
		if err != nil {
			return nil, err
		}
		rows = append(rows, page...)
		if len(page) < jsonRPCLinePageSize {
			return rows, nil
		}
		offset += len(page)
	}
}

func (c *JSONRPCClient) searchRead(
	ctx context.Context,
	uid int64,
	model string,
	domain []any,
	fields []string,
	limit int,
	offset int,
	order string,
) ([]map[string]json.RawMessage, error) {
	kwargs := map[string]any{
		"fields":  fields,
		"context": map[string]any{"active_test": false},
	}
	if limit > 0 {
		kwargs["limit"] = limit
	}
	if offset > 0 {
		kwargs["offset"] = offset
	}
	if order != "" {
		kwargs["order"] = order
	}

	raw, err := c.call(ctx, "object", "execute_kw", []any{
		c.db,
		uid,
		c.password,
		model,
		"search_read",
		[]any{domain},
		kwargs,
	})
	if err != nil {
		return nil, err
	}

	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("%w: decode search_read result", ErrUnexpectedResponse)
	}
	return rows, nil
}

func (c *JSONRPCClient) serviceUID(ctx context.Context) (int64, error) {
	c.mu.Lock()
	if c.uid > 0 {
		uid := c.uid
		c.mu.Unlock()
		return uid, nil
	}
	c.mu.Unlock()

	raw, err := c.call(ctx, "common", "login", []any{c.db, c.login, c.password})
	if err != nil {
		return 0, err
	}

	var uid int64
	if err := json.Unmarshal(raw, &uid); err == nil && uid > 0 {
		c.mu.Lock()
		c.uid = uid
		c.mu.Unlock()
		return uid, nil
	}

	var valid bool
	if err := json.Unmarshal(raw, &valid); err == nil && !valid {
		return 0, ErrUnauthorized
	}
	return 0, fmt.Errorf("%w: login result is missing a uid", ErrUnexpectedResponse)
}

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  jsonRPCParams `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCParams struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Args    []any  `json:"args"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonRPCError   `json:"error"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (c *JSONRPCClient) call(ctx context.Context, service, method string, args []any) (json.RawMessage, error) {
	endpoint, err := url.JoinPath(c.baseURL, "jsonrpc")
	if err != nil {
		return nil, fmt.Errorf("%w: build JSON-RPC URL: %v", ErrUnavailable, err)
	}

	payload, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "call",
		Params:  jsonRPCParams{Service: service, Method: method, Args: args},
		ID:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: encode JSON-RPC request", ErrUnavailable)
	}

	var lastErr error
	for attempt := 0; attempt <= len(jsonRPCBackoff); attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, jsonRPCBackoff[attempt-1]); err != nil {
				return nil, err
			}
		}

		result, retry, err := c.callOnce(ctx, endpoint, payload)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retry || ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *JSONRPCClient) callOnce(ctx context.Context, endpoint string, payload []byte) (json.RawMessage, bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("%w: build JSON-RPC request: %v", ErrUnavailable, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return nil, true, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, jsonRPCMaxResponseBytes))
	if err != nil {
		return nil, true, fmt.Errorf("%w: read JSON-RPC response", ErrUnavailable)
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, false, ErrUnauthorized
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return nil, true, fmt.Errorf("%w: JSON-RPC status %d", ErrUnavailable, response.StatusCode)
	default:
		return nil, false, fmt.Errorf("%w: JSON-RPC status %d", ErrUnavailable, response.StatusCode)
	}

	var envelope jsonRPCResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, false, fmt.Errorf("%w: decode JSON-RPC envelope", ErrUnexpectedResponse)
	}
	if envelope.Error != nil {
		message := strings.TrimSpace(envelope.Error.Message)
		if message == "" {
			message = "unknown Odoo JSON-RPC error"
		}
		if envelope.Error.Code == http.StatusUnauthorized || envelope.Error.Code == http.StatusForbidden {
			return nil, false, ErrUnauthorized
		}
		if envelope.Error.Code >= 500 {
			return nil, true, fmt.Errorf("%w: Odoo JSON-RPC %d: %s", ErrUnavailable, envelope.Error.Code, message)
		}
		return nil, false, fmt.Errorf("%w: Odoo JSON-RPC %d: %s", ErrUnavailable, envelope.Error.Code, message)
	}
	if len(envelope.Result) == 0 {
		return nil, false, fmt.Errorf("%w: JSON-RPC result is empty", ErrUnexpectedResponse)
	}
	return envelope.Result, false, nil
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func uniqueSortedIDs(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

type rpcLine struct {
	reportID int64
	dto      lineDTO
	raw      map[string]json.RawMessage
}

func buildReportsFromRPC(ids []int64, reportRows, lineRows []map[string]json.RawMessage) ([]Report, error) {
	reportsByID := make(map[int64]map[string]json.RawMessage, len(reportRows))
	for _, row := range reportRows {
		id, err := decodeInt64Field(row, "id")
		if err != nil {
			return nil, fmt.Errorf("%w: report row: %v", ErrUnexpectedResponse, err)
		}
		reportsByID[id] = row
	}

	linesByReport := make(map[int64][]rpcLine)
	for _, row := range lineRows {
		reportID, line, err := parseRPCLine(row)
		if err != nil {
			return nil, err
		}
		linesByReport[reportID] = append(linesByReport[reportID], rpcLine{
			reportID: reportID,
			dto:      line,
			raw:      row,
		})
	}

	result := make([]Report, 0, len(ids))
	for _, id := range ids {
		row, ok := reportsByID[id]
		if !ok {
			return nil, fmt.Errorf("%w: report %d is missing from Odoo detail", ErrUnexpectedResponse, id)
		}
		dto, err := parseRPCReport(row)
		if err != nil {
			return nil, err
		}

		rawLines := make([]map[string]json.RawMessage, 0, len(linesByReport[id]))
		dto.ChecklistLines = make([]lineDTO, 0, len(linesByReport[id]))
		for _, line := range linesByReport[id] {
			dto.ChecklistLines = append(dto.ChecklistLines, line.dto)
			rawLines = append(rawLines, line.raw)
		}

		report, err := parseReport(dto)
		if err != nil {
			return nil, err
		}
		report.CheckedItems = len(dto.ChecklistLines)
		report.Findings = parseFindings(dto.ChecklistLines)
		report.Raw = compactRPCReport(row, rawLines)
		result = append(result, report)
	}
	return result, nil
}

func parseRPCReport(row map[string]json.RawMessage) (reportDTO, error) {
	id, err := decodeInt64Field(row, "id")
	if err != nil {
		return reportDTO{}, fmt.Errorf("%w: report row: %v", ErrUnexpectedResponse, err)
	}
	return reportDTO{
		ID:             id,
		Name:           decodeStringField(row, "name"),
		Tanggal:        decodeStringField(row, "tanggal"),
		Unit:           decodeNameRef(row["unit_id"]),
		UnitType:       decodeStringField(row, "unit_type"),
		Shift:          decodeNameRef(row["shift_id"]),
		Foreman:        decodeNameRef(row["foreman_id"]),
		Employee:       decodeNameRef(row["employee_id"]),
		Lokasi:         decodeStringField(row, "lokasi"),
		State:          decodeStringField(row, "state"),
		ApprovalStatus: decodeStringField(row, "approval_status"),
		KMAkhir:        decodeOptionalFloat(row["km_akhir"]),
		HMAwal:         decodeOptionalFloat(row["hm_awal"]),
		Notes:          decodeStringField(row, "catatan"),
	}, nil
}

func parseRPCLine(row map[string]json.RawMessage) (int64, lineDTO, error) {
	id, err := decodeInt64Field(row, "id")
	if err != nil {
		return 0, lineDTO{}, fmt.Errorf("%w: checklist line: %v", ErrUnexpectedResponse, err)
	}

	reportRef := decodeNameRef(row["report_id"])
	if reportRef.ID == nil || *reportRef.ID <= 0 {
		return 0, lineDTO{}, fmt.Errorf("%w: checklist line %d has no report_id", ErrUnexpectedResponse, id)
	}

	questionRef := decodeNameRef(row["question_id"])
	var questionID *int64
	if questionRef.ID != nil && *questionRef.ID > 0 {
		questionID = questionRef.ID
	}

	sequence := 0
	if value := decodeOptionalInt(row["sequence"]); value != nil {
		sequence = int(*value)
	}

	return *reportRef.ID, lineDTO{
		ID:         id,
		QuestionID: questionID,
		Name:       decodeStringField(row, "name"),
		Category:   decodeStringField(row, "category"),
		Sequence:   sequence,
		State:      decodeStringField(row, "state"),
		Remark:     decodeStringField(row, "remark"),
	}, nil
}

func decodeStringField(row map[string]json.RawMessage, key string) string {
	raw, ok := row[key]
	if !ok {
		return ""
	}
	return decodeStringValue(raw)
}

func decodeStringValue(raw json.RawMessage) string {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" || value == "false" {
		return ""
	}
	var result string
	if err := json.Unmarshal(raw, &result); err != nil {
		return ""
	}
	return strings.TrimSpace(result)
}

func decodeInt64Field(row map[string]json.RawMessage, key string) (int64, error) {
	raw, ok := row[key]
	if !ok {
		return 0, fmt.Errorf("%s is missing", key)
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%s is not an integer", key)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s is not positive", key)
	}
	return value, nil
}

func decodeOptionalFloat(raw json.RawMessage) *float64 {
	value := strings.TrimSpace(string(raw))
	if value == "" || value == "null" || value == "false" {
		return nil
	}
	var result float64
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return &result
}

func decodeNameRef(raw json.RawMessage) nameRef {
	var ref nameRef
	if len(raw) > 0 {
		_ = ref.UnmarshalJSON(raw)
	}
	return ref
}

func compactRPCReport(report map[string]json.RawMessage, lines []map[string]json.RawMessage) json.RawMessage {
	payload := struct {
		Report map[string]json.RawMessage   `json:"report"`
		Lines  []map[string]json.RawMessage `json:"lines,omitempty"`
	}{
		Report: report,
		Lines:  lines,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}
