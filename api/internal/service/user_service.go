package service

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
	"github.com/alfin-akhret/ecommerce-system/internal/repository"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

var ErrInvalidCredentials = errors.New("invalid credentials")

func (s *UserService) CreateUser(ctx context.Context, name string, email string, password string) (*model.User, error) {
	return s.createUser(ctx, name, email, password)
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	return s.createUser(ctx, req.Name, req.Email, req.Password)
}

func (s *UserService) createUser(ctx context.Context, name string, email string, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(ctx context.Context, email string, password string) (*model.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
