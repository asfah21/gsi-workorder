package workorders

import "testing"

func TestValidateTransitionBaseline(t *testing.T) {
	tests := []struct {
		name    string
		item    WorkOrder
		actor   Actor
		input   PatchInput
		note    string
		wantErr bool
	}{
		{
			name:  "open to in progress",
			item:  WorkOrder{Status: "open"},
			actor: Actor{ID: 1, Role: "mechanic"},
			input: PatchInput{Status: "in_progress"},
		},
		{
			name:    "open to done is invalid",
			item:    WorkOrder{Status: "open"},
			actor:   Actor{ID: 1, Role: "planner"},
			input:   PatchInput{Status: "done"},
			note:    "selesai",
			wantErr: true,
		},
		{
			name:    "in progress to waiting part requires note",
			item:    WorkOrder{Status: "in_progress"},
			actor:   Actor{ID: 1, Role: "mechanic"},
			input:   PatchInput{Status: "waiting_part"},
			wantErr: true,
		},
		{
			name:  "in progress to waiting part with note",
			item:  WorkOrder{Status: "in_progress"},
			actor: Actor{ID: 1, Role: "mechanic"},
			input: PatchInput{Status: "waiting_part"},
			note:  "menunggu part",
		},
		{
			name:    "waiting part to done is invalid",
			item:    WorkOrder{Status: "waiting_part"},
			actor:   Actor{ID: 1, Role: "planner"},
			input:   PatchInput{Status: "done"},
			note:    "selesai",
			wantErr: true,
		},
		{
			name:    "in progress to done requires note",
			item:    WorkOrder{Status: "in_progress"},
			actor:   Actor{ID: 1, Role: "mechanic"},
			input:   PatchInput{Status: "done"},
			wantErr: true,
		},
		{
			name:  "in progress to done with note",
			item:  WorkOrder{Status: "in_progress"},
			actor: Actor{ID: 1, Role: "mechanic"},
			input: PatchInput{Status: "done"},
			note:  "pekerjaan selesai",
		},
		{
			name:    "mechanic cannot cancel",
			item:    WorkOrder{Status: "in_progress"},
			actor:   Actor{ID: 1, Role: "mechanic"},
			input:   PatchInput{Status: "cancelled"},
			note:    "batal",
			wantErr: true,
		},
		{
			name:  "planner can cancel with reason",
			item:  WorkOrder{Status: "open"},
			actor: Actor{ID: 1, Role: "planner"},
			input: PatchInput{Status: "cancelled"},
			note:  "unit tidak dipakai",
		},
		{
			name:    "terminal cannot transition",
			item:    WorkOrder{Status: "done"},
			actor:   Actor{ID: 1, Role: "admin"},
			input:   PatchInput{Status: "in_progress"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateTransition(test.item, test.actor, test.input, test.note)
			if test.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWorkOrderVisibility(t *testing.T) {
	assigned := WorkOrder{AssigneeID: ptrInt64(7)}
	unassigned := WorkOrder{}

	if !canView(Actor{ID: 7, Role: "mechanic"}, assigned) {
		t.Fatal("assigned mechanic should view own WO")
	}
	if canView(Actor{ID: 8, Role: "mechanic"}, assigned) {
		t.Fatal("other mechanic must not view assigned WO")
	}
	if !canView(Actor{ID: 8, Role: "mechanic"}, unassigned) {
		t.Fatal("mechanic should see unassigned WO for claiming")
	}
	if !canModify(Actor{ID: 7, Role: "mechanic"}, assigned) {
		t.Fatal("assigned mechanic should modify own WO")
	}
	if canModify(Actor{ID: 8, Role: "mechanic"}, assigned) {
		t.Fatal("other mechanic must not modify assigned WO")
	}
	if canModify(Actor{ID: 1, Role: "viewer"}, assigned) {
		t.Fatal("viewer must not modify WO")
	}
}

func TestWorkOrderVocabulary(t *testing.T) {
	for _, value := range []string{"preventive", "corrective", "breakdown", "inspection"} {
		if !validMaintenanceType(value) {
			t.Fatalf("maintenance type %q should be valid", value)
		}
	}
	if validMaintenanceType("unknown") {
		t.Fatal("unknown maintenance type should be invalid")
	}
	for _, value := range []string{"low", "medium", "high", "critical"} {
		if !validPriority(value) {
			t.Fatalf("priority %q should be valid", value)
		}
	}
	if validPriority("urgent") {
		t.Fatal("unknown priority should be invalid")
	}
}

func ptrInt64(value int64) *int64 {
	return &value
}
