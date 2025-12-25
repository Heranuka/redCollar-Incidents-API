//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"redCollar/internal/domain"
	"redCollar/pkg/e"
)

var (
	testPool *pgxpool.Pool
	tc       testcontainers.Container
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	user := "postgres"
	pass := "postgres"
	db := "postgres"

	req := testcontainers.ContainerRequest{
		Image:        "postgis/postgis:16-3.4-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": pass,
			"POSTGRES_DB":       db,
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(90 * time.Second),
	}

	var err error
	tc, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Println("cannot start container:", err)
		os.Exit(1)
	}

	host, _ := tc.Host(ctx)
	mappedPort, _ := tc.MappedPort(ctx, "5432/tcp")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, mappedPort.Port(), db)

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Println("pgxpool.New:", err)
		_ = tc.Terminate(ctx)
		os.Exit(1)
	}

	if err := testPool.Ping(ctx); err != nil {
		fmt.Println("pool.Ping:", err)
		testPool.Close()
		_ = tc.Terminate(ctx)
		os.Exit(1)
	}

	if err := setupSchema(ctx, testPool); err != nil {
		fmt.Println("setupSchema:", err)
		testPool.Close()
		_ = tc.Terminate(ctx)
		os.Exit(1)
	}

	code := m.Run()

	testPool.Close()
	_ = tc.Terminate(ctx)
	os.Exit(code)
}

func setupSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS postgis;

		CREATE TABLE IF NOT EXISTS incidents (
			id uuid PRIMARY KEY,
			geo_point geography(Point, 4326) NOT NULL,
			radius_km double precision NOT NULL,
			status text NOT NULL,
			created_at timestamptz NOT NULL
		);
	`)
	return err
}

func truncateIncidents(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), `TRUNCATE TABLE incidents`)
	if err != nil {
		t.Fatalf("truncate incidents: %v", err)
	}
}

func TestIncidentAdmin_Create_SetsDefaults(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	inc := &domain.Incident{
		Lat:      55.75,
		Lng:      37.61,
		RadiusKM: 1,
	}

	err := repo.Create(context.Background(), inc)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if inc.ID == uuid.Nil {
		t.Fatalf("expected ID set")
	}
	if inc.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt set")
	}
	if inc.Status != domain.IncidentActive {
		t.Fatalf("expected status=%s got=%s", domain.IncidentActive, inc.Status)
	}

	got, err := repo.Get(context.Background(), inc.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Lat != inc.Lat || got.Lng != inc.Lng {
		t.Fatalf("lat/lng mismatch got=(%v,%v) want=(%v,%v)", got.Lat, got.Lng, inc.Lat, inc.Lng)
	}
	if got.RadiusKM != inc.RadiusKM {
		t.Fatalf("radius mismatch got=%v want=%v", got.RadiusKM, inc.RadiusKM)
	}
	if got.Status != domain.IncidentActive {
		t.Fatalf("status mismatch got=%v", got.Status)
	}
}

func TestIncidentAdmin_List_OnlyActive_WithPagination(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	var ids []uuid.UUID
	for i := 0; i < 3; i++ {
		inc := &domain.Incident{
			Lat:      10 + float64(i),
			Lng:      20 + float64(i),
			RadiusKM: 1,
			Status:   domain.IncidentActive,

			CreatedAt: time.Date(2025, 1, 1, 0, 0, i, 0, time.UTC),
		}
		if err := repo.Create(context.Background(), inc); err != nil {
			t.Fatalf("Create: %v", err)
		}
		ids = append(ids, inc.ID)
	}

	inactive := &domain.Incident{
		Lat:       99,
		Lng:       99,
		RadiusKM:  1,
		Status:    domain.IncidentInactive,
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 99, 0, time.UTC),
	}
	if err := repo.Create(context.Background(), inactive); err != nil {
		t.Fatalf("Create inactive: %v", err)
	}

	list1, total, err := repo.List(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total=3 got=%d", total)
	}
	if len(list1) != 2 {
		t.Fatalf("expected len=2 got=%d", len(list1))
	}

	if list1[0].CreatedAt.Before(list1[1].CreatedAt) {
		t.Fatalf("expected DESC order by created_at")
	}

	list2, total2, err := repo.List(context.Background(), 2, 2)
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if total2 != 3 {
		t.Fatalf("expected total=3 got=%d", total2)
	}
	if len(list2) != 1 {
		t.Fatalf("expected len=1 got=%d", len(list2))
	}
}

func TestIncidentAdmin_Update_OK(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	inc := &domain.Incident{
		Lat:       10,
		Lng:       20,
		RadiusKM:  1,
		Status:    domain.IncidentActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), inc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	inc.Lat = 11
	inc.Lng = 21
	inc.RadiusKM = 2
	inc.Status = domain.IncidentInactive

	if err := repo.Update(context.Background(), inc); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.Get(context.Background(), inc.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Lat != 11 || got.Lng != 21 || got.RadiusKM != 2 || got.Status != domain.IncidentInactive {
		t.Fatalf("unexpected updated row: %+v", got)
	}
}

func TestIncidentAdmin_Update_NotFound(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	inc := &domain.Incident{
		ID:       uuid.New(),
		Lat:      10,
		Lng:      20,
		RadiusKM: 1,
		Status:   domain.IncidentActive,
	}

	err := repo.Update(context.Background(), inc)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, e.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestIncidentAdmin_Delete_SoftDelete(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	inc := &domain.Incident{
		Lat:       10,
		Lng:       20,
		RadiusKM:  1,
		Status:    domain.IncidentActive,
		CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(context.Background(), inc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(context.Background(), inc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	list, total, err := repo.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("expected empty list after delete, total=%d len=%d", total, len(list))
	}

	err = repo.Delete(context.Background(), inc.ID)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, e.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestIncidentAdmin_Create_LngLatOrder_RoundTrip(t *testing.T) {

	truncateIncidents(t)

	repo := NewIncidentAdmin(testPool)

	inc := &domain.Incident{
		Lat:      49.281441,
		Lng:      -123.055913,
		RadiusKM: 1,
	}
	if err := repo.Create(context.Background(), inc); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(context.Background(), inc.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.Lat != inc.Lat || got.Lng != inc.Lng {
		t.Fatalf("expected round-trip lat/lng equal; got=(%v,%v) want=(%v,%v)", got.Lat, got.Lng, inc.Lat, inc.Lng)
	}
}
