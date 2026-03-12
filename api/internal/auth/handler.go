package auth

import (
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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	var req user.LoginRequest

	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	account, err := h.userService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			return helper.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}
		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	token, err := GenerateToken(account.ID.String())
	if err != nil {
		return helper.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	resp := user.LoginResponse{
		Token: token,
	}

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) error {
	userID, ok := GetUserID(r.Context())
	if !ok || userID == "" {
		return helper.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	helper.WriteSuccess(w, http.StatusOK, map[string]string{"user_id": userID})
	return nil
}
