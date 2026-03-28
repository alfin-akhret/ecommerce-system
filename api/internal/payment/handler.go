package payment

import (
	"errors"
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

	resp, err := h.service.CreatePayment(r.Context(), req.OrderID, helper.ToCents(req.Amount), req.PaymentMethod)
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

	resp := toPaymentResponse(payment)

	helper.WriteSuccess(w, http.StatusOK, resp)
	return nil
}

// HandleCallback endpoint
func (h *Handler) HandleCallback(w http.ResponseWriter, r *http.Request) error {
	var req PaymentCallbackRequest
	if err := helper.DecodeJSON(r, &req); err != nil {
		return helper.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := h.service.HandleCallback(req); err != nil {
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

	helper.WriteSuccess(w, http.StatusOK, "callback processed")
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

func toPaymentResponse(payment *Payment) PaymentByIDResponse {
	return PaymentByIDResponse{
		ID:            payment.ID.String(),
		OrderID:       payment.OrderID.String(),
		Amount:        helper.ToFloat(payment.Amount),
		Status:        payment.Status,
		PaymentMethod: payment.PaymentMethod,
		PaidAt:        helper.FormatOptionalTime(payment.PaidAt),
		CreatedAt:     helper.FormatOptionalTime(payment.CreatedAt),
		UpdatedAt:     helper.FormatOptionalTime(payment.UpdatedAt),
		ExpiredAt:     helper.FormatOptionalTime(payment.ExpiredAt),
	}
}
