// mocks_test.go — repository interfaces and mock implementations for handler tests.
// All types are unexported and live only in the test binary (package handlers).
package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"qetero/internal/models"
	"qetero/internal/repository"
)

// ---------------------------------------------------------------------------
// Repository interfaces
// ---------------------------------------------------------------------------

type userStore interface {
	Create(ctx context.Context, user *models.User) error
	GetByPhoneForAuth(ctx context.Context, phone string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, name, phone string) error
}

type listingStore interface {
	Create(ctx context.Context, l *models.Listing) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Listing, error)
	Browse(ctx context.Context, f repository.ListingFilter) ([]models.Listing, error)
	Update(ctx context.Context, l *models.Listing) error
	Delete(ctx context.Context, id, ownerID uuid.UUID) error
}

type bookingStore interface {
	Create(ctx context.Context, b *models.Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.BookingStatus, reason string) error
	HasConflict(ctx context.Context, listingID uuid.UUID, start, end time.Time) (bool, error)
	GetBookedDates(ctx context.Context, listingID uuid.UUID) ([]repository.DateRange, error)
	GetByRenter(ctx context.Context, renterID uuid.UUID) ([]models.Booking, error)
	GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.Booking, error)
}

// ---------------------------------------------------------------------------
// Mock — userStore
// ---------------------------------------------------------------------------

type mockUserStore struct {
	createFn          func(ctx context.Context, user *models.User) error
	getByPhoneAuthFn  func(ctx context.Context, phone string) (*models.User, error)
	getByEmailFn      func(ctx context.Context, email string) (*models.User, error)
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*models.User, error)
	updateFn          func(ctx context.Context, id uuid.UUID, name, phone string) error
}

func (m *mockUserStore) Create(ctx context.Context, user *models.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}
func (m *mockUserStore) GetByPhoneForAuth(ctx context.Context, phone string) (*models.User, error) {
	if m.getByPhoneAuthFn != nil {
		return m.getByPhoneAuthFn(ctx, phone)
	}
	return nil, nil
}
func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockUserStore) Update(ctx context.Context, id uuid.UUID, name, phone string) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, name, phone)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock — listingStore
// ---------------------------------------------------------------------------

type mockListingStore struct {
	createFn  func(ctx context.Context, l *models.Listing) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*models.Listing, error)
	browseFn  func(ctx context.Context, f repository.ListingFilter) ([]models.Listing, error)
	updateFn  func(ctx context.Context, l *models.Listing) error
	deleteFn  func(ctx context.Context, id, ownerID uuid.UUID) error
}

func (m *mockListingStore) Create(ctx context.Context, l *models.Listing) error {
	if m.createFn != nil {
		return m.createFn(ctx, l)
	}
	return nil
}
func (m *mockListingStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Listing, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockListingStore) Browse(ctx context.Context, f repository.ListingFilter) ([]models.Listing, error) {
	if m.browseFn != nil {
		return m.browseFn(ctx, f)
	}
	return nil, nil
}
func (m *mockListingStore) Update(ctx context.Context, l *models.Listing) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, l)
	}
	return nil
}
func (m *mockListingStore) Delete(ctx context.Context, id, ownerID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, ownerID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock — bookingStore
// ---------------------------------------------------------------------------

type mockBookingStore struct {
	createFn          func(ctx context.Context, b *models.Booking) error
	getByIDFn         func(ctx context.Context, id uuid.UUID) (*models.Booking, error)
	updateStatusFn    func(ctx context.Context, id uuid.UUID, status models.BookingStatus, reason string) error
	hasConflictFn     func(ctx context.Context, listingID uuid.UUID, start, end time.Time) (bool, error)
	getBookedDatesFn  func(ctx context.Context, listingID uuid.UUID) ([]repository.DateRange, error)
	getByRenterFn     func(ctx context.Context, renterID uuid.UUID) ([]models.Booking, error)
	getByOwnerFn      func(ctx context.Context, ownerID uuid.UUID) ([]models.Booking, error)
}

func (m *mockBookingStore) Create(ctx context.Context, b *models.Booking) error {
	if m.createFn != nil {
		return m.createFn(ctx, b)
	}
	return nil
}
func (m *mockBookingStore) GetByID(ctx context.Context, id uuid.UUID) (*models.Booking, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockBookingStore) UpdateStatus(ctx context.Context, id uuid.UUID, status models.BookingStatus, reason string) error {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, id, status, reason)
	}
	return nil
}
func (m *mockBookingStore) HasConflict(ctx context.Context, listingID uuid.UUID, start, end time.Time) (bool, error) {
	if m.hasConflictFn != nil {
		return m.hasConflictFn(ctx, listingID, start, end)
	}
	return false, nil
}
func (m *mockBookingStore) GetBookedDates(ctx context.Context, listingID uuid.UUID) ([]repository.DateRange, error) {
	if m.getBookedDatesFn != nil {
		return m.getBookedDatesFn(ctx, listingID)
	}
	return nil, nil
}
func (m *mockBookingStore) GetByRenter(ctx context.Context, renterID uuid.UUID) ([]models.Booking, error) {
	if m.getByRenterFn != nil {
		return m.getByRenterFn(ctx, renterID)
	}
	return nil, nil
}
func (m *mockBookingStore) GetByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.Booking, error) {
	if m.getByOwnerFn != nil {
		return m.getByOwnerFn(ctx, ownerID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Testable handler types — identical logic to production handlers but wired
// to interfaces so we can inject mocks without a real database.
// ---------------------------------------------------------------------------

// testAuthHandler mirrors AuthHandler with interface-typed fields.
type testAuthHandler struct {
	users userStore
	cfg   testCfg
}

type testCfg struct {
	jwtSecret string
	expiry    time.Duration
}

// testListingHandler mirrors ListingHandler with interface-typed fields.
type testListingHandler struct {
	listings listingStore
	bookings bookingStore
}

// testBookingHandler mirrors BookingHandler with interface-typed fields.
type testBookingHandler struct {
	bookings bookingStore
	listings listingStore
}

// testUserHandler mirrors UserHandler with interface-typed fields.
type testUserHandler struct {
	users    userStore
	bookings bookingStore
}
