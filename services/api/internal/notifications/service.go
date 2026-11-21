package notifications

import (
	"context"
	"errors"
)

const MaxMarkReadIDs = 100

var ErrValidation = errors.New("notifications: validation failed")

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID int64, page, limit int, unreadOnly bool) ([]Notification, int, int, error) {
	if userID <= 0 || page < 1 || limit < 1 {
		return nil, 0, 0, ErrValidation
	}
	return s.repo.List(ctx, userID, page, limit, unreadOnly)
}

func (s *Service) MarkRead(ctx context.Context, userID int64, ids []int64, all bool) (int64, error) {
	if userID <= 0 {
		return 0, ErrValidation
	}
	if !all && (len(ids) == 0 || len(ids) > MaxMarkReadIDs) {
		return 0, ErrValidation
	}
	for _, id := range ids {
		if id <= 0 {
			return 0, ErrValidation
		}
	}
	return s.repo.MarkRead(ctx, userID, ids, all)
}
