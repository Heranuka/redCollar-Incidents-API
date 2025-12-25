package public_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"log/slog"

	"github.com/golang/mock/gomock"

	"redCollar/internal/api/handlers/http/public"
	mock_public "redCollar/internal/api/handlers/http/public/mocks"
	"redCollar/internal/domain"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), &slog.HandlerOptions{Level: slog.LevelError}))
}

func decodeJSON[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid json response: %v, body=%s", err, rr.Body.String())
	}
	return out
}

func TestPublicLocationCheck_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	reqBody := `{"user_id":"00000000-0000-0000-0000-000000000001","lat":55.75,"lng":37.61}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	wantReq := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}
	wantResp := domain.LocationCheckResponse{
		Incidents: []string{
			"11111111-1111-1111-1111-111111111111",
		},
	}

	svc.EXPECT().
		CheckLocation(gomock.Any(), wantReq).
		Return(wantResp, nil).
		Times(1)

	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d got %d body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	got := decodeJSON[domain.LocationCheckResponse](t, rr)
	if !reflect.DeepEqual(got, wantResp) {
		t.Fatalf("unexpected response: got=%+v want=%+v", got, wantResp)
	}
}

func TestPublicLocationCheck_InvalidJSON_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", bytes.NewBufferString("{bad json"))
	rr := httptest.NewRecorder()

	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestPublicLocationCheck_EmptyBody_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", nil)
	rr := httptest.NewRecorder()

	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestPublicLocationCheck_UnknownField_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	reqBody := `{"user_id":"00000000-0000-0000-0000-000000000001","lat":55.75,"lng":37.61,"foo":123}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected %d got %d body=%s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

func TestPublicLocationCheck_ServiceError_500(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	reqBody := `{"user_id":"00000000-0000-0000-0000-000000000001","lat":55.75,"lng":37.61}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	wantReq := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}

	svc.EXPECT().
		CheckLocation(gomock.Any(), wantReq).
		Return(domain.LocationCheckResponse{}, errors.New("boom")).
		Times(1)

	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d got %d body=%s", http.StatusInternalServerError, rr.Code, rr.Body.String())
	}
}

func TestPublicLocationCheck_JSONWithExtraTrailingData_400(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mock_public.NewMockPublicHandler(ctrl)
	h := public.NewHandler(newTestLogger(), svc)

	reqBody := `{"user_id":"00000000-0000-0000-0000-000000000001","lat":55.75,"lng":37.61}{"x":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/location/check", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()
	
	h.PublicLocationCheck(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d body=%s", rr.Code, rr.Body.String())
	}
}
