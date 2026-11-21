package p2h

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

const pageSize = 100

type OdooClient interface {
	Configured() bool
	List(ctx context.Context, limit, offset int) (ListPage, error)
	Get(ctx context.Context, id int64) (Report, error)
}

// DetailLoader fetches report headers and findings in batches. The REST client
// uses Get; the JSON-RPC client uses LoadDetails so it can avoid photo fields.
type DetailLoader interface {
	LoadDetails(ctx context.Context, ids []int64) ([]Report, error)
}

type SyncService struct {
	client      OdooClient
	detail      DetailLoader
	repo        *Repository
	logger      *slog.Logger
	concurrency int
	loc         *time.Location
}

func NewSyncService(client OdooClient, repo *Repository, concurrency int, loc *time.Location, logger *slog.Logger) *SyncService {
	return NewSyncServiceWithDetail(client, nil, repo, concurrency, loc, logger)
}

func NewSyncServiceWithDetail(client OdooClient, detail DetailLoader, repo *Repository, concurrency int, loc *time.Location, logger *slog.Logger) *SyncService {
	if logger == nil {
		logger = slog.Default()
	}
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 4 {
		concurrency = 4
	}
	return &SyncService{
		client:      client,
		detail:      detail,
		repo:        repo,
		logger:      logger,
		concurrency: concurrency,
		loc:         loc,
	}
}

// Sync pulls P2H metadata, details, findings, and candidates. forceDetails is
// used by the sync-full CLI; periodic sync only refreshes new/changed reports.
func (s *SyncService) Sync(ctx context.Context, forceDetails bool) error {
	if !s.client.Configured() {
		return ErrNotConfigured
	}
	release, locked, err := s.repo.TrySyncLock(ctx)
	if err != nil {
		return err
	}
	if !locked {
		s.logger.Warn("P2H sync skipped: another sync is running")
		return ErrSyncInProgress
	}
	defer release()

	startedAt := time.Now()
	runID, err := s.repo.StartSyncRun(ctx, "p2h")
	if err != nil {
		return err
	}

	counters := SyncCounters{}
	finish := func(status, message string) {
		finishCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.repo.FinishSyncRun(finishCtx, runID, status, message, counters); err != nil {
			s.logger.Error("finish P2H sync run failed", "error", err)
		}
		stateStatus := status
		_ = s.repo.SetSyncState(finishCtx, "p2h_last_synced_at", time.Now().UTC().Format(time.RFC3339))
		_ = s.repo.SetSyncState(finishCtx, "p2h_last_status", stateStatus)
		_ = s.repo.SetSyncState(finishCtx, "p2h_last_reports", fmt.Sprint(counters.Reports))
		_ = s.repo.SetSyncState(finishCtx, "p2h_last_details", fmt.Sprint(counters.Details))
		_ = s.repo.SetSyncState(finishCtx, "p2h_last_candidates", fmt.Sprint(counters.Candidates))
	}

	settings, err := s.repo.LoadSettings(ctx)
	if err != nil {
		finish("failed", truncate(err.Error()))
		return err
	}

	detailIDs := make(map[int64]struct{})
	offset := 0
	var firstErr error
	for {
		page, err := s.client.List(ctx, pageSize, offset)
		if err != nil {
			firstErr = fmt.Errorf("list P2H reports at offset %d: %w", offset, err)
			break
		}
		for _, report := range page.Reports {
			result, err := s.repo.UpsertMetadata(ctx, report)
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				s.logger.Error("upsert P2H report failed", "odoo_report_id", report.OdooReportID, "error", err)
				continue
			}
			counters.Reports++
			if forceDetails || result.NeedsDetail {
				detailIDs[report.OdooReportID] = struct{}{}
			}
		}
		if len(page.Reports) == 0 {
			break
		}
		offset += len(page.Reports)
		if page.Total >= 0 && offset >= page.Total {
			break
		}
		if page.Total < 0 && len(page.Reports) < pageSize {
			break
		}
	}

	if len(detailIDs) > 0 {
		synced, detailErr := s.syncDetails(ctx, detailIDs, settings, &counters)
		counters.Details += synced
		if firstErr == nil {
			firstErr = detailErr
		}
	}

	if firstErr != nil {
		finish("failed", truncate(firstErr.Error()))
		return firstErr
	}
	finish("success", "")
	s.logger.Info("P2H sync completed",
		"reports", counters.Reports,
		"details", counters.Details,
		"findings", counters.Findings,
		"candidates", counters.Candidates,
		"work_orders", counters.WorkOrders,
		"duration_ms", time.Since(startedAt).Milliseconds(),
	)
	return nil
}

func (s *SyncService) syncDetails(ctx context.Context, ids map[int64]struct{}, settings SettingsValues, counters *SyncCounters) (int, error) {
	if s.detail != nil {
		return s.syncDetailsBatch(ctx, ids, settings, counters)
	}
	return s.syncDetailsREST(ctx, ids, settings, counters)
}

