package units

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInstallComponentDefaultsFromUnitAndComponent(t *testing.T) {
	repo := &fakeRepository{
		unitsByID: map[int64]Unit{
			1: {ID: 1, Code: "LV-01", CurrentHM: 120, CurrentKM: 500, IsActive: true},
		},
		component: Component{
			ID:                  2,
			Code:                "ENG-01",
			Name:                "Engine",
			IsActive:            true,
			DefaultLifetimeHM:   floatPointer(1000),
			DefaultLifetimeDays: intPointerValue(365),
			DefaultLifetimeKM:   floatPointer(5000),
		},
		createdInstallation: ComponentInstallation{ID: 9, UnitID: 1, ComponentID: 2},
	}
	service := NewService(repo, nil)

	item, err := service.InstallComponent(context.Background(), 1, ComponentInstallationInput{
		ComponentID:  2,
		SerialNumber: "  SN-ENG-01  ",
		Notes:        "  install awal  ",
	}, 7)
	if err != nil {
		t.Fatalf("InstallComponent() error = %v", err)
	}
	if item.ID != 9 {
		t.Fatalf("InstallComponent() ID = %d, want 9", item.ID)
	}
	input := repo.createdInstallationInput
	if input.UnitID != 1 || input.ComponentID != 2 {
		t.Fatalf("created unit/component = %d/%d", input.UnitID, input.ComponentID)
	}
	if input.SerialNumber != "SN-ENG-01" {
		t.Fatalf("serial = %q", input.SerialNumber)
	}
	if input.InstalledHM != 120 || input.InstalledKM != 500 {
		t.Fatalf("installed HM/KM = %v/%v, want 120/500", input.InstalledHM, input.InstalledKM)
	}
	if input.LifetimeHM == nil || *input.LifetimeHM != 1000 {
		t.Fatalf("lifetime HM = %v, want 1000", input.LifetimeHM)
	}
	if input.LifetimeDays == nil || *input.LifetimeDays != 365 {
		t.Fatalf("lifetime days = %v, want 365", input.LifetimeDays)
	}
	if input.LifetimeKM == nil || *input.LifetimeKM != 5000 {
		t.Fatalf("lifetime KM = %v, want 5000", input.LifetimeKM)
	}
	if input.CreatedBy == nil || *input.CreatedBy != 7 {
		t.Fatalf("created by = %v, want 7", input.CreatedBy)
	}
	if input.Notes != "install awal" {
		t.Fatalf("notes = %q", input.Notes)
	}
}

func TestInstallComponentRejectsInactiveUnit(t *testing.T) {
	repo := &fakeRepository{
		unitsByID: map[int64]Unit{
			1: {ID: 1, IsActive: false},
		},
		component: Component{ID: 2, IsActive: true},
	}
	service := NewService(repo, nil)

	_, err := service.InstallComponent(context.Background(), 1, ComponentInstallationInput{
		ComponentID:  2,
		SerialNumber: "SN-1",
	}, 1)
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("InstallComponent() error = %v, want ErrInactive", err)
	}
}

func TestReplaceComponentClosesOldAndUsesComponentDefaults(t *testing.T) {
	installedAt := time.Now().UTC().Add(-100 * time.Hour)
	repo := &fakeRepository{
		unitsByID: map[int64]Unit{
			1: {ID: 1, CurrentHM: 150, CurrentKM: 600, IsActive: true},
		},
		component: Component{
			ID:                2,
			IsActive:          true,
			DefaultLifetimeHM: floatPointer(2000),
		},
		installation: ComponentInstallation{
			ID:           5,
			UnitID:       1,
			ComponentID:  2,
			SerialNumber: "OLD-SN",
			InstalledAt:  installedAt,
			InstalledHM:  100,
			InstalledKM:  400,
			Status:       installationStatusActive,
			LifetimeHM:   floatPointer(1000),
		},
		createdInstallation: ComponentInstallation{ID: 6, UnitID: 1, ComponentID: 2},
	}
	service := NewService(repo, nil)

	item, err := service.ReplaceComponent(context.Background(), 1, 5, ComponentReplacementInput{
		SerialNumber: "NEW-SN",
	}, 7)
	if err != nil {
		t.Fatalf("ReplaceComponent() error = %v", err)
	}
	if item.ID != 6 {
		t.Fatalf("ReplaceComponent() ID = %d, want 6", item.ID)
	}
	if repo.closedInstallationID != 5 {
		t.Fatalf("closed installation = %d, want 5", repo.closedInstallationID)
	}
	if repo.closedInstallation.Status != installationStatusRemoved {
		t.Fatalf("close status = %q, want removed", repo.closedInstallation.Status)
	}
	if repo.closedInstallation.RemovedHM == nil || *repo.closedInstallation.RemovedHM != 150 {
		t.Fatalf("removed HM = %v, want 150", repo.closedInstallation.RemovedHM)
	}
	input := repo.createdInstallationInput
	if input.SerialNumber != "NEW-SN" {
		t.Fatalf("new serial = %q", input.SerialNumber)
	}
	if input.InstalledHM != 150 || input.InstalledKM != 600 {
		t.Fatalf("new installed HM/KM = %v/%v, want 150/600", input.InstalledHM, input.InstalledKM)
	}
	if input.LifetimeHM == nil || *input.LifetimeHM != 2000 {
		t.Fatalf("new lifetime HM = %v, want 2000", input.LifetimeHM)
	}
}

