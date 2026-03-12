package product

import (
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
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