func (s *SyncService) syncDetailsBatch(ctx context.Context, ids map[int64]struct{}, settings SettingsValues, counters *SyncCounters) (int, error) {
	allIDs := make([]int64, 0, len(ids))
	for id := range ids {
		allIDs = append(allIDs, id)
	}
	sort.Slice(allIDs, func(i, j int) bool { return allIDs[i] < allIDs[j] })

	total := len(allIDs)
	processed := 0
	var details int
	var findings int
	var candidates int
	var workOrders int
	var firstErr error
	errorCount := 0
	applyCounters := func() {
		counters.Findings += findings
		counters.Candidates += candidates
		counters.WorkOrders += workOrders
	}

	recordError := func(err error) {
		errorCount++
		if firstErr == nil {
			firstErr = err
		}
		s.logger.Error("sync P2H detail failed", "error", err)
	}

	for start := 0; start < len(allIDs); start += jsonRPCDetailBatchSize {
		if err := ctx.Err(); err != nil {
			applyCounters()
			return details, err
		}
		end := start + jsonRPCDetailBatchSize
		if end > len(allIDs) {
			end = len(allIDs)
		}
		batch := allIDs[start:end]

		reports, err := s.detail.LoadDetails(ctx, batch)
		if err != nil {
			recordError(fmt.Errorf("load P2H detail batch: %w", err))
			if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrNotConfigured) {
				applyCounters()
				return details, firstErr
			}
			processed += len(batch)
			continue
		}

		seen := make(map[int64]struct{}, len(reports))
		for _, report := range reports {
			seen[report.OdooReportID] = struct{}{}
			if _, err := s.repo.UpsertMetadata(ctx, report); err != nil {
				recordError(fmt.Errorf("upsert P2H report %d detail metadata: %w", report.OdooReportID, err))
				continue
			}
			result, err := s.repo.SaveDetail(ctx, report, settings, time.Now())
			if err != nil {
				recordError(fmt.Errorf("save P2H report %d: %w", report.OdooReportID, err))
				continue
			}
			details++
			findings += result.Findings
			if result.Candidate != nil {
				candidates++
				if result.Candidate.AutoWO {
					workOrders++
				}
			}
		}
		for _, id := range batch {
			if _, ok := seen[id]; !ok {
				recordError(fmt.Errorf("P2H report %d is missing from detail batch", id))
			}
		}

		processed += len(batch)
		if processed%100 == 0 || processed == total {
			s.logger.Info("P2H detail progress",
				"processed", processed,
				"total", total,
				"details", details,
				"errors", errorCount,
			)
		}
	}

	applyCounters()
	return details, firstErr
}

func (s *SyncService) syncDetailsREST(ctx context.Context, ids map[int64]struct{}, settings SettingsValues, counters *SyncCounters) (int, error) {
	jobs := make(chan int64)
	limiter := time.NewTicker(time.Second)
	defer limiter.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var details int
	var findings int
	var candidates int
	var workOrders int

	recordError := func(err error) {
		mu.Lock()
		if firstErr == nil {
			firstErr = err
		}
		mu.Unlock()
		s.logger.Error("sync P2H detail failed", "error", err)
	}

	worker := func() {
		defer wg.Done()
		for id := range jobs {
			select {
			case <-ctx.Done():
				return
			case <-limiter.C:
			}
			report, err := s.client.Get(ctx, id)
			if err != nil {
				recordError(fmt.Errorf("get P2H report %d: %w", id, err))
				continue
			}
			result, err := s.repo.SaveDetail(ctx, report, settings, time.Now())
			if err != nil {
				recordError(fmt.Errorf("save P2H report %d: %w", id, err))
				continue
			}
			mu.Lock()
			details++
			findings += result.Findings
			if result.Candidate != nil {
				candidates++
				if result.Candidate.AutoWO {
					workOrders++
				}
			}
			mu.Unlock()
		}
	}

	wg.Add(s.concurrency)
	for i := 0; i < s.concurrency; i++ {
		go worker()
	}
	for id := range ids {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			mu.Lock()
			counters.Findings += findings
			counters.Candidates += candidates
			counters.WorkOrders += workOrders
			mu.Unlock()
			return details, ctx.Err()
		case jobs <- id:
		}
	}
	close(jobs)
	wg.Wait()

	counters.Findings += findings
	counters.Candidates += candidates
	counters.WorkOrders += workOrders
	return details, firstErr
}

func (s *SyncService) SendReminders(ctx context.Context) error {
	settings, err := s.repo.LoadSettings(ctx)
	if err != nil {
		return err
	}
	candidates, notificationsCreated, err := s.repo.SendCandidateReminders(ctx, time.Now(), settings.SLAHours)
	if err != nil {
		return err
	}
	if candidates > 0 {
		s.logger.Info("P2H reminders sent", "candidates", candidates, "notifications", notificationsCreated)
	}
	return nil
}

type ReportService struct {
	repo *Repository
}

func NewReportService(repo *Repository) *ReportService {
	return &ReportService{repo: repo}
}

func (s *ReportService) List(ctx context.Context, filter ReportFilter) ([]Report, int, error) {
	return s.repo.ListReports(ctx, filter)
}

func (s *ReportService) Get(ctx context.Context, id int64) (Report, error) {
	if id <= 0 {
		return Report{}, ErrReportNotFound
	}
	return s.repo.GetReport(ctx, id)
}

func truncate(message string) string {
	message = strings.TrimSpace(message)
	if len(message) <= 1000 {
		return message
	}
	return message[:1000]
}
