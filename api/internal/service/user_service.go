package service

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
	"github.com/alfin-akhret/ecommerce-system/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) CreateUser(ctx context.Context, name string, email string) (*model.User, error) {
	user := &model.User{
		Name:  name,
		Email: email,
	}

	err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}
