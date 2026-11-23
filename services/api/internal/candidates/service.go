package candidates

import (
	"context"
	"strings"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Candidate, int, error) {
	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > MaxPageSize {
		return nil, 0, ErrValidation
	}
	return s.repo.List(ctx, filter)
}

func (s *Service) Get(ctx context.Context, id int64) (Detail, error) {
	if id <= 0 {
		return Detail{}, ErrNotFound
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Candidate, error) {
	fields := map[string]string{}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	input.Priority = strings.TrimSpace(input.Priority)
	if input.ActorID <= 0 {
		return Candidate{}, ErrValidation
	}
	if input.UnitID <= 0 {
		fields["unit_id"] = "Unit wajib dipilih."
	}
	if input.Title == "" {
		fields["title"] = "Judul Work Request wajib diisi."
	} else if len(input.Title) > 200 {
		fields["title"] = "Judul maksimal 200 karakter."
	}
	if len(input.Description) > 2000 {
		fields["description"] = "Deskripsi maksimal 2000 karakter."
	}
	if !ValidWorkCategory(input.WorkCategory) {
		fields["work_category"] = "Kategori Work Request tidak valid."
	}
	if input.Priority == "" {
		input.Priority = "medium"
	}
	if !ValidPriority(input.Priority) {
		fields["suggested_priority"] = "Prioritas tidak valid."
	}
	if len(fields) > 0 {
		return Candidate{}, &ValidationError{Fields: fields}
	}
	return s.repo.Create(ctx, input)
}

func (s *Service) Validate(ctx context.Context, id, actorID int64, input ValidateInput) (Candidate, error) {
	if id <= 0 || actorID <= 0 {
		return Candidate{}, ErrValidation
	}
	if len(strings.TrimSpace(input.Note)) > maxTextLen {
		return Candidate{}, &ValidationError{Fields: map[string]string{"note": "Catatan maksimal 1000 karakter."}}
	}
	return s.repo.Validate(ctx, id, actorID, input)
}

func (s *Service) Approve(ctx context.Context, id, actorID int64, input ApproveInput) (ApproveResult, error) {
	if id <= 0 || actorID <= 0 {
		return ApproveResult{}, ErrValidation
	}
	return s.repo.Approve(ctx, id, actorID, input)
}

func (s *Service) Convert(ctx context.Context, id, actorID int64, input ApproveInput) (ApproveResult, error) {
	if id <= 0 || actorID <= 0 {
		return ApproveResult{}, ErrValidation
	}
	return s.repo.Convert(ctx, id, actorID, input)
}

func (s *Service) Skip(ctx context.Context, id, actorID int64, input SkipInput) (Candidate, error) {
	if id <= 0 || actorID <= 0 {
		return Candidate{}, ErrValidation
	}
	return s.repo.Skip(ctx, id, actorID, input)
}

func (s *Service) Bulk(ctx context.Context, actorID int64, input BulkInput) (BulkResult, error) {
	if actorID <= 0 {
		return BulkResult{}, ErrValidation
	}
	return s.repo.Bulk(ctx, actorID, input)
}
