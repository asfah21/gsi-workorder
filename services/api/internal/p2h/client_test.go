package p2h

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientListAndDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/p2h":
			if r.URL.Query().Get("limit") != "100" || r.URL.Query().Get("offset") != "0" {
				t.Errorf("unexpected query: %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"code":200,"status":"ok","data":{"data":[{
				"id":5001,"name":"P2H-5001","tanggal":"2026-11-20",
				"unit":{"id":1438,"name":"LV.02.011"},"unit_type":"light_vehicle",
				"state":"submitted","approval_status":"pending"
			}],"total":1}}`)
		case "/api/p2h/5001":
			fmt.Fprint(w, `{"code":200,"status":"ok","data":{
				"id":5001,"name":"P2H-5001","tanggal":"2026-11-20",
				"unit":{"id":1438,"name":"LV.02.011"},"unit_type":"light_vehicle",
				"shift":{"id":2,"name":"Shift 2"},"foreman":{"id":18,"name":"Rendi"},
				"employee":{"id":998,"name":"Rendi Rahman"},"lokasi":"office 1",
				"state":"submitted","approval_status":"pending","km_akhir":172166,
				"checklist_lines":[
					{"id":279532,"question_id":1,"name":"TANGGA / PEGANGAN","category":"KELILING","state":"baik"},
					{"id":279533,"question_id":2,"name":"LAMPU","category":"KELILING","state":"rusak","remark":"mati"}
				]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, "test-token", "", "")
	page, err := client.List(context.Background(), 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Reports) != 1 {
		t.Fatalf("list page = %+v", page)
	}
	if page.Reports[0].UnitCode != "LV.02.011" || page.Reports[0].ReportDate == nil {
		t.Fatalf("report = %+v", page.Reports[0])
	}

	report, err := client.Get(context.Background(), 5001)
	if err != nil {
		t.Fatal(err)
	}
	if report.CheckedItems != 2 {
		t.Fatalf("checked items = %d, want 2", report.CheckedItems)
	}
	if len(report.Findings) != 1 || report.Findings[0].Condition != "rusak" {
		t.Fatalf("findings = %+v", report.Findings)
	}
	if report.Raw == nil {
		t.Fatal("detail raw JSON must be retained")
	}
}

func TestClientUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"code":401,"status":"unauthorized","data":null,"message":"token expired"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, 5*time.Second, "bad-token", "", "")
	_, err := client.List(context.Background(), 100, 0)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}
