package cart

import (
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/auth"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/go-chi/chi"
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

func (h *Handler) DeleteCart(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	resp, err := h.service.DeleteCart(ctx, ownerID)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) RemoveItem(w http.ResponseWriter, r *http.Request) error {
	productID, err := uuid.Parse(chi.URLParam(r, "product_id"))
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, "invalid product id")
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

	resp, err := h.service.RemoveItem(ctx, ownerID, productID)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil

}

func (h *Handler) UpdateQuantity(w http.ResponseWriter, r *http.Request) error {
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

	resp, err := h.service.UpdateQuantity(ctx, ownerID, req)
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return helper.NewHTTPError(http.StatusUnauthorized, "missing user")
	}

	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	resp, err := h.service.Get(ctx, ownerID)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}
