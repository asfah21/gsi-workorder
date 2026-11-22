package inventory

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"gsi-workorder/internal/notifications"
)

type balanceKey struct {
	SparepartID int64
	LocationID  int64
}

type balanceState struct {
	OnHand   float64
	Reserved float64
}

func (r *Repository) Post(ctx context.Context, actor Actor, input PostInput) (Transaction, error) {
	input.Type = strings.ToLower(trim(input.Type))
	if !canPostType(actor.Role, input.Type) {
		return Transaction{}, ErrValidation
	}
	input.IdempotencyKey = trim(input.IdempotencyKey)
	if input.IdempotencyKey != "" {
		existing, err := r.getTransactionByIdempotency(ctx, input.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Transaction{}, err
		}
	}
	normalized, err := normalizePostInput(input)
	if err != nil {
		return Transaction{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin inventory post: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if normalized.IdempotencyKey != "" {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1)::bigint)`, normalized.IdempotencyKey); err != nil {
			return Transaction{}, fmt.Errorf("lock inventory idempotency key: %w", err)
		}
		var existingID int64
		err := tx.QueryRow(ctx, `
			SELECT id FROM inventory_transactions WHERE idempotency_key = $1
		`, normalized.IdempotencyKey).Scan(&existingID)
		if err == nil {
			if err := tx.Rollback(ctx); err != nil {
				return Transaction{}, fmt.Errorf("rollback inventory duplicate: %w", err)
			}
			return r.GetTransaction(ctx, existingID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, fmt.Errorf("check inventory idempotency key: %w", err)
		}
	}
	if err := validatePostReferences(ctx, tx, normalized.Items); err != nil {
		return Transaction{}, err
	}
	id, err := r.postLedgerTx(ctx, tx, actor.ID, normalized.Type, normalized.ReferenceType,
		normalized.ReferenceID, normalized.ReferenceNo, normalized.Notes,
		normalized.IdempotencyKey, normalized.Items, nil)
	if err != nil {
		return Transaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit inventory post: %w", err)
	}
	return r.GetTransaction(ctx, id)
}

func (r *Repository) Reverse(ctx context.Context, id int64, actor Actor, reason string) (Transaction, error) {
	if !canApproveStock(actor.Role) {
		return Transaction{}, ErrValidation
	}
	reason = trim(reason)
	if reason == "" || len(reason) > 1000 {
		return Transaction{}, &ValidationError{Fields: map[string]string{"reason": "Alasan reversal wajib 1-1000 karakter."}}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Transaction{}, fmt.Errorf("begin inventory reversal: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var original Transaction
	err = tx.QueryRow(ctx, `
		SELECT id, code, type, status, reversal_of_id
		FROM inventory_transactions
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&original.ID, &original.Code, &original.Type, &original.Status, &original.ReversalOfID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("lock inventory transaction: %w", err)
	}
	if original.Status != "posted" {
		return Transaction{}, ErrInvalidState
	}
	if original.ReversalOfID != nil {
		return Transaction{}, ErrConflict
	}
	if original.Type == "reserve" || original.Type == "release" {
		return Transaction{}, ErrInvalidState
	}
	items, err := loadTransactionItems(ctx, tx, id)
	if err != nil {
		return Transaction{}, err
	}
	mirrored := make([]PostItem, 0, len(items))
	for _, item := range items {
		mirrored = append(mirrored, PostItem{
			SparepartID:    item.SparepartID,
			FromLocationID: item.ToLocationID,
			ToLocationID:   item.FromLocationID,
			Qty:            item.Qty,
			Notes:          item.Notes,
		})
	}
	if err := validateReversalReferences(ctx, tx, mirrored); err != nil {
		return Transaction{}, err
	}
	reversalID, err := r.postLedgerTx(ctx, tx, actor.ID, original.Type, "reversal",
		&id, original.Code, reason, "", mirrored, &id)
	if err != nil {
		return Transaction{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE inventory_transactions SET status = 'reversed' WHERE id = $1
	`, id); err != nil {
		return Transaction{}, fmt.Errorf("mark original transaction reversed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Transaction{}, fmt.Errorf("commit inventory reversal: %w", err)
	}
	return r.GetTransaction(ctx, reversalID)
}

func (r *Repository) postLedgerTx(
	ctx context.Context,
	tx pgx.Tx,
	actorID int64,
	transactionType string,
	referenceType string,
	referenceID *int64,
	referenceNo string,
	notes string,
	idempotencyKey string,
	items []PostItem,
	reversalOfID *int64,
) (int64, error) {
	if len(items) == 0 {
		return 0, &ValidationError{Fields: map[string]string{"items": "Minimal satu item transaksi wajib diisi."}}
	}
	deltas, err := buildDeltas(items)
	if err != nil {
		return 0, err
	}
	keys := make([]balanceKey, 0, len(deltas))
	for key := range deltas {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].SparepartID == keys[j].SparepartID {
			return keys[i].LocationID < keys[j].LocationID
		}
		return keys[i].SparepartID < keys[j].SparepartID
	})

	states := make(map[balanceKey]balanceState, len(keys))
	for _, key := range keys {
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_balances (sparepart_id, warehouse_location_id, on_hand, reserved)
			VALUES ($1, $2, 0, 0)
			ON CONFLICT (sparepart_id, warehouse_location_id) DO NOTHING
		`, key.SparepartID, key.LocationID); err != nil {
			return 0, fmt.Errorf("ensure inventory balance: %w", err)
		}
		var state balanceState
		if err := tx.QueryRow(ctx, `
			SELECT on_hand::float8, reserved::float8
			FROM inventory_balances
			WHERE sparepart_id = $1 AND warehouse_location_id = $2
			FOR UPDATE
		`, key.SparepartID, key.LocationID).Scan(&state.OnHand, &state.Reserved); err != nil {
			return 0, fmt.Errorf("lock inventory balance: %w", err)
		}
		state.OnHand += deltas[key]
		if state.OnHand < 0 || state.Reserved < 0 || state.Reserved > state.OnHand {
			return 0, ErrInsufficientStock
		}
		states[key] = state
	}

	code, err := nextTransactionCode(ctx, tx)
	if err != nil {
		return 0, err
	}
	var transactionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO inventory_transactions (
			code, type, status, reference_type, reference_id, reference_no,
			reversal_of_id, idempotency_key, posted_by, posted_at, notes
		)
		VALUES ($1, $2, 'posted', NULLIF($3, ''), $4, NULLIF($5, ''),
		        $6, NULLIF($7, ''), $8, now(), NULLIF($9, ''))
		RETURNING id
	`, code, transactionType, trim(referenceType), referenceID, trim(referenceNo),
		reversalOfID, trim(idempotencyKey), actorID, trim(notes)).Scan(&transactionID); err != nil {
		return 0, fmt.Errorf("insert inventory transaction: %w", err)
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_transaction_items (
				transaction_id, sparepart_id, from_location_id, to_location_id, qty, notes
			)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''))
		`, transactionID, item.SparepartID, item.FromLocationID, item.ToLocationID,
			item.Qty, trim(item.Notes)); err != nil {
			return 0, fmt.Errorf("insert inventory transaction item: %w", err)
		}
	}
	for _, key := range keys {
		state := states[key]
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET on_hand = $3, updated_at = now()
			WHERE sparepart_id = $1 AND warehouse_location_id = $2
		`, key.SparepartID, key.LocationID, state.OnHand); err != nil {
			return 0, fmt.Errorf("update inventory balance: %w", err)
		}
	}
	if reversalOfID == nil && (transactionType == "adjustment" || transactionType == "opname") {
		title := "Adjustment stok posted"
		if transactionType == "opname" {
			title = "Stock opname posted"
		}
		body := code
		if trim(notes) != "" {
			body += " - " + trim(notes)
		}
		_, err := notifications.CreateForRoles(ctx, tx, []string{"admin", "planner"},
			notifications.CreateInput{
				Type:      "inventory_" + transactionType,
				Title:     title,
				Body:      body,
				RefType:   "inventory_transaction",
				RefID:     &transactionID,
				DedupeKey: fmt.Sprintf("inv:%d:posted", transactionID),
			}, &actorID)
		if err != nil {
			return 0, fmt.Errorf("notify inventory posting: %w", err)
		}
	}
	return transactionID, nil
}

func (r *Repository) ListTransactions(ctx context.Context, filter TransactionFilter) ([]Transaction, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 5)
	where := []string{"1 = 1"}
	if filter.Type != "" {
		args = append(args, filter.Type)
		where = append(where, fmt.Sprintf("t.type = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+trim(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(t.code ILIKE $%d OR COALESCE(t.reference_no, '') ILIKE $%d
			OR COALESCE(t.notes, '') ILIKE $%d)`, n, n, n))
	}
	if filter.From != nil {
		args = append(args, filter.From.Format("2006-01-02"))
		where = append(where, fmt.Sprintf("t.posted_at >= ($%d::date AT TIME ZONE 'Asia/Makassar')", len(args)))
	}
	if filter.To != nil {
		args = append(args, filter.To.Format("2006-01-02"))
		where = append(where, fmt.Sprintf("t.posted_at < (($%d::date + 1) AT TIME ZONE 'Asia/Makassar')", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM inventory_transactions t WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inventory transactions: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, transactionSelect+`
		WHERE `+whereSQL+`
		ORDER BY t.posted_at DESC, t.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list inventory transactions: %w", err)
	}
	defer rows.Close()
	items := make([]Transaction, 0)
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan inventory transaction: %w", err)
		}
		lines, err := r.listTransactionItems(ctx, item.ID)
		if err != nil {
			return nil, 0, err
		}
		item.Items = lines
		item.ItemCount = len(lines)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate inventory transactions: %w", err)
	}
	return items, total, nil
}

func (r *Repository) GetTransaction(ctx context.Context, id int64) (Transaction, error) {
	item, err := scanTransaction(r.pool.QueryRow(ctx, transactionSelect+` WHERE t.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	if err != nil {
		return Transaction{}, fmt.Errorf("get inventory transaction: %w", err)
	}
	lines, err := r.listTransactionItems(ctx, id)
	if err != nil {
		return Transaction{}, err
	}
	item.Items = lines
	item.ItemCount = len(lines)
	return item, nil
}

func (r *Repository) getTransactionByIdempotency(ctx context.Context, key string) (Transaction, error) {
	var id int64
	if err := r.pool.QueryRow(ctx, `
		SELECT id FROM inventory_transactions WHERE idempotency_key = $1
	`, key).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, fmt.Errorf("get inventory transaction by idempotency key: %w", err)
	}
	return r.GetTransaction(ctx, id)
}

const transactionSelect = `
	SELECT t.id, t.code, t.type, t.status, COALESCE(t.reference_type, ''),
	       t.reference_id, COALESCE(t.reference_no, ''), t.reversal_of_id,
	       COALESCE(t.idempotency_key, ''), t.posted_by, COALESCE(u.name, ''),
	       t.posted_at, COALESCE(t.notes, ''),
	       (SELECT count(*) FROM inventory_transaction_items ti WHERE ti.transaction_id = t.id),
	       t.created_at
	FROM inventory_transactions t
	LEFT JOIN users u ON u.id = t.posted_by
`

func scanTransaction(row interface{ Scan(...any) error }) (Transaction, error) {
	var item Transaction
	if err := row.Scan(&item.ID, &item.Code, &item.Type, &item.Status, &item.ReferenceType,
		&item.ReferenceID, &item.ReferenceNo, &item.ReversalOfID, &item.IdempotencyKey,
		&item.PostedBy, &item.PostedByName, &item.PostedAt, &item.Notes,
		&item.ItemCount, &item.CreatedAt); err != nil {
		return Transaction{}, err
	}
	item.Items = make([]TransactionItem, 0)
	return item, nil
}

func (r *Repository) listTransactionItems(ctx context.Context, transactionID int64) ([]TransactionItem, error) {
	return loadTransactionItems(ctx, r.pool, transactionID)
}

type rowQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func loadTransactionItems(ctx context.Context, q rowQuerier, transactionID int64) ([]TransactionItem, error) {
	rows, err := q.Query(ctx, `
		SELECT ti.id, ti.transaction_id, ti.sparepart_id, s.code, s.name, s.uom,
		       ti.from_location_id,
		       TRIM(COALESCE(fw.code, '') || CASE WHEN fl.code IS NULL THEN '' ELSE ' / ' || fl.code END),
		       ti.to_location_id,
		       TRIM(COALESCE(tw.code, '') || CASE WHEN tl.code IS NULL THEN '' ELSE ' / ' || tl.code END),
		       ti.qty::float8, COALESCE(ti.notes, '')
		FROM inventory_transaction_items ti
		JOIN spareparts s ON s.id = ti.sparepart_id
		LEFT JOIN warehouse_locations fl ON fl.id = ti.from_location_id
		LEFT JOIN warehouses fw ON fw.id = fl.warehouse_id
		LEFT JOIN warehouse_locations tl ON tl.id = ti.to_location_id
		LEFT JOIN warehouses tw ON tw.id = tl.warehouse_id
		WHERE ti.transaction_id = $1
		ORDER BY ti.id ASC
	`, transactionID)
	if err != nil {
		return nil, fmt.Errorf("list inventory transaction items: %w", err)
	}
	defer rows.Close()
	items := make([]TransactionItem, 0)
	for rows.Next() {
		var item TransactionItem
		if err := rows.Scan(&item.ID, &item.TransactionID, &item.SparepartID,
			&item.SparepartCode, &item.SparepartName, &item.UOM,
			&item.FromLocationID, &item.FromLocation, &item.ToLocationID,
			&item.ToLocation, &item.Qty, &item.Notes); err != nil {
			return nil, fmt.Errorf("scan inventory transaction item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inventory transaction items: %w", err)
	}
	return items, nil
}

func (r *Repository) ListMovements(ctx context.Context, filter MovementFilter) ([]Movement, int, error) {
	filter.Page, filter.Limit = normalizePage(filter.Page, filter.Limit)
	args := make([]any, 0, 8)
	where := []string{"1 = 1"}
	if filter.Type != "" {
		args = append(args, filter.Type)
		where = append(where, fmt.Sprintf("t.type = $%d", len(args)))
	}
	if filter.Query != "" {
		args = append(args, "%"+trim(filter.Query)+"%")
		n := len(args)
		where = append(where, fmt.Sprintf(`(t.code ILIKE $%d OR COALESCE(t.reference_no, '') ILIKE $%d
			OR s.code ILIKE $%d OR s.name ILIKE $%d OR COALESCE(ti.notes, '') ILIKE $%d)`,
			n, n, n, n, n))
	}
	if filter.SparepartID != nil {
		args = append(args, *filter.SparepartID)
		where = append(where, fmt.Sprintf("ti.sparepart_id = $%d", len(args)))
	}
	if filter.LocationID != nil {
		args = append(args, *filter.LocationID)
		where = append(where, fmt.Sprintf("(ti.from_location_id = $%d OR ti.to_location_id = $%d)",
			len(args), len(args)))
	}
	if filter.From != nil {
		args = append(args, filter.From.Format("2006-01-02"))
		where = append(where, fmt.Sprintf("t.posted_at >= ($%d::date AT TIME ZONE 'Asia/Makassar')", len(args)))
	}
	if filter.To != nil {
		args = append(args, filter.To.Format("2006-01-02"))
		where = append(where, fmt.Sprintf("t.posted_at < (($%d::date + 1) AT TIME ZONE 'Asia/Makassar')", len(args)))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM inventory_transaction_items ti
		JOIN inventory_transactions t ON t.id = ti.transaction_id
		JOIN spareparts s ON s.id = ti.sparepart_id
		WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inventory movements: %w", err)
	}
	args = append(args, filter.Limit, (filter.Page-1)*filter.Limit)
	rows, err := r.pool.Query(ctx, `
		SELECT ti.id, t.id, t.code, t.type, t.status, COALESCE(t.reference_no, ''),
		       t.posted_at, ti.sparepart_id, s.code, s.name, s.uom,
		       ti.from_location_id,
		       TRIM(COALESCE(fw.code, '') || CASE WHEN fl.code IS NULL THEN '' ELSE ' / ' || fl.code END),
		       ti.to_location_id,
		       TRIM(COALESCE(tw.code, '') || CASE WHEN tl.code IS NULL THEN '' ELSE ' / ' || tl.code END),
		       ti.qty::float8, COALESCE(ti.notes, '')
		FROM inventory_transaction_items ti
		JOIN inventory_transactions t ON t.id = ti.transaction_id
		JOIN spareparts s ON s.id = ti.sparepart_id
		LEFT JOIN warehouse_locations fl ON fl.id = ti.from_location_id
		LEFT JOIN warehouses fw ON fw.id = fl.warehouse_id
		LEFT JOIN warehouse_locations tl ON tl.id = ti.to_location_id
		LEFT JOIN warehouses tw ON tw.id = tl.warehouse_id
		WHERE `+whereSQL+`
		ORDER BY t.posted_at DESC, ti.id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list inventory movements: %w", err)
	}
	defer rows.Close()
	items := make([]Movement, 0)
	for rows.Next() {
		var item Movement
		if err := rows.Scan(&item.ID, &item.TransactionID, &item.TransactionCode, &item.Type,
			&item.Status, &item.ReferenceNo, &item.PostedAt, &item.SparepartID,
			&item.SparepartCode, &item.SparepartName, &item.UOM,
			&item.FromLocationID, &item.FromLocation, &item.ToLocationID,
			&item.ToLocation, &item.Qty, &item.Notes); err != nil {
			return nil, 0, fmt.Errorf("scan inventory movement: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate inventory movements: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Reconcile(ctx context.Context) (Reconciliation, error) {
	rows, err := r.pool.Query(ctx, `
		WITH ledger_lines AS (
			SELECT ti.sparepart_id, ti.to_location_id AS location_id, ti.qty
			FROM inventory_transaction_items ti
			JOIN inventory_transactions t ON t.id = ti.transaction_id
			WHERE t.type NOT IN ('reserve', 'release')
			  AND ti.to_location_id IS NOT NULL
			UNION ALL
			SELECT ti.sparepart_id, ti.from_location_id AS location_id, -ti.qty
			FROM inventory_transaction_items ti
			JOIN inventory_transactions t ON t.id = ti.transaction_id
			WHERE t.type NOT IN ('reserve', 'release')
			  AND ti.from_location_id IS NOT NULL
		),
		ledger AS (
			SELECT sparepart_id, location_id, SUM(qty) AS qty
			FROM ledger_lines
			GROUP BY sparepart_id, location_id
		)
		SELECT COALESCE(b.sparepart_id, l.sparepart_id),
		       s.code, s.name,
		       COALESCE(b.warehouse_location_id, l.location_id),
		       w.code, wl.code,
		       COALESCE(b.on_hand, 0)::float8,
		       COALESCE(l.qty, 0)::float8,
		       (COALESCE(b.on_hand, 0) - COALESCE(l.qty, 0))::float8
		FROM inventory_balances b
		FULL OUTER JOIN ledger l
		       ON l.sparepart_id = b.sparepart_id
		      AND l.location_id = b.warehouse_location_id
		JOIN spareparts s ON s.id = COALESCE(b.sparepart_id, l.sparepart_id)
		JOIN warehouse_locations wl ON wl.id = COALESCE(b.warehouse_location_id, l.location_id)
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE COALESCE(b.on_hand, 0) <> COALESCE(l.qty, 0)
		ORDER BY s.code ASC, w.code ASC, wl.code ASC
	`)
	if err != nil {
		return Reconciliation{}, fmt.Errorf("reconcile inventory balances: %w", err)
	}
	defer rows.Close()
	result := Reconciliation{Rows: make([]ReconciliationRow, 0), CheckedAt: time.Now().UTC()}
	for rows.Next() {
		var item ReconciliationRow
		if err := rows.Scan(&item.SparepartID, &item.SparepartCode, &item.SparepartName,
			&item.LocationID, &item.WarehouseCode, &item.LocationCode,
			&item.BalanceOnHand, &item.LedgerOnHand, &item.Difference); err != nil {
			return Reconciliation{}, fmt.Errorf("scan inventory reconciliation: %w", err)
		}
		result.Rows = append(result.Rows, item)
	}
	if err := rows.Err(); err != nil {
		return Reconciliation{}, fmt.Errorf("iterate inventory reconciliation: %w", err)
	}
	result.Mismatch = len(result.Rows)
	return result, nil
}

func buildDeltas(items []PostItem) (map[balanceKey]float64, error) {
	deltas := make(map[balanceKey]float64, len(items)*2)
	for _, item := range items {
		if item.SparepartID <= 0 {
			return nil, &ValidationError{Fields: map[string]string{"items": "Sparepart wajib dipilih."}}
		}
		if item.Qty <= 0 {
			return nil, &ValidationError{Fields: map[string]string{"items": "Qty harus lebih dari 0."}}
		}
		if item.FromLocationID == nil && item.ToLocationID == nil {
			return nil, &ValidationError{Fields: map[string]string{"items": "Lokasi transaksi wajib diisi."}}
		}
		if item.FromLocationID != nil && *item.FromLocationID <= 0 {
			return nil, &ValidationError{Fields: map[string]string{"items": "Lokasi asal tidak valid."}}
		}
		if item.ToLocationID != nil && *item.ToLocationID <= 0 {
			return nil, &ValidationError{Fields: map[string]string{"items": "Lokasi tujuan tidak valid."}}
		}
		if item.FromLocationID != nil {
			key := balanceKey{SparepartID: item.SparepartID, LocationID: *item.FromLocationID}
			deltas[key] -= item.Qty
		}
		if item.ToLocationID != nil {
			key := balanceKey{SparepartID: item.SparepartID, LocationID: *item.ToLocationID}
			deltas[key] += item.Qty
		}
	}
	return deltas, nil
}

func normalizePostInput(input PostInput) (PostInput, error) {
	fields := map[string]string{}
	input.ReferenceType = trim(input.ReferenceType)
	input.ReferenceNo = trim(input.ReferenceNo)
	input.Notes = trim(input.Notes)
	if input.ReferenceType == "" {
		input.ReferenceType = "manual"
	}
	if len(input.ReferenceType) > 50 {
		fields["reference_type"] = "Tipe referensi maksimal 50 karakter."
	}
	if len(input.ReferenceNo) > 150 {
		fields["reference_no"] = "Nomor referensi maksimal 150 karakter."
	}
	if len(input.Notes) > 2000 {
		fields["notes"] = "Catatan maksimal 2000 karakter."
	}
	if len(input.Items) == 0 {
		fields["items"] = "Minimal satu item transaksi wajib diisi."
	}
	if len(input.Items) > 100 {
		fields["items"] = "Maksimal 100 item per transaksi."
	}
	if (input.Type == "issue" || input.Type == "adjustment") && input.Notes == "" {
		fields["notes"] = "Alasan wajib diisi untuk issue manual dan adjustment."
	}
	for _, item := range input.Items {
		switch input.Type {
		case "receipt", "return":
			if item.FromLocationID != nil || item.ToLocationID == nil {
				fields["items"] = "Receipt/return hanya boleh memiliki lokasi tujuan."
				break
			}
		case "issue":
			if item.FromLocationID == nil || item.ToLocationID != nil {
				fields["items"] = "Issue hanya boleh memiliki lokasi asal."
				break
			}
		case "transfer":
			if item.FromLocationID == nil || item.ToLocationID == nil {
				fields["items"] = "Transfer wajib memiliki lokasi asal dan tujuan."
				break
			}
			if *item.FromLocationID == *item.ToLocationID {
				fields["items"] = "Lokasi asal dan tujuan transfer harus berbeda."
			}
		case "adjustment":
			if (item.FromLocationID == nil) == (item.ToLocationID == nil) {
				fields["items"] = "Adjustment wajib memiliki tepat satu arah lokasi."
			}
		default:
			fields["type"] = "Tipe transaksi tidak valid untuk M5."
		}
		if item.Qty <= 0 {
			fields["items"] = "Qty harus lebih dari 0."
		}
	}
	if len(fields) > 0 {
		return PostInput{}, &ValidationError{Fields: fields}
	}
	return input, nil
}

func canPostType(role, transactionType string) bool {
	switch transactionType {
	case "receipt", "issue", "return", "transfer":
		return canMoveStock(role)
	case "adjustment":
		return canApproveStock(role)
	default:
		return false
	}
}

func validatePostReferences(ctx context.Context, tx pgx.Tx, items []PostItem) error {
	return validateReferences(ctx, tx, items, true)
}

// Reversal must remain possible after a master is deactivated, because the
// historical transaction is still part of the immutable ledger.
func validateReversalReferences(ctx context.Context, tx pgx.Tx, items []PostItem) error {
	return validateReferences(ctx, tx, items, false)
}

func validateReferences(ctx context.Context, tx pgx.Tx, items []PostItem, requireActive bool) error {
	sparepartSeen := make(map[int64]struct{})
	locationSeen := make(map[int64]struct{})
	sparepartSQL := `SELECT id FROM spareparts WHERE id = $1`
	locationSQL := `
		SELECT wl.id
		FROM warehouse_locations wl
		JOIN warehouses w ON w.id = wl.warehouse_id
		WHERE wl.id = $1`
	if requireActive {
		sparepartSQL += ` AND is_active = true`
		locationSQL += ` AND wl.is_active = true AND w.is_active = true`
	}
	for _, item := range items {
		if _, ok := sparepartSeen[item.SparepartID]; !ok {
			var exists int64
			if err := tx.QueryRow(ctx, sparepartSQL, item.SparepartID).Scan(&exists); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return &ValidationError{Fields: map[string]string{"sparepart_id": "Sparepart tidak ditemukan atau tidak aktif."}}
				}
				return fmt.Errorf("check inventory sparepart: %w", err)
			}
			sparepartSeen[item.SparepartID] = struct{}{}
		}
		for _, locationID := range []*int64{item.FromLocationID, item.ToLocationID} {
			if locationID == nil {
				continue
			}
			if _, ok := locationSeen[*locationID]; ok {
				continue
			}
			var exists int64
			if err := tx.QueryRow(ctx, locationSQL, *locationID).Scan(&exists); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return &ValidationError{Fields: map[string]string{"location_id": "Lokasi tidak ditemukan atau tidak aktif."}}
				}
				return fmt.Errorf("check inventory location: %w", err)
			}
			locationSeen[*locationID] = struct{}{}
		}
	}
	return nil
}

func nextTransactionCode(ctx context.Context, tx pgx.Tx) (string, error) {
	var code string
	if err := tx.QueryRow(ctx, `
		SELECT 'INV-' || to_char(now() AT TIME ZONE 'Asia/Makassar', 'YYYYMM') || '-'
		       || lpad(nextval('inventory_transaction_code_seq')::text, 4, '0')
	`).Scan(&code); err != nil {
		return "", fmt.Errorf("next inventory transaction code: %w", err)
	}
	return code, nil
}
