package service_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"redCollar/internal/domain"
	"redCollar/internal/service"

	mock_service "redCollar/internal/service/mocks" // <-- поправь
)

type ctxKey string

func TestService_GetStats_Delegates_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_service.NewMockStatsService(ctrl)

	req := domain.StatsRequest{Minutes: 60}
	want := &domain.IncidentStats{UserCount: 123}

	statsSvc.EXPECT().
		GetStats(gomock.Any(), req).
		Return(want, nil).
		Times(1)

	svc := service.NewService(nil, nil, statsSvc)

	got, err := svc.GetStats(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected stats: got=%+v want=%+v", got, want)
	}
}

func TestService_GetStats_Delegates_ErrorPropagated(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_service.NewMockStatsService(ctrl)

	req := domain.StatsRequest{Minutes: 15}
	wantErr := errors.New("stats failed")

	statsSvc.EXPECT().
		GetStats(gomock.Any(), req).
		Return(nil, wantErr).
		Times(1)

	svc := service.NewService(nil, nil, statsSvc)

	_, err := svc.GetStats(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected err=%v got=%v", wantErr, err)
	}
}

func TestService_GetStats_PassesContextValue(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_service.NewMockStatsService(ctrl)

	req := domain.StatsRequest{Minutes: 1}
	ctx := context.WithValue(context.Background(), ctxKey("trace_id"), "trace-123")

	statsSvc.EXPECT().
		GetStats(gomock.Any(), req).
		DoAndReturn(func(ctx context.Context, gotReq domain.StatsRequest) (*domain.IncidentStats, error) {
			if ctx.Value(ctxKey("trace_id")) != "trace-123" {
				t.Fatalf("context value not passed")
			}
			if gotReq != req {
				t.Fatalf("request mismatch: got=%+v want=%+v", gotReq, req)
			}
			return &domain.IncidentStats{UserCount: 0}, nil
		}).
		Times(1)

	svc := service.NewService(nil, nil, statsSvc)

	_, err := svc.GetStats(ctx, req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestService_GetStats_MultipleCalls_DifferentInputs(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statsSvc := mock_service.NewMockStatsService(ctrl)

	req1 := domain.StatsRequest{Minutes: 10}
	req2 := domain.StatsRequest{Minutes: 120}

	statsSvc.EXPECT().
		GetStats(gomock.Any(), req1).
		Return(&domain.IncidentStats{UserCount: 1}, nil).
		Times(1)

	statsSvc.EXPECT().
		GetStats(gomock.Any(), req2).
		Return(&domain.IncidentStats{UserCount: 2}, nil).
		Times(1)

	svc := service.NewService(nil, nil, statsSvc)

	s1, err := svc.GetStats(context.Background(), req1)
	if err != nil || s1.UserCount != 1 {
		t.Fatalf("unexpected s1=%+v err=%v", s1, err)
	}

	s2, err := svc.GetStats(context.Background(), req2)
	if err != nil || s2.UserCount != 2 {
		t.Fatalf("unexpected s2=%+v err=%v", s2, err)
	}
}
