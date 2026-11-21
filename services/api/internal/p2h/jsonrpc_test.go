package p2h

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestJSONRPCClientLoadDetailsOmitsPhotos(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if request.Params.Service == "common" {
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":7}`)
			return
		}

		if request.Params.Service != "object" || request.Params.Method != "execute_kw" {
			t.Errorf("unexpected RPC: %+v", request.Params)
			http.Error(w, "unexpected RPC", http.StatusBadRequest)
			return
		}
		if len(request.Params.Args) < 7 {
			t.Errorf("short execute_kw args: %#v", request.Params.Args)
			http.Error(w, "short args", http.StatusBadRequest)
			return
		}
		model, _ := request.Params.Args[3].(string)
		kwargs, _ := request.Params.Args[6].(map[string]any)
		if hasRPCField(kwargs, "foto_kendaraan") {
			t.Error("foto_kendaraan must not be requested")
		}

		switch model {
		case "p2h.report":
			if !hasRPCField(kwargs, "catatan") {
				t.Error("catatan must be requested")
			}
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[{
				"id":5001,
				"name":"P2H-5001",
				"tanggal":"2026-11-20",
				"unit_id":[1438,"LV.02.011"],
				"unit_type":"light_vehicle",
				"shift_id":[2,"Shift 2"],
				"foreman_id":[18,"Rendi"],
				"employee_id":[998,"Rendi Rahman"],
				"state":"submitted",
				"approval_status":"pending",
				"km_akhir":172166,
				"hm_awal":false,
				"lokasi":false,
				"catatan":false
			}]}`)
		case "p2h.report.line":
			if !hasRPCField(kwargs, "state") || !hasRPCField(kwargs, "report_id") {
				t.Error("line fields must include state and report_id")
			}
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[
				{"id":279532,"report_id":[5001,"P2H-5001"],"question_id":[1,"TANGGA"],"name":"TANGGA / PEGANGAN","category":"KELILING","sequence":1,"state":"baik","remark":false},
				{"id":279533,"report_id":[5001,"P2H-5001"],"question_id":[2,"LAMPU"],"name":"LAMPU","category":"KELILING","sequence":2,"state":"rusak","remark":"mati"},
				{"id":279534,"report_id":[5001,"P2H-5001"],"question_id":[3,"TOOLKIT"],"name":"TOOLKIT","category":"PERLENGKAPAN","sequence":3,"state":"tidak_ada","remark":false}
			]}`)
		default:
			t.Errorf("unexpected model %q", model)
			http.Error(w, "unexpected model", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	client := NewJSONRPCClient(server.URL, "odoo_new", "svc_gsi_workorder", "secret", 5*time.Second)
	reports, err := client.LoadDetails(context.Background(), []int64{5001})
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}

	report := reports[0]
	if report.OdooReportID != 5001 || report.UnitCode != "LV.02.011" {
		t.Fatalf("report identity = %+v", report)
	}
	if report.Shift != "Shift 2" || report.ForemanName != "Rendi" || report.EmployeeName != "Rendi Rahman" {
		t.Fatalf("report names = %+v", report)
	}
	if report.Notes != "" || report.KMAkhir == nil || *report.KMAkhir != 172166 || report.HMAwal != nil {
		t.Fatalf("report optional fields = %+v", report)
	}
	if report.CheckedItems != 3 || len(report.Findings) != 2 {
		t.Fatalf("checked/findings = %d/%d, want 3/2", report.CheckedItems, len(report.Findings))
	}
	if report.Findings[0].Condition != "rusak" || report.Findings[1].Condition != "tidak_ada" {
		t.Fatalf("findings = %+v", report.Findings)
	}
	if strings.Contains(string(report.Raw), "foto_kendaraan") {
		t.Fatal("raw detail must not contain photo fields")
	}
}

func TestBuildReportsFromRPCEmptyLines(t *testing.T) {
	row := map[string]json.RawMessage{
		"id":      json.RawMessage(`7001`),
		"name":    json.RawMessage(`"P2H-7001"`),
		"tanggal": json.RawMessage(`"2026-11-20"`),
		"unit_id": json.RawMessage(`[1,"UNIT.01"]`),
		"state":   json.RawMessage(`"submitted"`),
		"catatan": json.RawMessage(`false`),
	}

	reports, err := buildReportsFromRPC(
		[]int64{7001},
		[]map[string]json.RawMessage{row},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	if reports[0].CheckedItems != 0 || len(reports[0].Findings) != 0 {
		t.Fatalf("empty detail = %+v", reports[0])
	}
	if reports[0].UnitCode != "UNIT.01" || reports[0].Raw == nil {
		t.Fatalf("empty detail metadata = %+v", reports[0])
	}
}

func TestJSONRPCClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	client := NewJSONRPCClient(server.URL, "odoo_new", "svc", "secret", time.Second)
	_, err := client.LoadDetails(context.Background(), []int64{5001})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func hasRPCField(kwargs map[string]any, name string) bool {
	fields, _ := kwargs["fields"].([]any)
	for _, field := range fields {
		if value, _ := field.(string); value == name {
			return true
		}
	}
	return false
}
