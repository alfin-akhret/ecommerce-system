package user

import (
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

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) error {
	var req createUserRequest

	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.service.CreateUser(
		r.Context(),
		req.Name,
		req.Email,
		req.Password,
	)

	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusCreated, resp)
	return nil
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")

	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		return helper.NewHTTPError(http.StatusNotFound, "user not found")
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterRequest

	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := toUserResponse(user)

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

func toUserResponse(user *User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
