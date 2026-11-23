package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"gsi-workorder/internal/units"
)

func TestM8ComponentInstallations(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	truncateM6(t, ctx, pool)

	var adminID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (nik, name, role, is_active)
		VALUES ('m8-admin', 'Admin M8', 'admin', true) RETURNING id
	`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}

	var unitA, unitB int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, current_hm, current_km, is_active)
		VALUES ('UNIT-M8-A', 'dump_truck', 100, 1000, true) RETURNING id
	`).Scan(&unitA); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO units (code, unit_type, current_hm, current_km, is_active)
		VALUES ('UNIT-M8-B', 'dump_truck', 10, 100, true) RETURNING id
	`).Scan(&unitB); err != nil {
		t.Fatal(err)
	}

	var componentID int64
	if err := pool.QueryRow(ctx, `
		INSERT INTO components (
			code, name, category, component_class,
			default_lifetime_hm, default_lifetime_days, default_lifetime_km
		)
		VALUES ('ENG-M8', 'Engine M8', 'Engine', 'major', 500, 365, 5000)
		RETURNING id
	`).Scan(&componentID); err != nil {
		t.Fatal(err)
	}

	repo := units.NewPostgresRepository(pool)
	service := units.NewService(repo, nil)

	installed, err := service.InstallComponent(ctx, unitA, units.ComponentInstallationInput{
		ComponentID:  componentID,
		SerialNumber: "SN-M8-001",
		Notes:        "initial install",
	}, adminID)
	if err != nil {
		t.Fatalf("InstallComponent() error = %v", err)
	}
	if installed.UnitID != unitA || installed.ComponentID != componentID {
		t.Fatalf("installed unit/component = %d/%d", installed.UnitID, installed.ComponentID)
	}
	if installed.InstalledHM != 100 || installed.InstalledKM != 1000 {
		t.Fatalf("installed HM/KM = %v/%v, want 100/1000", installed.InstalledHM, installed.InstalledKM)
	}
	if installed.LifetimeHM == nil || *installed.LifetimeHM != 500 {
		t.Fatalf("lifetime HM = %v, want 500", installed.LifetimeHM)
	}
	if installed.CreatedBy == nil || *installed.CreatedBy != adminID {
		t.Fatalf("created by = %v, want %d", installed.CreatedBy, adminID)
	}

	lifetime, err := service.ListComponentLifetime(ctx, unitA, true)
	if err != nil {
		t.Fatalf("ListComponentLifetime() error = %v", err)
	}
	if len(lifetime) != 1 {
		t.Fatalf("lifetime rows = %d, want 1", len(lifetime))
	}
	if lifetime[0].UsedHM != 0 || lifetime[0].RemainingHM == nil || *lifetime[0].RemainingHM != 500 {
		t.Fatalf("initial lifetime used/remaining = %v/%v", lifetime[0].UsedHM, lifetime[0].RemainingHM)
	}
	if lifetime[0].LifetimeState != "safe" {
		t.Fatalf("initial lifetime state = %q, want safe", lifetime[0].LifetimeState)
	}

	if _, err := pool.Exec(ctx, `
		UPDATE units SET current_hm = 200, current_km = 1200 WHERE id = $1
	`, unitA); err != nil {
		t.Fatal(err)
	}
	lifetime, err = service.ListComponentLifetime(ctx, unitA, true)
	if err != nil {
		t.Fatalf("ListComponentLifetime() after reading error = %v", err)
	}
	if len(lifetime) != 1 || lifetime[0].UsedHM != 100 {
		t.Fatalf("lifetime after reading = %#v, want used_hm 100", lifetime)
	}

	replacement, err := service.ReplaceComponent(ctx, unitA, installed.ID, units.ComponentReplacementInput{
		SerialNumber: "SN-M8-002",
		Notes:        "replace engine",
	}, adminID)
	if err != nil {
		t.Fatalf("ReplaceComponent() error = %v", err)
	}
	if replacement.SerialNumber != "SN-M8-002" || replacement.UnitID != unitA {
		t.Fatalf("replacement = %#v", replacement)
	}
	old, err := service.GetComponentInstallation(ctx, unitA, installed.ID)
	if err != nil {
		t.Fatalf("GetComponentInstallation(old) error = %v", err)
	}
	if old.Status != "removed" || old.RemovedHM == nil || *old.RemovedHM != 200 {
		t.Fatalf("old installation = %#v", old)
	}

	transferred, err := service.TransferComponent(ctx, unitA, replacement.ID, units.ComponentTransferInput{
		ToUnitID: unitB,
		Notes:    "move engine",
	}, adminID)
	if err != nil {
		t.Fatalf("TransferComponent() error = %v", err)
	}
	if transferred.UnitID != unitB || transferred.SerialNumber != "SN-M8-002" {
		t.Fatalf("transferred = %#v", transferred)
	}
	if transferred.InstalledHM != 10 || transferred.InstalledKM != 100 {
		t.Fatalf("transferred installed HM/KM = %v/%v, want 10/100", transferred.InstalledHM, transferred.InstalledKM)
	}
	oldReplacement, err := service.GetComponentInstallation(ctx, unitA, replacement.ID)
	if err != nil {
		t.Fatalf("GetComponentInstallation(replacement) error = %v", err)
	}
	if oldReplacement.Status != "transferred" {
		t.Fatalf("replacement status = %q, want transferred", oldReplacement.Status)
	}

	history, err := service.ListComponentInstallations(ctx, unitA, false)
	if err != nil {
		t.Fatalf("ListComponentInstallations(unit A) error = %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("unit A installation history = %d, want 2", len(history))
	}
	activeTarget, err := service.ListComponentInstallations(ctx, unitB, true)
	if err != nil {
		t.Fatalf("ListComponentInstallations(unit B) error = %v", err)
	}
	if len(activeTarget) != 1 || activeTarget[0].ID != transferred.ID {
		t.Fatalf("unit B active installations = %#v", activeTarget)
	}

	_, err = service.InstallComponent(ctx, unitB, units.ComponentInstallationInput{
		ComponentID:  componentID,
		SerialNumber: "SN-M8-003",
	}, adminID)
	if !errors.Is(err, units.ErrConflict) {
		t.Fatalf("duplicate active component error = %v, want ErrConflict", err)
	}
}
