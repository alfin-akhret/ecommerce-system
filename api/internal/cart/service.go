package cart

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type CartService struct {
	repo CartUpdater
}

func NewCartService(repo CartUpdater) *CartService {
	return &CartService{
		repo: repo,
	}
}

func (s *CartService) AddItem(ctx context.Context, ownerID uuid.UUID, itemReq AddCartItemRequest) (string, error) {

	cartItem, err := toCartItem(itemReq)
	if err != nil {
		return "", err
	}

	cart, _ := s.repo.Get(ctx, ownerID)

	if cart == nil {
		cart, err = NewCart(ownerID)
		if err != nil {
			return "", err
		}
	}

	if err := cart.AddItem(cartItem.ProductID, cartItem.Qty, cartItem.Price); err != nil {
		return "", err
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		return "", err
	}

	return "cart updated", nil
}

func (s *CartService) DeleteCart(ctx context.Context, ownerID uuid.UUID) (string, error) {
	if err := s.repo.Delete(ctx, ownerID); err != nil {
		return "", err
	}

	return "cart deleted", nil
}

func (s *CartService) RemoveItem(ctx context.Context, ownerID uuid.UUID, productID uuid.UUID) (string, error) {
	cart, err := s.repo.Get(ctx, ownerID)
	if err != nil {
		return "", err
	}

	if err := cart.RemoveItem(productID); err != nil {
		return "", err
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		return "", err
	}

	resp := fmt.Sprintf("item %v had been removed from cart", productID.String())

	return resp, nil
}
