package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"

	"redCollar/internal/api/handlers/http/admin"
	mock_admin "redCollar/internal/api/handlers/http/admin/mocks"
	"redCollar/internal/domain"
)

func newTestLogger() *slog.Logger {

	return slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), &slog.HandlerOptions{Level: slog.LevelError}))
}

func addChiURLParam(r *http.Request, key, val string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func decodeJSON[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json response: %v, body=%s", err, rr.Body.String())
	}
	return out
}

func TestAdminIncidentCreate_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	statsSvc := mock_admin.NewMockStatsGetter(ctrl)
	locSvc := mock_admin.NewMockLocationChecker(ctrl)

	h := admin.NewHandler(newTestLogger(), adminSvc, statsSvc, locSvc)

	reqBody := `{"lat":55.75,"lng":37.61,"radius_km":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/incidents/", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	wantID := uuid.New()

	adminSvc.EXPECT().
		Create(gomock.Any(), domain.CreateIncidentRequest{Lat: 55.75, Lng: 37.61, RadiusKM: 1}).
		Return(wantID, nil).
		Times(1)

	h.AdminIncidentCreate(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected %d got %d, body=%s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	got := decodeJSON[map[string]string](t, rr)
	if got["id"] != wantID.String() {
		t.Fatalf("expected id=%s got=%s", wantID.String(), got["id"])
	}
}

func TestAdminIncidentCreate_InvalidJSON_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/incidents/", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	h.AdminIncidentCreate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d, body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentCreate_ServiceError_500or4xx(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	reqBody := `{"lat":55.75,"lng":37.61,"radius_km":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/incidents/", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	adminSvc.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(uuid.Nil, errors.New("boom")).
		Times(1)

	h.AdminIncidentCreate(rr, req)

	if rr.Code < 400 {
		t.Fatalf("expected error status, got %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentList_Defaults_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/", nil)
	rr := httptest.NewRecorder()

	adminSvc.EXPECT().
		List(gomock.Any(), 1, 20).
		Return([]*domain.Incident{}, int64(0), nil).
		Times(1)

	h.AdminIncidentList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	resp := decodeJSON[map[string]any](t, rr)
	if int(resp["page"].(float64)) != 1 || int(resp["limit"].(float64)) != 20 {
		t.Fatalf("unexpected pagination: %+v", resp)
	}
}

func TestAdminIncidentList_LimitClampedTo100(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/?page=2&limit=500", nil)
	rr := httptest.NewRecorder()

	adminSvc.EXPECT().
		List(gomock.Any(), 2, 100).
		Return([]*domain.Incident{}, int64(0), nil).
		Times(1)

	h.AdminIncidentList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentGet_InvalidID_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/bad/", nil)
	req = addChiURLParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.AdminIncidentGet(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentGet_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	id := uuid.New()
	want := &domain.Incident{ID: id, Lat: 1, Lng: 2, RadiusKM: 3, Status: domain.IncidentActive}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/"+id.String()+"/", nil)
	req = addChiURLParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	adminSvc.EXPECT().
		Get(gomock.Any(), id).
		Return(want, nil).
		Times(1)

	h.AdminIncidentGet(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	got := decodeJSON[domain.Incident](t, rr)
	if got.ID != id {
		t.Fatalf("expected id=%s got=%s", id, got.ID)
	}
}

func TestAdminIncidentUpdate_InvalidID_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/incidents/bad/", bytes.NewBufferString(`{}`))
	req = addChiURLParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.AdminIncidentUpdate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentUpdate_InvalidJSON_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/incidents/"+id.String()+"/", bytes.NewBufferString("{bad"))
	req = addChiURLParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	h.AdminIncidentUpdate(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentUpdate_OK_204(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	id := uuid.New()
	body := `{"status":"inactive"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/incidents/"+id.String()+"/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = addChiURLParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	st := domain.IncidentInactive
	adminSvc.EXPECT().
		Update(gomock.Any(), id, domain.UpdateIncidentRequest{Status: &st}).
		Return(nil).
		Times(1)

	h.AdminIncidentUpdate(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d body=%s", http.StatusNoContent, rr.Code, rr.Body.String())
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("expected empty body, got=%q", rr.Body.String())
	}
}

func TestAdminIncidentDelete_InvalidID_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/incidents/bad/", nil)
	req = addChiURLParam(req, "id", "not-a-uuid")
	rr := httptest.NewRecorder()

	h.AdminIncidentDelete(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestAdminIncidentDelete_OK_204(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	adminSvc := mock_admin.NewMockAdminIncidents(ctrl)
	h := admin.NewHandler(newTestLogger(), adminSvc,
		mock_admin.NewMockStatsGetter(ctrl),
		mock_admin.NewMockLocationChecker(ctrl),
	)

	id := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/incidents/"+id.String()+"/", nil)
	req = addChiURLParam(req, "id", id.String())
	rr := httptest.NewRecorder()

	adminSvc.EXPECT().
		Delete(gomock.Any(), id).
		Return(nil).
		Times(1)

	h.AdminIncidentDelete(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected %d got %d body=%s", http.StatusNoContent, rr.Code, rr.Body.String())
	}
}

func TestAdminStats_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_admin.NewMockStatsGetter(ctrl)
	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		statsSvc,
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/stats?minutes=60", nil)
	rr := httptest.NewRecorder()

	want := &domain.IncidentStats{UserCount: 42}
	statsSvc.EXPECT().
		GetStats(gomock.Any(), domain.StatsRequest{Minutes: 60}).
		Return(want, nil).
		Times(1)

	h.AdminStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	got := decodeJSON[domain.IncidentStats](t, rr)
	if got.UserCount != 42 {
		t.Fatalf("expected user_count=42 got=%d", got.UserCount)
	}
}

func TestAdminStats_DefaultMinutes_60(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_admin.NewMockStatsGetter(ctrl)
	h := admin.NewHandler(newTestLogger(),
		mock_admin.NewMockAdminIncidents(ctrl),
		statsSvc,
		mock_admin.NewMockLocationChecker(ctrl),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/incidents/stats", nil)
	rr := httptest.NewRecorder()

	statsSvc.EXPECT().
		GetStats(gomock.Any(), domain.StatsRequest{Minutes: 60}).
		Return(&domain.IncidentStats{UserCount: 0}, nil).
		Times(1)

	h.AdminStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d", http.StatusOK, rr.Code)
	}
}
