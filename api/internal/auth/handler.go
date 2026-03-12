package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/user"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
)

type AuthHandler struct {
	userService *user.UserService
}

func NewAuthHandler(userService *user.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req user.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		helper.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	account, err := h.userService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			helper.WriteError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		helper.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	token, err := GenerateToken(account.ID.String())
	if err != nil {
		helper.WriteError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	resp := user.LoginResponse{
		Token: token,
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r.Context())
	if !ok || userID == "" {
		helper.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	helper.WriteSuccess(w, http.StatusOK, map[string]string{"user_id": userID})
}
