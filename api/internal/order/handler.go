package order

import (
	"errors"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) error {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	var req CreateOrderRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.CreateOrder(r.Context(), userID, req); err != nil {
		if errors.Is(err, ErrInvalidQty) || errors.Is(err, product.ErrNotEnoughStock) {
			return helper.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusCreated, "order created")
	return nil
}
