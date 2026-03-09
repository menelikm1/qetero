package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
)

type ListingFilter struct {
	Category string
	Location string
	MinPrice *float64
	MaxPrice *float64
	Page     int
	Limit    int
}

type ListingRepo struct {
	db *pgxpool.Pool
}

func NewListingRepo(db *pgxpool.Pool) *ListingRepo {
	return &ListingRepo{db: db}
}

func (r *ListingRepo) Create(ctx context.Context, l *models.Listing) error {
	specsJSON, err := json.Marshal(l.Specs)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO listings (id, owner_id, title, category, description, location, price_per_day, minimum_days, images, specs, is_available, year, total_hours, last_serviced)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14)
		RETURNING status, is_available, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		l.ID, l.OwnerID, l.Title, l.Category, l.Description,
		l.Location, l.PricePerDay, l.MinimumDays, l.Images, specsJSON, l.IsAvailable,
		l.Year, l.TotalHours, l.LastServiced,
	).Scan(&l.Status, &l.IsAvailable, &l.CreatedAt, &l.UpdatedAt)
}

func (r *ListingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	l := &models.Listing{}
	var specsRaw []byte
	query := `
		SELECT id, owner_id, title, category, description, location, price_per_day,
		       minimum_days, images, specs, is_available, status, year, total_hours, last_serviced,
		       created_at, updated_at
		FROM listings WHERE id=$1 AND deleted_at IS NULL`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&l.ID, &l.OwnerID, &l.Title, &l.Category, &l.Description,
		&l.Location, &l.PricePerDay, &l.MinimumDays, &l.Images, &specsRaw,
		&l.IsAvailable, &l.Status, &l.Year, &l.TotalHours, &l.LastServiced,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	l.Specs = specsRaw
	return l, nil
}

func (r *ListingRepo) Browse(ctx context.Context, f ListingFilter) ([]models.Listing, error) {
	args := []any{}
	conditions := []string{"deleted_at IS NULL", "is_available = true", "status = 'active'"}
	i := 1

	if f.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", i))
		args = append(args, f.Category)
		i++
	}
	if f.Location != "" {
		conditions = append(conditions, fmt.Sprintf("location ILIKE $%d", i))
		args = append(args, "%"+f.Location+"%")
		i++
	}
	if f.MinPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price_per_day >= $%d", i))
		args = append(args, *f.MinPrice)
		i++
	}
	if f.MaxPrice != nil {
		conditions = append(conditions, fmt.Sprintf("price_per_day <= $%d", i))
		args = append(args, *f.MaxPrice)
		i++
	}

	limit := f.Limit
	if limit == 0 || limit > 50 {
		limit = 20
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT id, owner_id, title, category, description, location, price_per_day,
		       minimum_days, images, specs, is_available, status, year, total_hours, last_serviced,
		       created_at, updated_at
		FROM listings
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(conditions, " AND "), i, i+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var listings []models.Listing
	for rows.Next() {
		var l models.Listing
		var specsRaw []byte
		if err := rows.Scan(
			&l.ID, &l.OwnerID, &l.Title, &l.Category, &l.Description,
			&l.Location, &l.PricePerDay, &l.MinimumDays, &l.Images, &specsRaw,
			&l.IsAvailable, &l.Status, &l.Year, &l.TotalHours, &l.LastServiced,
			&l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		l.Specs = specsRaw
		listings = append(listings, l)
	}
	return listings, rows.Err()
}

func (r *ListingRepo) Update(ctx context.Context, l *models.Listing) error {
	specsJSON, err := json.Marshal(l.Specs)
	if err != nil {
		return err
	}
	query := `
		UPDATE listings
		SET title=$1, category=$2, description=$3, location=$4, price_per_day=$5,
		    minimum_days=$6, specs=$7::jsonb, is_available=$8, updated_at=NOW()
		WHERE id=$9 AND owner_id=$10`
	_, err = r.db.Exec(ctx, query,
		l.Title, l.Category, l.Description, l.Location, l.PricePerDay,
		l.MinimumDays, specsJSON, l.IsAvailable, l.ID, l.OwnerID,
	)
	return err
}

// UpdateStatus sets the listing status and syncs is_available accordingly.
func (r *ListingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ListingStatus) error {
	isAvailable := status == models.ListingStatusActive
	_, err := r.db.Exec(ctx,
		`UPDATE listings SET status=$1, is_available=$2, updated_at=NOW() WHERE id=$3`,
		status, isAvailable, id,
	)
	return err
}

func (r *ListingRepo) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE listings SET deleted_at=NOW() WHERE id=$1 AND owner_id=$2`,
		id, ownerID,
	)
	return err
}
