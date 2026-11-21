package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gsi-workorder/internal/auth"
	"gsi-workorder/internal/candidates"
)

type fakeCandidateService struct {
	skipErr error
	bulkErr error
}

func (f *fakeCandidateService) List(context.Context, candidates.Filter) ([]candidates.Candidate, int, error) {
	return nil, 0, nil
}

func (f *fakeCandidateService) Get(context.Context, int64) (candidates.Detail, error) {
	return candidates.Detail{}, nil
}

func (f *fakeCandidateService) Approve(context.Context, int64, int64, candidates.ApproveInput) (candidates.ApproveResult, error) {
	return candidates.ApproveResult{}, nil
}

func (f *fakeCandidateService) Skip(context.Context, int64, int64, candidates.SkipInput) (candidates.Candidate, error) {
	return candidates.Candidate{}, f.skipErr
}

func (f *fakeCandidateService) Bulk(context.Context, int64, candidates.BulkInput) (candidates.BulkResult, error) {
	return candidates.BulkResult{}, f.bulkErr
}

func candidateTestRouter(service CandidateService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return NewRouterWithM2(
		testLogger(),
		CheckFunc(func(context.Context) error { return nil }),
		&fakeAuthService{authUser: auth.User{ID: 1, Role: auth.RoleAdmin, IsActive: true}},
		nil,
		M2Services{Candidates: service},
	)
}

func TestSkipValidationReturns400(t *testing.T) {
	service := &fakeCandidateService{
		skipErr: &candidates.ValidationError{Fields: map[string]string{"reason": "Alasan minimal 5 karakter."}},
	}
	router := candidateTestRouter(service)

	request := httptest.NewRequest(http.MethodPost, "/api/wo-candidates/1/skip", strings.NewReader(`{"reason":"x"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestBulkValidationRemains422(t *testing.T) {
	service := &fakeCandidateService{
		bulkErr: &candidates.ValidationError{Fields: map[string]string{"ids": "Pilih 1 sampai 50 usulan."}},
	}
	router := candidateTestRouter(service)

	request := httptest.NewRequest(http.MethodPost, "/api/wo-candidates/bulk", strings.NewReader(`{"ids":[1],"action":"approve"}`))
	request.Header.Set("Authorization", "Bearer good")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}
