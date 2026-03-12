package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `
	insert into users (id, name, email, password_hash)
	values ($1, $2, $3, $4)
	returning created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
	).Scan(&user.CreatedAt)

	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
	select id, name, email, password_hash, created_at
	from users
	where id = $1::uuid
	`

	var user User
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

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
	select id, name, email, password_hash, created_at
	from users
	where email = $1
	limit 1
	`

	var user User
	err := r.db.QueryRow(ctx, query, email).Scan(
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
