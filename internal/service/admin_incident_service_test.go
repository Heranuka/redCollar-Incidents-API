package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"

	"redCollar/internal/domain"
	"redCollar/internal/service"

	mock_service "redCollar/internal/service/mocks" // <-- поправь импорт на свой путь
)

// --- helpers ---

func f64ptr(v float64) *float64                                { return &v }
func statusPtr(s domain.IncidentStatus) *domain.IncidentStatus { return &s }

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	return uuid.New()
}

func mustTime(t *testing.T) time.Time {
	t.Helper()
	return time.Date(2025, 12, 23, 12, 0, 0, 0, time.UTC)
}
func assertIncidentForServiceCreate(t *testing.T, inc *domain.Incident) {
	t.Helper()
	if inc == nil {
		t.Fatalf("incident is nil")
	}
	// Оставь это только если service реально генерит ID.
	if inc.ID == uuid.Nil {
		t.Fatalf("incident.ID is nil")
	}
	// CreatedAt/Status НЕ проверяем: их может выставлять repo.
}

func assertIncidentAfterRepoDefaults(t *testing.T, inc *domain.Incident) {
	t.Helper()
	if inc == nil {
		t.Fatalf("incident is nil")
	}
	if inc.ID == uuid.Nil {
		t.Fatalf("incident.ID is nil")
	}
	if inc.CreatedAt.IsZero() {
		t.Fatalf("incident.CreatedAt is zero")
	}
	if inc.Status == "" {
		t.Fatalf("incident.Status is empty")
	}
}

// --- Create ---

func TestAdminIncidentService_Create_OK_Defaults(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)
	cache := mock_service.NewMockIncidentCacheService(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()
	var got *domain.Incident
	repo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
			got = inc
			return nil
		}).
		Times(1)

	svc := service.NewAdminIncidentService(repo, cache)

	req := domain.CreateIncidentRequest{
		Lat:      55.75,
		Lng:      37.61,
		RadiusKM: 1,
	}

	id, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if id == uuid.Nil {
		t.Fatalf("expected non-nil id")
	}

	assertIncidentForServiceCreate(t, got)

	if got.Lat != req.Lat || got.Lng != req.Lng || got.RadiusKM != req.RadiusKM {
		t.Fatalf("incident fields mismatch: got=%+v req=%+v", got, req)
	}

	// Ожидаем дефолтный статус (если у тебя так в сервисе)
	if got.Status != domain.IncidentActive {
		t.Fatalf("expected default status=%q, got=%q", domain.IncidentActive, got.Status)
	}
}

func TestAdminIncidentService_Create_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)

	wantErr := errors.New("db down")
	repo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(wantErr).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	_, err := svc.Create(context.Background(), domain.CreateIncidentRequest{
		Lat: 10, Lng: 10, RadiusKM: 1,
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestAdminIncidentService_Create_Boundaries(t *testing.T) {
	t.Parallel()

	type tc struct {
		name string
		req  domain.CreateIncidentRequest
	}

	cases := []tc{
		{"lat_min_lng_min_radius_min", domain.CreateIncidentRequest{Lat: -90, Lng: -180, RadiusKM: 0.1}},
		{"lat_max_lng_max_radius_max", domain.CreateIncidentRequest{Lat: 90, Lng: 180, RadiusKM: 100}},
		{"middle_values", domain.CreateIncidentRequest{Lat: 0, Lng: 0, RadiusKM: 1}},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock_service.NewMockIncidentRepository(ctrl)
			repo.EXPECT().
				ListActive(gomock.Any()).
				Return([]*domain.Incident{}, nil). // можешь подставить нужный список
				Times(1)

			cache := mock_service.NewMockIncidentCacheService(ctrl)
			cache.EXPECT().
				SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil).
				AnyTimes()

			repo.EXPECT().
				Create(gomock.Any(), gomock.Any()).
				Return(nil).
				Times(1)

			svc := service.NewAdminIncidentService(repo, cache)

			id, err := svc.Create(context.Background(), c.req)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if id == uuid.Nil {
				t.Fatalf("expected non-nil id")
			}
		})
	}
}

// --- Get ---

func TestAdminIncidentService_Get_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)

	id := mustUUID(t)
	want := &domain.Incident{
		ID:        id,
		Lat:       1,
		Lng:       2,
		RadiusKM:  3,
		Status:    domain.IncidentActive,
		CreatedAt: mustTime(t),
	}

	repo.EXPECT().
		Get(gomock.Any(), id).
		Return(want, nil).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	got, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.ID != id {
		t.Fatalf("unexpected incident: %+v", got)
	}
}

