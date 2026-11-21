package main

import "testing"

func TestNormalizeP2HSource(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", input: "", want: "jsonrpc"},
		{name: "jsonrpc", input: "jsonrpc", want: "jsonrpc"},
		{name: "rest", input: "rest", want: "rest"},
		{name: "trim and case", input: " REST ", want: "rest"},
		{name: "invalid", input: "photos", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeP2HSource(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("source = %q, want %q", got, test.want)
			}
		})
	}
}
