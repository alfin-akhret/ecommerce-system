package helper

import (
	"errors"
	"net/http"
)

type AppHandler func(http.ResponseWriter, *http.Request) error

type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(status int, message string) error {
	return HTTPError{Status: status, Message: message}
}

func Handle(h AppHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			var httpErr HTTPError
			if errors.As(err, &httpErr) {
				WriteError(w, httpErr.Status, httpErr.Message)
				return
			}
			WriteError(w, http.StatusInternalServerError, err.Error())
		}
	}
}
