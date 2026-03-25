package cart

import (
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
)

type Handler struct {
	service *CartService
}

func NewHandler(service *CartService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) AddItem(w http.ResponseWriter, r *http.Request) error {
	var req AddCartItemRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := r.Context()
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	resp, err := h.service.AddItem(ctx, ownerID, req)
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}
