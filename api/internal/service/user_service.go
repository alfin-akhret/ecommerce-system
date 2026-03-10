package service

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
	"github.com/alfin-akhret/ecommerce-system/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, name string, email string, password string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}
