package handler

import (
	"encoding/json"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/internal/model"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func toUserResponse(user *model.User) model.UserResponse {
	return model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func writeSuccess(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, v)
}
