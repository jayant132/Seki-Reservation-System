package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jayant132/seki/internal/domain"
)

type ResourceRepository struct {
	pool *pgxpool.Pool
}

func NewResourceRepository(pool *pgxpool.Pool) *ResourceRepository {
	return &ResourceRepository{pool: pool}
}

func (r *ResourceRepository) Create(ctx context.Context, name, description string, capacity int) (*domain.Resource, error) {
	res := &domain.Resource{}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO resources (name, description, capacity)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, capacity, created_at
	`, name, description, capacity).Scan(&res.ID, &res.Name, &res.Description, &res.Capacity, &res.CreatedAt)
	return res, err
}

func (r *ResourceRepository) List(ctx context.Context) ([]domain.Resource, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, capacity, created_at
		FROM resources ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Resource
	for rows.Next() {
		var res domain.Resource
		if err := rows.Scan(&res.ID, &res.Name, &res.Description, &res.Capacity, &res.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, rows.Err()
}

func (r *ResourceRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Resource, error) {
	res := &domain.Resource{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, capacity, created_at
		FROM resources WHERE id = $1
	`, id).Scan(&res.ID, &res.Name, &res.Description, &res.Capacity, &res.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return res, err
}
