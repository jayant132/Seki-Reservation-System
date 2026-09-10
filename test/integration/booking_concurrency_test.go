

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
	"github.com/jayant132/seki/internal/domain"
	"github.com/jayant132/seki/internal/repository"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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
		WaitingFor: wait.ForListeningPort("5432/tcp").
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Make sure the container is always cleaned up if anything below fails.
	cleanupContainer := func() {
		_ = container.Terminate(ctx)
	}

	host, err := container.Host(ctx)
	if err != nil {
		cleanupContainer()
		t.Fatalf("failed to get postgres host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		cleanupContainer()
		t.Fatalf("failed to get postgres mapped port: %v", err)
	}

	dsn := fmt.Sprintf(
		"postgres://seki:seki@%s:%s/seki?sslmode=disable",
		host,
		port.Port(),
	)

	// Give PostgreSQL a moment to finish initialization after the port
	// becomes available. The migration step below also retries, so this
	// protects CI from transient connection-reset errors.
	var migrateErr error

	migrationsPath, err := filepath.Abs("../../migrations")
	if err != nil {
		cleanupContainer()
		t.Fatalf("failed to resolve migrations path: %v", err)
	}

	for attempt := 1; attempt <= 10; attempt++ {
		cmd := exec.Command(
			"migrate",
			"-path",
			migrationsPath,
			"-database",
			dsn,
			"up",
		)

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		migrateErr = cmd.Run()

		if migrateErr == nil {
			break
		}

		if attempt < 10 {
			t.Logf(
				"migration attempt %d/10 failed: %v; retrying in 2s",
				attempt,
				migrateErr,
			)
			time.Sleep(2 * time.Second)
		}
	}

	if migrateErr != nil {
		cleanupContainer()
		t.Fatalf(
			"failed to run migrations after 10 attempts (is the `migrate` CLI installed?): %v",
			migrateErr,
		)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		cleanupContainer()
		t.Fatalf("failed to create database pool: %v", err)
	}

	// Explicitly verify that the pool can actually communicate with
	// PostgreSQL before returning it to the tests.
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		cleanupContainer()
		t.Fatalf("failed to ping postgres after migrations: %v", err)
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

	resource, err := resourceRepo.Create(
		ctx,
		"Concurrency Test Room",
		"",
		1,
	)
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	const numRacers = 25

	users := make([]uuid.UUID, numRacers)

	for i := 0; i < numRacers; i++ {
		u, err := userRepo.Create(
			ctx,
			fmt.Sprintf("racer%d@test.dev", i),
			"hash",
			domain.RoleCustomer,
		)
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

			_, err := bookingRepo.CreateBooking(
				ctx,
				userID,
				domain.CreateBookingInput{
					ResourceID: resource.ID,
					StartTime:  start,
					EndTime:    end,
				},
				"",
			)

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
		t.Errorf(
			"expected exactly 1 successful booking, got %d",
			successes,
		)
	}

	if conflicts != numRacers-1 {
		t.Errorf(
			"expected %d conflicts, got %d",
			numRacers-1,
			conflicts,
		)
	}

	if unexpected != 0 {
		t.Errorf(
			"expected 0 unexpected errors, got %d",
			unexpected,
		)
	}

	t.Logf(
		"Result: %d succeeded, %d correctly rejected as conflicts, %d unexpected errors",
		successes,
		conflicts,
		unexpected,
	)
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

	resource, err := resourceRepo.Create(
		ctx,
		"Idempotency Test Room",
		"",
		1,
	)
	if err != nil {
		t.Fatalf("failed to create resource: %v", err)
	}

	user, err := userRepo.Create(
		ctx,
		"idempotent@test.dev",
		"hash",
		domain.RoleCustomer,
	)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	start := time.Now().Add(48 * time.Hour)

	input := domain.CreateBookingInput{
		ResourceID: resource.ID,
		StartTime:  start,
		EndTime:    start.Add(time.Hour),
	}

	key := "retry-key-123"

	first, err := bookingRepo.CreateBooking(
		ctx,
		user.ID,
		input,
		key,
	)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	second, err := bookingRepo.CreateBooking(
		ctx,
		user.ID,
		input,
		key,
	)
	if err != nil {
		t.Fatalf("retried request failed: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf(
			"expected retried request to return the same booking ID, got %s vs %s",
			first.ID,
			second.ID,
		)
	}
}