package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jayant132/seki/internal/domain"
)

// pgErrExclusionViolation is the SQLSTATE Postgres raises when an INSERT
// would violate an EXCLUDE constraint — i.e. exactly the "two people booked
// the same slot at the same instant" race this whole project exists to
// prevent. We translate it to a domain error the HTTP layer maps to 409.
const pgErrExclusionViolation = "23P01"

type BookingRepository struct {
	pool *pgxpool.Pool
}

func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

// CreateBooking performs the insert, the audit-log entry, and (if an
// idempotency key was supplied) the idempotency record, all inside a
// single transaction. Either everything commits or nothing does — the
// audit log can never drift from the actual booking state, and a retried
// request can never produce two bookings.
//
// Correctness against concurrent requests for the *same* slot doesn't come
// from application-level locking here — it comes from the EXCLUDE
// constraint on the bookings table itself, which Postgres enforces against
// every transaction regardless of how many API instances are running. This
// function's job is just to catch that constraint violation and translate
// it into a clean domain error.
func (r *BookingRepository) CreateBooking(
	ctx context.Context,
	userID uuid.UUID,
	input domain.CreateBookingInput,
	idempotencyKey string,
) (*domain.Booking, error) {

	var result *domain.Booking

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Idempotency check happens inside the same transaction as the
		// insert so a concurrent retry with the same key can't slip
		// through between the check and the write.
		if idempotencyKey != "" {
			cached, err := checkIdempotency(ctx, tx, idempotencyKey, userID, input)
			if err != nil {
				return err
			}
			if cached != nil {
				result = cached
				return nil
			}
		}

		b := &domain.Booking{}
		err := tx.QueryRow(ctx, `
			INSERT INTO bookings (resource_id, user_id, start_time, end_time, status, notes)
			VALUES ($1, $2, $3, $4, 'confirmed', $5)
			RETURNING id, resource_id, user_id, start_time, end_time, status, notes, created_at, updated_at
		`, input.ResourceID, userID, input.StartTime, input.EndTime, input.Notes).Scan(
			&b.ID, &b.ResourceID, &b.UserID, &b.StartTime, &b.EndTime,
			&b.Status, &b.Notes, &b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgErrExclusionViolation {
				return domain.ErrSlotUnavailable
			}
			return err
		}

		if err := appendAuditEntry(ctx, tx, "booking", b.ID, &userID, "created", b); err != nil {
			return err
		}

		if idempotencyKey != "" {
			if err := storeIdempotency(ctx, tx, idempotencyKey, userID, input, 201, b); err != nil {
				return err
			}
		}

		result = b
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *BookingRepository) CancelBooking(ctx context.Context, id, userID uuid.UUID, isAdmin bool) (*domain.Booking, error) {
	var result *domain.Booking

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var ownerID uuid.UUID
		var status domain.BookingStatus
		err := tx.QueryRow(ctx, `SELECT user_id, status FROM bookings WHERE id = $1 FOR UPDATE`, id).
			Scan(&ownerID, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		if err != nil {
			return err
		}
		if ownerID != userID && !isAdmin {
			return domain.ErrForbidden
		}

		b := &domain.Booking{}
		err = tx.QueryRow(ctx, `
			UPDATE bookings SET status = 'cancelled', updated_at = now()
			WHERE id = $1
			RETURNING id, resource_id, user_id, start_time, end_time, status, notes, created_at, updated_at
		`, id).Scan(&b.ID, &b.ResourceID, &b.UserID, &b.StartTime, &b.EndTime,
			&b.Status, &b.Notes, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return err
		}

		if err := appendAuditEntry(ctx, tx, "booking", b.ID, &userID, "cancelled", b); err != nil {
			return err
		}

		result = b
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *BookingRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Booking, error) {
	return r.queryBookings(ctx, `
		SELECT id, resource_id, user_id, start_time, end_time, status, notes, created_at, updated_at
		FROM bookings WHERE user_id = $1 ORDER BY start_time DESC
	`, userID)
}

func (r *BookingRepository) ListForResource(ctx context.Context, resourceID uuid.UUID, from, to time.Time) ([]domain.Booking, error) {
	return r.queryBookings(ctx, `
		SELECT id, resource_id, user_id, start_time, end_time, status, notes, created_at, updated_at
		FROM bookings
		WHERE resource_id = $1 AND status = 'confirmed' AND during && tstzrange($2, $3, '[)')
		ORDER BY start_time ASC
	`, resourceID, from, to)
}

func (r *BookingRepository) queryBookings(ctx context.Context, query string, args ...interface{}) ([]domain.Booking, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Booking
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.ResourceID, &b.UserID, &b.StartTime, &b.EndTime,
			&b.Status, &b.Notes, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// --- idempotency helpers -----------------------------------------------

func checkIdempotency(ctx context.Context, tx pgx.Tx, key string, userID uuid.UUID, input domain.CreateBookingInput) (*domain.Booking, error) {
	reqHash := hashRequest(input)

	var storedHash string
	var body []byte
	err := tx.QueryRow(ctx, `
		SELECT request_hash, response_body FROM idempotency_keys WHERE key = $1 AND user_id = $2
	`, key, userID).Scan(&storedHash, &body)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // no cached response yet, proceed normally
	}
	if err != nil {
		return nil, err
	}
	if storedHash != reqHash {
		return nil, domain.ErrIdempotencyReuse
	}

	var b domain.Booking
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func storeIdempotency(ctx context.Context, tx pgx.Tx, key string, userID uuid.UUID, input domain.CreateBookingInput, code int, body *domain.Booking) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO idempotency_keys (key, user_id, request_hash, response_code, response_body)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (key) DO NOTHING
	`, key, userID, hashRequest(input), code, payload)
	return err
}

func hashRequest(input domain.CreateBookingInput) string {
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// --- audit chain helpers -------------------------------------------------

// appendAuditEntry writes a hash-chained audit row: each entry's hash is a
// function of its own payload plus the previous entry's hash, so altering
// any historical row breaks every hash after it. This mirrors the pattern
// used in the Ringi approval-workflow project, applied here to bookings.
func appendAuditEntry(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID, actorID *uuid.UUID, action string, payload interface{}) error {
	var prevHash string
	err := tx.QueryRow(ctx, `SELECT entry_hash FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&prevHash)
	if errors.Is(err, pgx.ErrNoRows) {
		prevHash = "genesis"
	} else if err != nil {
		return err
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	entryHash := computeEntryHash(prevHash, entityType, entityID, action, payloadJSON)

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_log (entity_type, entity_id, actor_id, action, payload, prev_hash, entry_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entityType, entityID, actorID, action, payloadJSON, prevHash, entryHash)
	return err
}

func computeEntryHash(prevHash, entityType string, entityID uuid.UUID, action string, payload []byte) string {
	h := sha256.New()
	h.Write([]byte(prevHash))
	h.Write([]byte(entityType))
	h.Write([]byte(entityID.String()))
	h.Write([]byte(action))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
