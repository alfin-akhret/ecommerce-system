package cart

import (
	"context"
	"errors"
	"fmt"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CartService struct {
	repo    CartUpdater
	product contracts.ProductManager
}

func NewCartService(repo CartUpdater, product contracts.ProductManager) *CartService {
	return &CartService{
		repo:    repo,
		product: product,
	}
}

var ErrCartNotFound = errors.New("cart not found")

func (s *CartService) AddItem(ctx context.Context, ownerID uuid.UUID, itemReq AddCartItemRequest) (string, error) {
	log := helper.LoggerFromCtx(ctx)
	lOwnerID := zap.String("owner_id", ownerID.String())

	pid, qty, err := parseCartItemRequest(itemReq)
	if err != nil {
		log.Warn(
			"Cart: invalid add item request",
			lOwnerID,
			zap.String("error_message", err.Error()),
		)
		return "", err
	}

	lProductID := zap.String("product_id", pid.String())
	lQty := zap.Int("qty", qty)

	cart, err := s.repo.Get(ctx, ownerID)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			cart, _ = NewCart(ownerID)
			log.Info("Cart: Created new cart", lOwnerID)
		} else {
			log.Error(
				"Cart: Failed to get cart",
				lOwnerID,
				lProductID,
				lQty,
				zap.String("error_message", err.Error()),
			)
			return "", err
		}
	}

	// get product price
	product, err := s.product.GetProductPrice(ctx, pid.String())
	if err != nil {
		log.Error(
			"Cart: Failed to get product price",
			lOwnerID,
			lProductID,
			lQty,
			zap.String("error_message", err.Error()),
		)
		return "", err
	}

	cartItem := &CartItem{
		ProductID: pid,
		Qty:       qty,
		Price:     product.GetPrice(),
	}

	if err := cart.AddItem(cartItem.ProductID, cartItem.Qty, cartItem.Price); err != nil {
		log.Error(
			"Cart: Failed to add item to cart",
			lOwnerID,
			lProductID,
			lQty,
			zap.String("error_message", err.Error()),
		)
		return "", err
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		log.Error(
			"Cart: Failed to save cart",
			lOwnerID,
			lProductID,
			lQty,
			zap.String("error_message", err.Error()),
		)
		return "", err
	}

	log.Info("Cart: Item added", lOwnerID, lProductID, lQty)

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

func (s *CartService) UpdateQuantity(ctx context.Context, ownerID uuid.UUID, itemReq AddCartItemRequest) (string, error) {
	pid, qty, err := parseCartItemRequest(itemReq)
	if err != nil {
		return "", err
	}

	cart, err := s.repo.Get(ctx, ownerID)
	if err != nil {
		return "", errors.New("cart not found")
	}

	if err := cart.UpdateQuantity(pid, qty); err != nil {
		return "", err
	}

	if err := s.repo.Save(ctx, cart); err != nil {
		return "", err
	}

	resp := fmt.Sprintf("item %v quantity updated to: %v", pid.String(), qty)

	return resp, nil
}

func (s *CartService) Get(ctx context.Context, ownerID uuid.UUID) (*GetCartResponse, error) {
	cart, err := s.repo.Get(ctx, ownerID)
	if err != nil {
		return nil, ErrCartNotFound
	}

	var cartItemsResponse []CartItemResponse
	for _, item := range cart.Items {
		cartItemsResponse = append(cartItemsResponse, toCartItemResponse(&item))
	}

	totalPrice := cart.Total()

	cartResponse := &GetCartResponse{
		Items:      cartItemsResponse,
		TotalPrice: helper.ToFloat(totalPrice),
	}

	return cartResponse, nil
}

func (s *CartService) GetCart(ctx context.Context, ownerID uuid.UUID) ([]contracts.CartItem, error) {
	cart, err := s.repo.Get(ctx, ownerID)
	if err != nil {
		return nil, ErrCartNotFound
	}

	var cartItems []contracts.CartItem
	for _, val := range cart.Items {
		cartItems = append(cartItems, contracts.CartItem{
			ProductID: val.ProductID,
			Qty:       val.Qty,
		})
	}

	return cartItems, nil
}
