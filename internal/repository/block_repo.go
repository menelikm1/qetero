package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
)

type BlockRepo struct {
	db *pgxpool.Pool
}

func NewBlockRepo(db *pgxpool.Pool) *BlockRepo {
	return &BlockRepo{db: db}
}

func (r *BlockRepo) Create(ctx context.Context, b *models.ListingBlock) error {
	query := `
		INSERT INTO listing_blocks (id, listing_id, owner_id, start_date, end_date, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	return r.db.QueryRow(ctx, query,
		b.ID, b.ListingID, b.OwnerID, b.StartDate, b.EndDate, b.Note,
	).Scan(&b.CreatedAt)
}

func (r *BlockRepo) GetByListing(ctx context.Context, listingID uuid.UUID) ([]models.ListingBlock, error) {
	query := `
		SELECT id, listing_id, owner_id, start_date, end_date, COALESCE(note, ''), created_at
		FROM listing_blocks
		WHERE listing_id = $1 AND end_date >= CURRENT_DATE
		ORDER BY start_date`
	rows, err := r.db.Query(ctx, query, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []models.ListingBlock
	for rows.Next() {
		var b models.ListingBlock
		if err := rows.Scan(&b.ID, &b.ListingID, &b.OwnerID, &b.StartDate, &b.EndDate, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// HasBlock returns true if a listing has a manual block overlapping the given dates.
func (r *BlockRepo) HasBlock(ctx context.Context, listingID uuid.UUID, start, end time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM listing_blocks
			WHERE listing_id = $1
			  AND NOT (end_date <= $2 OR start_date >= $3)
		)`
	var exists bool
	err := r.db.QueryRow(ctx, query, listingID, start, end).Scan(&exists)
	return exists, err
}

func (r *BlockRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM listing_blocks WHERE id=$1 AND owner_id=$2`,
		id, ownerID,
	)
	return err
}
