package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
)

type DateRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type BookingRepo struct {
	db *pgxpool.Pool
}

func NewBookingRepo(db *pgxpool.Pool) *BookingRepo {
	return &BookingRepo{db: db}
}

func (r *BookingRepo) Create(ctx context.Context, b *models.Booking) error {
	query := `
		INSERT INTO bookings (id, listing_id, renter_id, owner_id, start_date, end_date, total_days, total_price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		b.ID, b.ListingID, b.RenterID, b.OwnerID,
		b.StartDate, b.EndDate, b.TotalDays, b.TotalPrice, b.Status,
	).Scan(&b.CreatedAt, &b.UpdatedAt)
}

func (r *BookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error) {
	b := &models.Booking{}
	query := `
		SELECT id, listing_id, renter_id, owner_id, start_date, end_date,
		       total_days, total_price, status, deposit_status,
		       COALESCE(deposit_ref, ''), COALESCE(cancellation_reason, ''),
		       created_at, updated_at
		FROM bookings WHERE id=$1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.ListingID, &b.RenterID, &b.OwnerID,
		&b.StartDate, &b.EndDate, &b.TotalDays, &b.TotalPrice,
		&b.Status, &b.DepositStatus, &b.DepositRef, &b.CancellationReason,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BookingRepo) UpdateDeposit(ctx context.Context, id uuid.UUID, ref string, status models.DepositStatus) error {
	_, err := r.db.Exec(ctx,
		`UPDATE bookings SET deposit_ref=$1, deposit_status=$2, updated_at=NOW() WHERE id=$3`,
		ref, status, id,
	)
	return err
}

func (r *BookingRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.BookingStatus, reason string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE bookings SET status=$1, cancellation_reason=$2, updated_at=NOW() WHERE id=$3`,
		status, reason, id,
	)
	return err
}

// HasConflict returns true if the listing has a confirmed/active booking or a manual block overlapping the given dates.
func (r *BookingRepo) HasConflict(ctx context.Context, listingID uuid.UUID, start, end time.Time) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM bookings
			WHERE listing_id=$1
			  AND status IN ('confirmed','active')
			  AND NOT (end_date <= $2 OR start_date >= $3)
		) OR EXISTS(
			SELECT 1 FROM listing_blocks
			WHERE listing_id=$1
			  AND NOT (end_date <= $2 OR start_date >= $3)
		)`
	var exists bool
	err := r.db.QueryRow(ctx, query, listingID, start, end).Scan(&exists)
	return exists, err
}

// GetBookedDates returns future confirmed/active booking ranges for a listing (used for availability display).
func (r *BookingRepo) GetBookedDates(ctx context.Context, listingID uuid.UUID) ([]DateRange, error) {
	query := `
		SELECT start_date, end_date FROM bookings
		WHERE listing_id=$1
		  AND status IN ('confirmed','active')
		  AND end_date >= CURRENT_DATE
		ORDER BY start_date`
	rows, err := r.db.Query(ctx, query, listingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ranges []DateRange
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return nil, err
		}
		ranges = append(ranges, DateRange{
			Start: start.Format("2006-01-02"),
			End:   end.Format("2006-01-02"),
		})
	}
	return ranges, rows.Err()
}

func (r *BookingRepo) GetByRenter(ctx context.Context, renterID uuid.UUID) ([]models.Booking, error) {
	return r.scanBookings(ctx,
		`SELECT id, listing_id, renter_id, owner_id, start_date, end_date, total_days, total_price,
		        status, COALESCE(cancellation_reason, ''), created_at, updated_at
		 FROM bookings WHERE renter_id=$1 ORDER BY created_at DESC`,
		renterID,
	)
}

func (r *BookingRepo) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.Booking, error) {
	return r.scanBookings(ctx,
		`SELECT id, listing_id, renter_id, owner_id, start_date, end_date, total_days, total_price,
		        status, COALESCE(cancellation_reason, ''), created_at, updated_at
		 FROM bookings WHERE owner_id=$1 ORDER BY created_at DESC`,
		ownerID,
	)
}

func (r *BookingRepo) scanBookings(ctx context.Context, query string, arg uuid.UUID) ([]models.Booking, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		var b models.Booking
		if err := rows.Scan(
			&b.ID, &b.ListingID, &b.RenterID, &b.OwnerID,
			&b.StartDate, &b.EndDate, &b.TotalDays, &b.TotalPrice,
			&b.Status, &b.CancellationReason, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}
