package product

import (
	"encoding/json"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating a product
	var req CreateProductRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		helper.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	product, err := h.service.CreateProduct(r.Context(), req)
	if err != nil {
		helper.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ProductResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
	}

	helper.WriteSuccess(w, http.StatusCreated, resp)
}
