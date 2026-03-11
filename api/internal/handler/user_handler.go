package handler

import (
	"encoding/json"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
	"github.com/alfin-akhret/ecommerce-system/internal/service"
	"github.com/go-chi/chi"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.CreateUser(
		r.Context(),
		req.Name,
		req.Email,
		req.Password,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toUserResponse(user)

	writeSuccess(w, http.StatusCreated, resp)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	resp := toUserResponse(user)

	writeSuccess(w, http.StatusOK, resp)
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toUserResponse(user)

	writeSuccess(w, http.StatusOK, resp)
}
