package auth

import (
	"errors"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/user"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
)

type AuthHandler struct {
	userService *user.UserService
}

func NewAuthHandler(userService *user.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	var req user.LoginRequest

	log := helper.LoggerFromCtx(ctx)

	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	log.Info("Auth: login attempt", zap.String("email", req.Email))

	account, err := h.userService.Login(ctx, req.Email, req.Password)
	if err != nil {

		if errors.Is(err, user.ErrInvalidCredentials) {
			log.Warn("Auth: authentication failed",
				zap.String("error_message", err.Error()))

			return helper.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		}

		log.Error("Auth: authentication failed",
			zap.String("email", req.Email),
			zap.String("error_message", err.Error()))

		return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	token, err := GenerateToken(account.ID.String())
	if err != nil {
		log.Error("Auth: failed generate token",
			zap.String("email", req.Email),
			zap.String("error_message", err.Error()))

		return helper.NewHTTPError(http.StatusInternalServerError, "failed to generate token")
	}

	resp := user.LoginResponse{
		Token: token,
	}

	log.Info("Auth: login succeeded",
		zap.String("email", req.Email),
		zap.String("user_id", account.ID.String()))

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
