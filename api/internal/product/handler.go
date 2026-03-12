package product

import (
	"errors"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/go-chi/chi"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) error {
	// Implementation for creating a product
	var req CreateProductRequest

	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	product, err := h.service.CreateProduct(r.Context(), req)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := toProductResponse(product)

	helper.WriteSuccess(w, http.StatusCreated, resp)
	return nil
}

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) error {
	products, err := h.service.ListProducts(r.Context())
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	helper.WriteSuccess(w, http.StatusOK, products)
	return nil
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	product, err := h.service.GetProductByID(r.Context(), id)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if product == nil {
		return helper.NewHTTPError(http.StatusNotFound, "product not found")
	}

	helper.WriteSuccess(w, http.StatusOK, product)
	return nil
}

func (h *Handler) UpdateStock(w http.ResponseWriter, r *http.Request) error {

	id := chi.URLParam(r, "id")

	var req UpdateStockRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return err
	}

	err := h.service.UpdateStock(r.Context(), id, req.Qty)
	if err != nil {
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "stock updated")

	return nil
}

func (h *Handler) ReserveStock(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	var req UpdateStockRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return err
	}

	err := h.service.ReserveInventory(r.Context(), id, req.Qty)
	if err != nil {
		if errors.Is(err, ErrNotEnoughStock) {
			return helper.NewHTTPError(400, err.Error())
		}

		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "stock reserved")

	return nil
}

func (h *Handler) ReleaseStock(w http.ResponseWriter, r *http.Request) error {

	id := chi.URLParam(r, "id")

	var req UpdateStockRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return err
	}

	err := h.service.ReleaseInventory(r.Context(), id, req.Qty)
	if err != nil {
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "stock released")

	return nil
}

func (h *Handler) ConfirmStock(w http.ResponseWriter, r *http.Request) error {

	id := chi.URLParam(r, "id")

	var req UpdateStockRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return err
	}

	err := h.service.ConfirmInventory(r.Context(), id, req.Qty)
	if err != nil {
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "stock confirmed")

	return nil
}
