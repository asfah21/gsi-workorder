package candidates

import "context"

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

func (s *Service) Approve(ctx context.Context, id, actorID int64, input ApproveInput) (ApproveResult, error) {
	if id <= 0 || actorID <= 0 {
		return ApproveResult{}, ErrValidation
	}
	return s.repo.Approve(ctx, id, actorID, input)
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