func TestReplaceComponentAllowsInactiveMaster(t *testing.T) {
	installedAt := time.Now().UTC().Add(-10 * time.Hour)
	repo := &fakeRepository{
		unitsByID: map[int64]Unit{
			1: {ID: 1, CurrentHM: 60, CurrentKM: 200, IsActive: true},
		},
		component: Component{ID: 2, IsActive: false},
		installation: ComponentInstallation{
			ID:           5,
			UnitID:       1,
			ComponentID:  2,
			SerialNumber: "OLD",
			InstalledAt:  installedAt,
			InstalledHM:  50,
			InstalledKM:  180,
			Status:       installationStatusActive,
		},
		createdInstallation: ComponentInstallation{ID: 6, UnitID: 1, ComponentID: 2},
	}
	service := NewService(repo, nil)

	if _, err := service.ReplaceComponent(context.Background(), 1, 5, ComponentReplacementInput{
		SerialNumber: "NEW",
	}, 7); err != nil {
		t.Fatalf("ReplaceComponent() with inactive master error = %v", err)
	}
}

func TestTransferComponentCarriesSerialAndLifetime(t *testing.T) {
	installedAt := time.Now().UTC().Add(-50 * time.Hour)
	repo := &fakeRepository{
		unitsByID: map[int64]Unit{
			1: {ID: 1, CurrentHM: 150, CurrentKM: 600, IsActive: true},
			2: {ID: 2, CurrentHM: 50, CurrentKM: 100, IsActive: true},
		},
		component: Component{ID: 2, IsActive: true, DefaultLifetimeHM: floatPointer(2000)},
		installation: ComponentInstallation{
			ID:           5,
			UnitID:       1,
			ComponentID:  2,
			SerialNumber: "MOVE-SN",
			InstalledAt:  installedAt,
			InstalledHM:  100,
			InstalledKM:  400,
			Status:       installationStatusActive,
			LifetimeHM:   floatPointer(1000),
			LifetimeDays: intPointerValue(90),
		},
		createdInstallation: ComponentInstallation{ID: 7, UnitID: 2, ComponentID: 2},
	}
	service := NewService(repo, nil)

	item, err := service.TransferComponent(context.Background(), 1, 5, ComponentTransferInput{
		ToUnitID: 2,
	}, 7)
	if err != nil {
		t.Fatalf("TransferComponent() error = %v", err)
	}
	if item.ID != 7 {
		t.Fatalf("TransferComponent() ID = %d, want 7", item.ID)
	}
	if repo.closedInstallation.Status != installationStatusTransferred {
		t.Fatalf("close status = %q, want transferred", repo.closedInstallation.Status)
	}
	input := repo.createdInstallationInput
	if input.UnitID != 2 {
		t.Fatalf("target unit = %d, want 2", input.UnitID)
	}
	if input.SerialNumber != "MOVE-SN" {
		t.Fatalf("serial = %q, want MOVE-SN", input.SerialNumber)
	}
	if input.InstalledHM != 50 || input.InstalledKM != 100 {
		t.Fatalf("target installed HM/KM = %v/%v, want 50/100", input.InstalledHM, input.InstalledKM)
	}
	if input.LifetimeHM == nil || *input.LifetimeHM != 1000 {
		t.Fatalf("lifetime HM = %v, want 1000", input.LifetimeHM)
	}
	if input.LifetimeDays == nil || *input.LifetimeDays != 90 {
		t.Fatalf("lifetime days = %v, want 90", input.LifetimeDays)
	}
}

func TestComponentLifetimeState(t *testing.T) {
	cases := []struct {
		name         string
		status       string
		usedHM       float64
		usedKM       float64
		usedDays     float64
		lifetimeHM   *float64
		lifetimeKM   *float64
		lifetimeDays *int
		want         string
	}{
		{name: "inactive", status: installationStatusRemoved, want: "inactive"},
		{name: "unset", status: installationStatusActive, want: "unset"},
		{name: "safe", status: installationStatusActive, usedHM: 100, lifetimeHM: floatPointer(1000), want: "safe"},
		{name: "upcoming", status: installationStatusActive, usedHM: 850, lifetimeHM: floatPointer(1000), want: "upcoming"},
		{name: "due", status: installationStatusActive, usedHM: 950, lifetimeHM: floatPointer(1000), want: "due"},
		{name: "overdue", status: installationStatusActive, usedHM: 1001, lifetimeHM: floatPointer(1000), want: "overdue"},
		{name: "days upcoming", status: installationStatusActive, usedDays: 340, lifetimeDays: intPointerValue(365), want: "upcoming"},
		{name: "days due", status: installationStatusActive, usedDays: 360, lifetimeDays: intPointerValue(365), want: "due"},
		{name: "days overdue", status: installationStatusActive, usedDays: 366, lifetimeDays: intPointerValue(365), want: "overdue"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := componentLifetimeState(test.status, test.usedHM, test.usedKM, test.usedDays,
				test.lifetimeHM, test.lifetimeKM, test.lifetimeDays)
			if got != test.want {
				t.Fatalf("componentLifetimeState() = %q, want %q", got, test.want)
			}
		})
	}
}

func intPointerValue(value int) *int {
	return &value
}