func TestAdminIncidentService_Get_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)

	id := mustUUID(t)
	repo.EXPECT().
		Get(gomock.Any(), id).
		Return(nil, errors.New("not found")).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	_, err := svc.Get(context.Background(), id)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// --- List ---

func TestAdminIncidentService_List_OK_Empty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)

	repo.EXPECT().
		List(gomock.Any(), 1, 20).
		Return([]*domain.Incident{}, int64(0), nil).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	list, total, err := svc.List(context.Background(), 1, 20)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if total != 0 {
		t.Fatalf("expected total=0 got=%d", total)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list got=%d", len(list))
	}
}

func TestAdminIncidentService_List_OK_NonEmpty(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)

	wantList := []*domain.Incident{
		{ID: mustUUID(t), Status: domain.IncidentActive, CreatedAt: mustTime(t)},
		{ID: mustUUID(t), Status: domain.IncidentActive, CreatedAt: mustTime(t)},
	}
	var wantTotal int64 = 2

	repo.EXPECT().
		List(gomock.Any(), 2, 10).
		Return(wantList, wantTotal, nil).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	list, total, err := svc.List(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if total != wantTotal {
		t.Fatalf("expected total=%d got=%d", wantTotal, total)
	}
	if len(list) != len(wantList) {
		t.Fatalf("expected len=%d got=%d", len(wantList), len(list))
	}
}

func TestAdminIncidentService_List_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)

	repo.EXPECT().
		List(gomock.Any(), 1, 20).
		Return(nil, int64(0), errors.New("db error")).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	_, _, err := svc.List(context.Background(), 1, 20)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// --- Update (PATCH semantics) ---

func TestAdminIncidentService_Update_OK_AllFields(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{
		ID:        id,
		Lat:       10,
		Lng:       20,
		RadiusKM:  1,
		Status:    domain.IncidentActive,
		CreatedAt: mustTime(t),
	}

	req := domain.UpdateIncidentRequest{
		Lat:      f64ptr(55.76),
		Lng:      f64ptr(37.62),
		RadiusKM: f64ptr(2),
		Status:   statusPtr(domain.IncidentInactive),
	}

	var updated *domain.Incident

	// Порядок вызовов: Get -> Update. [web:564]
	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
				updated = inc
				return nil
			}).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)

	if err := svc.Update(context.Background(), id, req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if updated == nil {
		t.Fatalf("expected updated incident passed to repo.Update")
	}

	// Проверяем, что patch применился
	if updated.ID != id {
		t.Fatalf("expected ID=%s got=%s", id, updated.ID)
	}
	if updated.CreatedAt != existing.CreatedAt {
		t.Fatalf("CreatedAt must not change")
	}
	if updated.Lat != *req.Lat || updated.Lng != *req.Lng || updated.RadiusKM != *req.RadiusKM || updated.Status != *req.Status {
		t.Fatalf("patch mismatch, updated=%+v req=%+v", updated, req)
	}
}

func TestAdminIncidentService_Update_OK_OnlyLat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{
		ID:        id,
		Lat:       10,
		Lng:       20,
		RadiusKM:  1,
		Status:    domain.IncidentActive,
		CreatedAt: mustTime(t),
	}

	req := domain.UpdateIncidentRequest{
		Lat: f64ptr(-12.34),
	}

	var updated *domain.Incident
	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inc *domain.Incident) error { updated = inc; return nil }).
			Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)

	if err := svc.Update(context.Background(), id, req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if updated.Lat != *req.Lat {
		t.Fatalf("expected lat=%v got=%v", *req.Lat, updated.Lat)
	}
	// Остальные поля должны остаться как были
	if updated.Lng != existing.Lng || updated.RadiusKM != existing.RadiusKM || updated.Status != existing.Status {
		t.Fatalf("unexpected changes: updated=%+v existing=%+v", updated, existing)
	}
}

