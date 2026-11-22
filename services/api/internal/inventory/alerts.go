package inventory

import (
	"context"
	"fmt"
	"time"

	"gsi-workorder/internal/notifications"
)

func (r *Repository) SendLowStockAlerts(ctx context.Context, now time.Time) (StockAlertResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sparepart_id, sparepart_code, sparepart_name, uom,
		       warehouse_location_id, warehouse_code, location_code, location_name,
		       available::float8, min_qty::float8, critical_qty::float8,
		       stock_status, policy_id
		FROM v_low_stock
		WHERE policy_id IS NOT NULL
		  AND sparepart_active = true
		ORDER BY stock_status ASC, warehouse_code ASC, sparepart_code ASC
	`)
	if err != nil {
		return StockAlertResult{}, fmt.Errorf("list low stock alerts: %w", err)
	}
	type alert struct {
		SparepartID   int64
		SparepartCode string
		SparepartName string
		UOM           string
		LocationID    int64
		WarehouseCode string
		LocationCode  string
		LocationName  string
		Available     float64
		MinQty        float64
		CriticalQty   float64
		Status        string
		PolicyID      int64
	}
	alerts := make([]alert, 0)
	result := StockAlertResult{}
	for rows.Next() {
		var item alert
		if err := rows.Scan(&item.SparepartID, &item.SparepartCode, &item.SparepartName,
			&item.UOM, &item.LocationID, &item.WarehouseCode, &item.LocationCode,
			&item.LocationName, &item.Available, &item.MinQty, &item.CriticalQty,
			&item.Status, &item.PolicyID); err != nil {
			rows.Close()
			return StockAlertResult{}, fmt.Errorf("scan low stock alert: %w", err)
		}
		if item.Status == "critical" {
			result.Critical++
		} else {
			result.Low++
		}
		alerts = append(alerts, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return StockAlertResult{}, fmt.Errorf("iterate low stock alerts: %w", err)
	}
	rows.Close()
	if len(alerts) == 0 {
		return result, nil
	}

	day := now.In(r.loc).Format("2006-01-02")
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return StockAlertResult{}, fmt.Errorf("begin stock alert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, item := range alerts {
		title := "Stok rendah"
		if item.Status == "critical" {
			title = "Stok kritis"
		}
		title += ": " + item.SparepartCode + " - " + item.SparepartName
		body := fmt.Sprintf("%s / %s (%s): tersedia %.3f %s, minimum %.3f, critical %.3f.",
			item.WarehouseCode, item.LocationCode, item.LocationName,
			item.Available, item.UOM, item.MinQty, item.CriticalQty)
		policyID := item.PolicyID
		inserted, err := notifications.CreateForRoles(ctx, tx, []string{"storekeeper", "admin"},
			notifications.CreateInput{
				Type:      "stock_" + item.Status,
				Title:     title,
				Body:      body,
				RefType:   "inventory_stock",
				RefID:     &policyID,
				DedupeKey: fmt.Sprintf("stock:%d:%d:%s:%s", item.SparepartID, item.LocationID, item.Status, day),
			}, nil)
		if err != nil {
			return StockAlertResult{}, fmt.Errorf("create stock alert notification: %w", err)
		}
		result.Notified += inserted
	}
	if err := tx.Commit(ctx); err != nil {
		return StockAlertResult{}, fmt.Errorf("commit stock alerts: %w", err)
	}
	return result, nil
}
