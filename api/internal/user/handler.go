package user

import (
	"encoding/json"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/go-chi/chi"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(service *UserService) *UserHandler {
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
		helper.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.CreateUser(
		r.Context(),
		req.Name,
		req.Email,
		req.Password,
	)

	if err != nil {
		helper.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusCreated, resp)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		helper.WriteError(w, http.StatusNotFound, "user not found")
		return
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusOK, resp)
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		helper.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		helper.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusOK, resp)
}

func toUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