func TestAdminIncidentService_Update_OK_OnlyLng(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{ID: id, Lat: 10, Lng: 20, RadiusKM: 1, Status: domain.IncidentActive, CreatedAt: mustTime(t)}
	req := domain.UpdateIncidentRequest{Lng: f64ptr(179.999)}

	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
			if inc.Lng != *req.Lng {
				t.Fatalf("expected lng=%v got=%v", *req.Lng, inc.Lng)
			}
			if inc.Lat != existing.Lat || inc.RadiusKM != existing.RadiusKM || inc.Status != existing.Status {
				t.Fatalf("unexpected changes")
			}
			return nil
		}).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)
	if err := svc.Update(context.Background(), id, req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAdminIncidentService_Update_OK_OnlyRadius(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{ID: id, Lat: 10, Lng: 20, RadiusKM: 1, Status: domain.IncidentActive, CreatedAt: mustTime(t)}
	req := domain.UpdateIncidentRequest{RadiusKM: f64ptr(100)}

	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
			if inc.RadiusKM != *req.RadiusKM {
				t.Fatalf("expected radius=%v got=%v", *req.RadiusKM, inc.RadiusKM)
			}
			if inc.Lat != existing.Lat || inc.Lng != existing.Lng || inc.Status != existing.Status {
				t.Fatalf("unexpected changes")
			}
			return nil
		}).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)
	if err := svc.Update(context.Background(), id, req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAdminIncidentService_Update_OK_OnlyStatus(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{ID: id, Lat: 10, Lng: 20, RadiusKM: 1, Status: domain.IncidentActive, CreatedAt: mustTime(t)}
	req := domain.UpdateIncidentRequest{Status: statusPtr(domain.IncidentInactive)}

	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
			if inc.Status != *req.Status {
				t.Fatalf("expected status=%v got=%v", *req.Status, inc.Status)
			}
			if inc.Lat != existing.Lat || inc.Lng != existing.Lng || inc.RadiusKM != existing.RadiusKM {
				t.Fatalf("unexpected changes")
			}
			return nil
		}).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)
	if err := svc.Update(context.Background(), id, req); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAdminIncidentService_Update_GetError_NoUpdateCall(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)

	repo.EXPECT().
		Get(gomock.Any(), id).
		Return(nil, errors.New("not found")).
		Times(1)

	// Важно: repo.Update НЕ ожидаем вообще
	svc := service.NewAdminIncidentService(repo, cache)

	err := svc.Update(context.Background(), id, domain.UpdateIncidentRequest{
		Lat: f64ptr(1),
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestAdminIncidentService_Update_UpdateError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	id := mustUUID(t)
	existing := &domain.Incident{ID: id, Lat: 10, Lng: 20, RadiusKM: 1, Status: domain.IncidentActive, CreatedAt: mustTime(t)}

	wantErr := errors.New("db update failed")
	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(wantErr).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, nil)

	err := svc.Update(context.Background(), id, domain.UpdateIncidentRequest{
		RadiusKM: f64ptr(2),
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// Доп. кейс: все поля nil (пустой patch).
// Если в твоей реализации ты считаешь это ошибкой или no-op — скажи, подстроим ожидания.
func TestAdminIncidentService_Update_EmptyPatch(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)
	existing := &domain.Incident{ID: id, Lat: 10, Lng: 20, RadiusKM: 1, Status: domain.IncidentActive, CreatedAt: mustTime(t)}

	gomock.InOrder(
		repo.EXPECT().Get(gomock.Any(), id).Return(existing, nil).Times(1),
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, inc *domain.Incident) error {
			// Ничего не менялось — обновляем тем же объектом
			if inc.Lat != existing.Lat || inc.Lng != existing.Lng || inc.RadiusKM != existing.RadiusKM || inc.Status != existing.Status {
				t.Fatalf("expected no changes, got=%+v", inc)
			}
			return nil
		}).Times(1),
	)

	svc := service.NewAdminIncidentService(repo, cache)

	if err := svc.Update(context.Background(), id, domain.UpdateIncidentRequest{}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

// --- Delete ---

func TestAdminIncidentService_Delete_OK(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_service.NewMockIncidentRepository(ctrl)
	repo.EXPECT().
		ListActive(gomock.Any()).
		Return([]*domain.Incident{}, nil). // можешь подставить нужный список
		Times(1)

	cache := mock_service.NewMockIncidentCacheService(ctrl)
	cache.EXPECT().
		SetActive(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	id := mustUUID(t)

	repo.EXPECT().
		Delete(gomock.Any(), id).
		Return(nil).
		Times(1)

	svc := service.NewAdminIncidentService(repo, cache)

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAdminIncidentService_Delete_RepoError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mock_service.NewMockIncidentRepository(ctrl)

	id := mustUUID(t)

	repo.EXPECT().
		Delete(gomock.Any(), id).
		Return(errors.New("db error")).
		Times(1)

	svc := service.NewAdminIncidentService(repo, nil)

	if err := svc.Delete(context.Background(), id); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
