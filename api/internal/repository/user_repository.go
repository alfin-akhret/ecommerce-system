package repository

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
	insert into users (name, email, password_hash)
	values ($1, $2, $3)
	returning id, created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt)

	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
	select id, name, email, password_hash, created_at
	from users
	where id = $1::uuid
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
