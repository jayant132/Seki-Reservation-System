// Package integration exercises the booking repository against a real,
// ephemeral Postgres instance (via testcontainers-go) rather than a mock —
// the entire point of this test is to prove the database-level EXCLUDE
// constraint actually prevents double bookings, which a mock could never
// demonstrate.
//
// Run with:  go test ./test/integration/... -v
// Requires Docker to be running locally.
package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/jayant132/seki/internal/domain"
	"github.com/jayant132/seki/internal/repository"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "seki",
			"POSTGRES_PASSWORD": "seki",
			"POSTGRES_DB":       "seki",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://seki:seki@%s:%s/seki?sslmode=disable", host, port.Port())

	// Apply migrations using the `migrate` CLI against the ephemeral container.
	migrationsPath, _ := filepath.Abs("../../migrations")
	cmd := exec.Command("migrate", "-path", migrationsPath, "-database", dsn, "up")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to run migrations (is the `migrate` CLI installed? https://github.com/golang-migrate/migrate): %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}

// TestConcurrentBookings_OnlyOneWins fires 25 goroutines at the exact same
// resource + time slot simultaneously and asserts exactly one booking is
// created and every other attempt receives domain.ErrSlotUnavailable. This
// is the load-bearing test of the whole project.
func TestConcurrentBookings_OnlyOneWins(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := repository.NewUserRepository(pool)
	resourceRepo := repository.NewResourceRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)

	resource, err := resourceRepo.Create(ctx, "Concurrency Test Room", "", 1)
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	const numRacers = 25
	users := make([]uuid.UUID, numRacers)
	for i := 0; i < numRacers; i++ {
		u, err := userRepo.Create(ctx, fmt.Sprintf("racer%d@test.dev", i), "hash", domain.RoleCustomer)
		if err != nil {
			t.Fatalf("failed to create user %d: %v", i, err)
		}
		users[i] = u.ID
	}

	start := time.Now().Add(24 * time.Hour)
	end := start.Add(time.Hour)

	var wg sync.WaitGroup
	var successes int32
	var conflicts int32
	var unexpected int32

	for i := 0; i < numRacers; i++ {
		wg.Add(1)
		go func(userID uuid.UUID) {
			defer wg.Done()
			_, err := bookingRepo.CreateBooking(ctx, userID, domain.CreateBookingInput{
				ResourceID: resource.ID,
				StartTime:  start,
				EndTime:    end,
			}, "")

			switch err {
			case nil:
				atomic.AddInt32(&successes, 1)
			case domain.ErrSlotUnavailable:
				atomic.AddInt32(&conflicts, 1)
			default:
				atomic.AddInt32(&unexpected, 1)
				t.Logf("unexpected error: %v", err)
			}
		}(users[i])
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("expected exactly 1 successful booking, got %d", successes)
	}
	if conflicts != numRacers-1 {
		t.Errorf("expected %d conflicts, got %d", numRacers-1, conflicts)
	}
	if unexpected != 0 {
		t.Errorf("expected 0 unexpected errors, got %d", unexpected)
	}

	t.Logf("Result: %d succeeded, %d correctly rejected as conflicts, %d unexpected errors",
		successes, conflicts, unexpected)
}

// TestIdempotentRetry_ReturnsSameBooking proves that retrying the exact
// same booking request with the same idempotency key returns the original
// booking instead of creating a duplicate or a conflict.
func TestIdempotentRetry_ReturnsSameBooking(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	userRepo := repository.NewUserRepository(pool)
	resourceRepo := repository.NewResourceRepository(pool)
	bookingRepo := repository.NewBookingRepository(pool)

	resource, _ := resourceRepo.Create(ctx, "Idempotency Test Room", "", 1)
	user, _ := userRepo.Create(ctx, "idempotent@test.dev", "hash", domain.RoleCustomer)

	start := time.Now().Add(48 * time.Hour)
	input := domain.CreateBookingInput{
		ResourceID: resource.ID,
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
	}

	key := "retry-key-123"

	first, err := bookingRepo.CreateBooking(ctx, user.ID, input, key)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	second, err := bookingRepo.CreateBooking(ctx, user.ID, input, key)
	if err != nil {
		t.Fatalf("retried request failed: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected retried request to return the same booking ID, got %s vs %s", first.ID, second.ID)
	}
}
