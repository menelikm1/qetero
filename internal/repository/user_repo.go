package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/models"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, name, email, phone, password_hash, role)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		user.ID, user.Name, user.Email, user.Phone, user.PasswordHash, user.Role,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, name, email, phone, password_hash, role, verified, created_at, updated_at
		FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone,
		&user.PasswordHash, &user.Role, &user.Verified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, name, email, phone, role, verified, created_at, updated_at
		FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Phone,
		&user.Role, &user.Verified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, name, phone string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name=$1, phone=$2, updated_at=NOW() WHERE id=$3`,
		name, phone, id,
	)
	return err
}

func (r *UserRepo) GetByPhoneForAuth(ctx context.Context, phone string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, name, phone, email, password_hash, role, verified, created_at, updated_at
		FROM users WHERE phone = $1`
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Phone, &user.Email,
		&user.PasswordHash, &user.Role, &user.Verified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, name, phone, email, role, telegram_chat_id, verified, created_at, updated_at
		FROM users WHERE phone = $1`
	err := r.db.QueryRow(ctx, query, phone).Scan(
		&user.ID, &user.Name, &user.Phone, &user.Email,
		&user.Role, &user.TelegramChatID, &user.Verified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) GetByChatID(ctx context.Context, chatID int64) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, name, phone, email, role, telegram_chat_id, verified, created_at, updated_at
		FROM users WHERE telegram_chat_id = $1`
	err := r.db.QueryRow(ctx, query, chatID).Scan(
		&user.ID, &user.Name, &user.Phone, &user.Email,
		&user.Role, &user.TelegramChatID, &user.Verified,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) LinkTelegramChatID(ctx context.Context, userID uuid.UUID, chatID int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET telegram_chat_id=$1, updated_at=NOW() WHERE id=$2`,
		chatID, userID,
	)
	return err
}
