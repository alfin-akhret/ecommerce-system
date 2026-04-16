package user

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *UserRepository
}

func NewUserService(repo *UserRepository) *UserService {
	return &UserService{repo: repo}
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (s *UserService) CreateUser(ctx context.Context, name string, email string, password string) (*User, error) {
	return s.createUser(ctx, name, email, password)
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*User, error) {
	return s.createUser(ctx, req.Name, req.Email, req.Password)
}

func (s *UserService) createUser(ctx context.Context, name string, email string, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.New(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(ctx context.Context, email string, password string) (*User, error) {
	log := helper.LoggerFromCtx(ctx)
	lEmail := zap.String("email", email)

	log.Info("Auth: Processing user authentication", lEmail)

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			log.Error("Auth: invalid credentials",
				lEmail,
				zap.String("error_message", ErrInvalidCredentials.Error()),
			)
			return nil, ErrInvalidCredentials
		}
		log.Error("Auth: error processing user authentication",
			lEmail,
			zap.String("error_message", err.Error()),
		)
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Error("Auth: invalid credentials",
			lEmail,
			zap.String("error_message", ErrInvalidCredentials.Error()),
		)
		return nil, ErrInvalidCredentials
	}

	log.Info("Auth: User authenticated", lEmail, zap.String("user_id", user.ID.String()))

	return user, nil
}
