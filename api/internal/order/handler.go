package order

import (
	"errors"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/internal/product"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/go-chi/chi"
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

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) error {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	orderID := chi.URLParam(r, "id")
	if orderID == "" {
		return helper.NewHTTPError(http.StatusBadRequest, "missing order id")
	}

	order, err := h.service.GetOrder(r.Context(), userID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return helper.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, order)
	return nil
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) error {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	orders, err := h.service.ListOrders(r.Context(), userID)
	if err != nil {
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, orders)
	return nil
}
