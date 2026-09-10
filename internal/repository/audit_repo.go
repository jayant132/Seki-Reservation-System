package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID         int64      `json:"id"`
	EntityType string     `json:"entity_type"`
	EntityID   uuid.UUID  `json:"entity_id"`
	ActorID    *uuid.UUID `json:"actor_id,omitempty"`
	Action     string     `json:"action"`
	PrevHash   string     `json:"prev_hash"`
	EntryHash  string     `json:"entry_hash"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) ListForEntity(ctx context.Context, entityID uuid.UUID) ([]AuditEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, entity_type, entity_id, actor_id, action, prev_hash, entry_hash, created_at
		FROM audit_log WHERE entity_id = $1 ORDER BY id ASC
	`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.EntityType, &e.EntityID, &e.ActorID, &e.Action, &e.PrevHash, &e.EntryHash, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// VerifyChain recomputes every entry's hash from its stored payload and
// compares it against the stored entry_hash, and checks that each entry's
// prev_hash matches the previous row's entry_hash. Any mismatch means a
// row was altered or deleted after the fact.
func (r *AuditRepository) VerifyChain(ctx context.Context) (bool, int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, entity_type, entity_id, action, payload, prev_hash, entry_hash
		FROM audit_log ORDER BY id ASC
	`)
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()

	expectedPrev := "genesis"
	for rows.Next() {
		var (
			id                                      int64
			entityType, action, prevHash, entryHash string
			entityID                                uuid.UUID
			payload                                 []byte
		)
		if err := rows.Scan(&id, &entityType, &entityID, &action, &payload, &prevHash, &entryHash); err != nil {
			return false, id, err
		}

		if prevHash != expectedPrev {
			return false, id, nil
		}
		recomputed := computeEntryHash(prevHash, entityType, entityID, action, payload)
		if recomputed != entryHash {
			return false, id, nil
		}
		expectedPrev = entryHash
	}

	return true, 0, rows.Err()
}
