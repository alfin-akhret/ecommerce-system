package payment

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/go-chi/chi"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CreatePayment endpoint
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) error {
	var req CreatePaymentRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.service.CreatePayment(r.Context(), req.OrderID, req.Amount, req.PaymentMethod)
	if err != nil {
		if errors.Is(err, ErrInvalidAmount) || errors.Is(err, ErrInvalidPaymentMethod) {
			return helper.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusCreated, resp)
	return nil
}

// UpdatePaymentStatus endpoint (e.g., webhook)
func (h *Handler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) error {
	var req UpdatePaymentStatusRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	paymentID := chi.URLParam(r, "payment_id")
	if paymentID == "" {
		return helper.NewHTTPError(http.StatusBadRequest, "payment_id required")
	}

	payment, err := h.service.UpdatePaymentStatus(r.Context(), paymentID, req.Status)
	if err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			return helper.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, payment)
	return nil
}

// GetPayment endpoint
func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) error {
	paymentID := chi.URLParam(r, "payment_id")
	if paymentID == "" {
		return helper.NewHTTPError(http.StatusBadRequest, "payment_id required")
	}

	payment, err := h.service.GetPaymentByID(r.Context(), paymentID)
	if err != nil {
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, payment)
	return nil
}

// ProcessPaymentSuccess endpoint
func (h *Handler) ProcessPaymentSuccess(w http.ResponseWriter, r *http.Request) error {
	paymentID := chi.URLParam(r, "payment_id")
	if paymentID == "" {
		return helper.NewHTTPError(http.StatusBadRequest, "payment_id required")
	}

	if err := h.service.ProcessPaymentSuccess(r.Context(), paymentID); err != nil {
		if errors.Is(err, ErrPaymentNotFound) {
			return helper.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrInvalidStatus) {
			return helper.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, ErrOrderStatusUpdaterNotSet) {
			return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "payment processed: "+StatusSuccess)
	return nil
}

// ProcessPaymentFailed endpoint
func (h *Handler) ProcessPaymentFailed(w http.ResponseWriter, r *http.Request) error {
	paymentID := chi.URLParam(r, "payment_id")
	if paymentID == "" {
		return helper.NewHTTPError(http.StatusBadRequest, "payment_id required")
	}

	if err := h.service.ProcessPaymentFailed(r.Context(), paymentID); err != nil {
		if errors.Is(err, ErrInvalidStatus) {
			return helper.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		if errors.Is(err, ErrPaymentNotFound) {
			return helper.NewHTTPError(http.StatusNotFound, err.Error())
		}
		if errors.Is(err, ErrOrderStatusUpdaterNotSet) {
			return helper.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return err
	}

	helper.WriteSuccess(w, http.StatusOK, "payment processed: "+StatusFailed)
	return nil
}

// Mock payment page to simulate a payment gateway.
// In production, we integrate with the payment gateway via H2H
// and redirect the user only after receiving a successful response.
func (h *Handler) PaymentPage(w http.ResponseWriter, r *http.Request) {
	paymentID := chi.URLParam(r, "payment_id")

	html := fmt.Sprintf(`
	<h1>Payment Gateway</h1>
	<p>Payment ID: %s</p>

	<form method="POST" action="/payments/%s/success">
		<button type="submit">Pay Success</button>
	</form>

	<form method="POST" action="/payments/%s/fail">
		<button type="submit">Pay Failed</button>
	</form>
	`, paymentID, paymentID, paymentID)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}
