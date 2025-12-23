package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"redCollar/internal/domain"
	"redCollar/internal/service"

	mock_service "redCollar/internal/service/mocks" // <-- поправь на свой путь
)

func TestService_CheckLocation_Delegates_OK_EmptyIncidents(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish() // можно оставить явно для совместимости [web:564]

	publicSvc := mock_service.NewMockPublicIncidentService(ctrl)

	req := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}

	want := domain.LocationCheckResponse{Incidents: []string{}}

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req).
		Return(want, nil).
		Times(1)

	svc := service.NewService(nil, publicSvc, nil)

	got, err := svc.CheckLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: got=%+v want=%+v", got, want)
	}
}

func TestService_CheckLocation_Delegates_OK_WithIncidents(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publicSvc := mock_service.NewMockPublicIncidentService(ctrl)

	req := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}

	want := domain.LocationCheckResponse{
		Incidents: []string{
			"11111111-1111-1111-1111-111111111111",
			"22222222-2222-2222-2222-222222222222",
		},
	}

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req).
		Return(want, nil).
		Times(1)

	svc := service.NewService(nil, publicSvc, nil)

	got, err := svc.CheckLocation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected response: got=%+v want=%+v", got, want)
	}
}

func TestService_CheckLocation_Delegates_ErrorPropagated(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publicSvc := mock_service.NewMockPublicIncidentService(ctrl)

	req := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}

	wantErr := errors.New("boom")

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req).
		Return(domain.LocationCheckResponse{}, wantErr).
		Times(1)

	svc := service.NewService(nil, publicSvc, nil)

	_, err := svc.CheckLocation(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected err=%v got=%v", wantErr, err)
	}
}

func TestService_CheckLocation_PassesContextValue(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publicSvc := mock_service.NewMockPublicIncidentService(ctrl)

	req := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    55.75,
		Lng:    37.61,
	}

	ctx := context.WithValue(context.Background(), ctxKey("trace_id"), "trace-123")

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req).
		DoAndReturn(func(ctx context.Context, gotReq domain.LocationCheckRequest) (domain.LocationCheckResponse, error) {
			if ctx.Value(ctxKey("trace_id")) != "trace-123" {
				t.Fatalf("context value not passed")
			}
			if !reflect.DeepEqual(gotReq, req) {
				t.Fatalf("request mismatch: got=%+v want=%+v", gotReq, req)
			}
			return domain.LocationCheckResponse{Incidents: nil}, nil
		}).
		Times(1) // контролируем, что прокси вызвал ровно один раз [web:564][web:583]

	svc := service.NewService(nil, publicSvc, nil)

	_, err := svc.CheckLocation(ctx, req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestService_CheckLocation_MultipleCalls(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	publicSvc := mock_service.NewMockPublicIncidentService(ctrl)

	req1 := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000001",
		Lat:    1,
		Lng:    2,
	}
	req2 := domain.LocationCheckRequest{
		UserID: "00000000-0000-0000-0000-000000000002",
		Lat:    3,
		Lng:    4,
	}

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req1).
		Return(domain.LocationCheckResponse{Incidents: []string{"a"}}, nil).
		Times(1)

	publicSvc.EXPECT().
		CheckLocation(gomock.Any(), req2).
		Return(domain.LocationCheckResponse{Incidents: []string{"b"}}, nil).
		Times(1)

	svc := service.NewService(nil, publicSvc, nil)

	r1, err := svc.CheckLocation(context.Background(), req1)
	if err != nil || len(r1.Incidents) != 1 || r1.Incidents[0] != "a" {
		t.Fatalf("unexpected r1=%+v err=%v", r1, err)
	}

	r2, err := svc.CheckLocation(context.Background(), req2)
	if err != nil || len(r2.Incidents) != 1 || r2.Incidents[0] != "b" {
		t.Fatalf("unexpected r2=%+v err=%v", r2, err)
	}
}
