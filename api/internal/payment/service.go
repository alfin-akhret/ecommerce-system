package payment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type Service struct {
	db                 *pgxpool.Pool
	repo               *Repository
	orderStatusUpdater contracts.OrderUpdater
}

const (
	StatusPending   = "PENDING"
	StatusSuccess   = "SUCCESS"
	StatusFailed    = "FAILED"
	StatusCancelled = "CANCELLED"
	StatusPaid      = "PAID"
	StatusExpired   = "EXPIRED"
)

const paymentURL string = "http://localhost:8081/pay?payment_id="
const paymentExpiry = 1 * time.Minute // todo: move to config

var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrInvalidPaymentMethod = errors.New("payment method is required")
var ErrInvalidStatus = errors.New("invalid payment status")
var ErrOrderStatusUpdaterNotSet = errors.New("order status updater is not configured")
var ErrPaymentExpired = errors.New("payment expired")

func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:   db,
		repo: NewRepository(db),
	}
}

func (s *Service) SetOrderStatusUpdater(updater contracts.OrderUpdater) {
	s.orderStatusUpdater = updater
}

// Create payment (return DTO with payment URL)
func (s *Service) CreatePayment(ctx context.Context, orderID string, amount int64, paymentMethod string) (*CreatePaymentResponse, error) {
	return s.createPayment(ctx, s.repo, orderID, amount, paymentMethod)
}

func (s *Service) CreatePaymentWithTx(ctx context.Context, tx pgx.Tx,
	orderID string, amount int64, paymentMethod string) (*contracts.PaymentCreateResult, error) {

	// logger
	log := helper.LoggerFromCtx(ctx)
	lOrderID := zap.String("order_id", orderID)
	lAmount := zap.Int64("amount", amount)
	lPaymentMethod := zap.String("payment_method", paymentMethod)

	log.Info("Payment: Creating payment with transaction", lOrderID, lAmount, lPaymentMethod)

	// tracer
	tr := otel.Tracer("payment.service")
	ctx, span := tr.Start(ctx, "payment.service.CreatePaymentWithTx")
	defer span.End()
	span.SetAttributes(
		attribute.String("order_id", orderID),
		attribute.Int64("amount", amount),
		attribute.String("payment_method", paymentMethod),
	)

	// create payment
	resp, err := s.createPayment(ctx, s.repo.WithTx(tx), orderID, amount, paymentMethod)
	if err != nil {
		span.SetAttributes(attribute.Bool("payment.created", false))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Payment: Failed to create payment with transaction",
			lOrderID,
			lAmount,
			lPaymentMethod,
			zap.Error(err),
		)
		return nil, err
	}

	span.SetAttributes(attribute.Bool("payment.created", true))
	log.Info(
		"Payment: Payment created with transaction",
		lOrderID,
		zap.String("payment_id", resp.ID),
		lAmount,
		lPaymentMethod,
	)

	return &contracts.PaymentCreateResult{
		ID:         resp.ID,
		PaymentURL: resp.PaymentURL,
		ExpiredAt:  resp.ExpiredAt,
	}, nil
}

// Update payment status and optionally set PaidAt
func (s *Service) UpdatePaymentStatus(ctx context.Context, paymentID string, status string) (*Payment, error) {
	return s.updatePaymentStatus(ctx, s.repo, paymentID, status)
}

func (s *Service) UpdatePaymentStatusWithTx(ctx context.Context, tx pgx.Tx, paymentID string, status string) (*Payment, error) {
	return s.updatePaymentStatus(ctx, s.repo.WithTx(tx), paymentID, status)
}

// Get payment by ID
func (s *Service) GetPaymentByID(ctx context.Context, paymentID string) (*Payment, error) {

	return s.repo.GetByID(ctx, paymentID)

}

// Get payment by order ID
func (s *Service) GetPaymentByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}

func (s *Service) createPayment(ctx context.Context, repo *Repository,
	orderID string, amount int64, paymentMethod string) (*CreatePaymentResponse, error) {

	// logger
	log := helper.LoggerFromCtx(ctx)
	lOrderID := zap.String("order_id", orderID)
	lAmount := zap.Int64("amount", amount)
	lPaymentMethod := zap.String("payment_method", paymentMethod)

	log.Info("Payment: Creating payment", lOrderID, lAmount, lPaymentMethod)

	// tracer
	tr := otel.Tracer("payment.service")
	ctx, span := tr.Start(ctx, "payment.service.createPayment")
	defer span.End()
	span.SetAttributes(
		attribute.String("order_id", orderID),
		attribute.Int64("amount", amount),
		attribute.String("payment_method", paymentMethod),
	)

	if amount <= 0 {
		span.SetAttributes(attribute.Bool("payment.created", false))
		span.RecordError(ErrInvalidAmount)
		span.SetStatus(codes.Error, ErrInvalidAmount.Error())
		log.Warn("Payment: Invalid payment amount", lOrderID, lAmount, lPaymentMethod, zap.Error(ErrInvalidAmount))
		return nil, ErrInvalidAmount
	}
	if strings.TrimSpace(paymentMethod) == "" {
		span.SetAttributes(attribute.Bool("payment.created", false))
		span.RecordError(ErrInvalidPaymentMethod)
		span.SetStatus(codes.Error, ErrInvalidPaymentMethod.Error())
		log.Warn("Payment: Payment method is required", lOrderID, lAmount, lPaymentMethod, zap.Error(ErrInvalidPaymentMethod))
		return nil, ErrInvalidPaymentMethod
	}

	paymentID := uuid.New()
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		span.SetAttributes(attribute.Bool("payment.created", false))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Payment: Failed to parse order ID", lOrderID, lAmount, lPaymentMethod, zap.Error(err))
		return nil, err
	}

	expiredAt := time.Now().UTC().Add(paymentExpiry)

	p := &Payment{
		ID:            paymentID,
		OrderID:       orderUUID,
		Amount:        amount,
		Status:        StatusPending,
		PaymentMethod: paymentMethod,
		ExpiredAt:     &expiredAt,
	}

	if err := repo.CreatePayment(ctx, p); err != nil {
		span.SetAttributes(attribute.Bool("payment.created", false))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error(
			"Payment: Failed to create payment",
			lOrderID,
			zap.String("payment_id", paymentID.String()),
			lAmount,
			lPaymentMethod,
			zap.Error(err),
		)
		return nil, err
	}

	paymentURL := paymentURL + paymentID.String()

	span.SetAttributes(attribute.Bool("payment.created", true))
	log.Info(
		"Payment: Payment created",
		lOrderID,
		zap.String("payment_id", paymentID.String()),
		lAmount,
		lPaymentMethod,
		zap.Time("expired_at", expiredAt),
	)

	return &CreatePaymentResponse{
		ID:            paymentID.String(),
		OrderID:       orderID,
		Status:        p.Status,
		Amount:        helper.ToFloat(amount),
		PaymentMethod: paymentMethod,
		PaymentURL:    paymentURL,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
		ExpiredAt:     &expiredAt,
	}, nil
}

func (s *Service) updatePaymentStatus(ctx context.Context, repo *Repository, paymentID string, status string) (*Payment, error) {
	normalizedStatus, err := normalizeStatus(status)
	if err != nil {
		return nil, err
	}

	paidAt := paidAtForStatus(normalizedStatus)

	if err := repo.UpdateStatus(ctx, paymentID, normalizedStatus, paidAt); err != nil {
		return nil, err
	}

	return repo.GetByID(ctx, paymentID)
}

func paidAtForStatus(status string) *time.Time {
	if status == StatusSuccess {
		t := time.Now()
		return &t
	}

	return nil
}

func normalizeStatus(status string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case StatusPending, StatusSuccess, StatusFailed, StatusCancelled:
		return normalized, nil
	default:
		return "", ErrInvalidStatus
	}
}

func (s *Service) ProcessPaymentSuccess(ctx context.Context, paymentID string) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", paymentID)

	log.Info("Payment: Processing payment success", lPaymentID)

	return s.processPayment(ctx, paymentID, StatusSuccess, func(tx pgx.Tx, payment *Payment) error {
		if err := s.orderStatusUpdater.ConfirmOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
			log.Error(
				"Payment: Failed to confirm order stock after payment success",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				zap.Error(err),
			)
			return err
		}

		log.Info(
			"Payment: Order stock confirmed after payment success",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
		)

		return s.orderStatusUpdater.UpdateOrderStatusWithTx(
			ctx,
			tx,
			payment.OrderID.String(),
			StatusPaid,
		)
	})
}

func (s *Service) ProcessPaymentFailed(ctx context.Context, paymentID string) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", paymentID)

	log.Info("Payment: Processing payment failure", lPaymentID)

	return s.processPayment(ctx, paymentID, StatusFailed, func(tx pgx.Tx, payment *Payment) error {
		if err := s.orderStatusUpdater.ReleaseOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
			log.Error(
				"Payment: Failed to release order stock after payment failure",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				zap.Error(err),
			)
			return err
		}

		log.Info(
			"Payment: Order stock released after payment failure",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
		)

		return s.orderStatusUpdater.UpdateOrderStatusWithTx(
			ctx,
			tx,
			payment.OrderID.String(),
			StatusCancelled,
		)
	})
}

func (s *Service) processPayment(
	ctx context.Context,
	paymentID string,
	nextStatus string,
	afterUpdate func(tx pgx.Tx, payment *Payment) error,
) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", paymentID)
	lNextStatus := zap.String("next_status", nextStatus)

	log.Info("Payment: Processing payment status transition", lPaymentID, lNextStatus)

	if s.orderStatusUpdater == nil {
		log.Error("Payment: Order status updater is not configured", lPaymentID, lNextStatus, zap.Error(ErrOrderStatusUpdaterNotSet))
		return ErrOrderStatusUpdaterNotSet
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Error("Payment: Failed to begin transaction for payment processing", lPaymentID, lNextStatus, zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	paymentRepo := s.repo.WithTx(tx)

	payment, err := paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		log.Error("Payment: Failed to get payment", lPaymentID, lNextStatus, zap.Error(err))
		return err
	}

	if payment.Status != StatusPending {
		log.Warn(
			"Payment: Payment is not pending",
			lPaymentID,
			zap.String("current_status", payment.Status),
			lNextStatus,
			zap.Error(ErrInvalidStatus),
		)
		return ErrInvalidStatus
	}

	var paidAt *time.Time
	if nextStatus == StatusSuccess {
		now := time.Now().UTC()
		paidAt = &now
	}

	if err := paymentRepo.UpdateStatus(ctx, paymentID, nextStatus, paidAt); err != nil {
		log.Error(
			"Payment: Failed to update payment status",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
			lNextStatus,
			zap.Error(err),
		)
		return err
	}

	log.Info(
		"Payment: Payment status updated",
		lPaymentID,
		zap.String("order_id", payment.OrderID.String()),
		zap.String("previous_status", payment.Status),
		lNextStatus,
	)

	if afterUpdate != nil {
		if err := afterUpdate(tx, payment); err != nil {
			log.Error(
				"Payment: Failed to run post-payment update",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				lNextStatus,
				zap.Error(err),
			)
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error(
			"Payment: Failed to commit payment processing transaction",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
			lNextStatus,
			zap.Error(err),
		)
		return err
	}

	log.Info(
		"Payment: Payment processing completed",
		lPaymentID,
		zap.String("order_id", payment.OrderID.String()),
		lNextStatus,
	)

	return nil
}

func (s *Service) HandleCallback(ctx context.Context, req PaymentCallbackRequest) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", req.PaymentID)
	lRequestedStatus := zap.String("requested_status", req.Status)

	log.Info("Payment: Handling payment callback", lPaymentID, lRequestedStatus)

	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "HandleCallback")
	defer span.End()

	span.SetAttributes(
		attribute.String("payment_id", req.PaymentID),
		attribute.String("status", req.Status),
	)

	status, err := normalizeStatus(req.Status)
	if err != nil {
		log.Warn("Payment: Invalid callback status", lPaymentID, lRequestedStatus, zap.Error(err))
		return err
	}

	log.Info("Payment: Callback status normalized", lPaymentID, lRequestedStatus, zap.String("normalized_status", status))

	switch status {
	case StatusSuccess:
		return s.processCallback(ctx, req.PaymentID, StatusSuccess, func(tx pgx.Tx, payment *Payment) error {
			if err := s.orderStatusUpdater.ConfirmOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
				log.Error(
					"Payment: Failed to confirm order stock during callback",
					lPaymentID,
					zap.String("order_id", payment.OrderID.String()),
					zap.String("normalized_status", status),
					zap.Error(err),
				)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}

			log.Info(
				"Payment: Order stock confirmed during callback",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				zap.String("normalized_status", status),
			)

			return s.orderStatusUpdater.UpdateOrderStatusWithTx(
				ctx,
				tx,
				payment.OrderID.String(),
				StatusPaid,
			)
		})
	case StatusFailed:
		return s.processCallback(ctx, req.PaymentID, StatusFailed, func(tx pgx.Tx, payment *Payment) error {
			if err := s.orderStatusUpdater.ReleaseOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
				log.Error(
					"Payment: Failed to release order stock during callback",
					lPaymentID,
					zap.String("order_id", payment.OrderID.String()),
					zap.String("normalized_status", status),
					zap.Error(err),
				)
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}

			log.Info(
				"Payment: Order stock released during callback",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				zap.String("normalized_status", status),
			)

			return s.orderStatusUpdater.UpdateOrderStatusWithTx(
				ctx,
				tx,
				payment.OrderID.String(),
				StatusCancelled,
			)
		})
	default:
		return ErrInvalidStatus
	}
}

func (s *Service) processCallback(
	ctx context.Context,
	paymentID string,
	nextStatus string,
	afterUpdate func(tx pgx.Tx, payment *Payment) error,
) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", paymentID)
	lNextStatus := zap.String("next_status", nextStatus)

	log.Info("Payment: Processing callback status transition", lPaymentID, lNextStatus)

	tr := otel.Tracer("payment-service")
	ctx, span := tr.Start(ctx, "processCallback")
	defer span.End()

	span.SetAttributes(attribute.String("payment_id", paymentID),
		attribute.String("payment_next_status", nextStatus),
	)

	if s.orderStatusUpdater == nil {
		log.Error("Payment: Order status updater is not configured for callback", lPaymentID, lNextStatus, zap.Error(ErrOrderStatusUpdaterNotSet))
		return ErrOrderStatusUpdaterNotSet
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		log.Error("Payment: Failed to begin callback transaction", lPaymentID, lNextStatus, zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	paymentRepo := s.repo.WithTx(tx)

	payment, err := paymentRepo.GetByIDForUpdate(ctx, paymentID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Payment: Failed to get payment for update during callback", lPaymentID, lNextStatus, zap.Error(err))
		return err
	}

	if payment.Status != StatusPending {
		if payment.Status == StatusExpired && nextStatus == StatusSuccess { // late payment
			log.Warn(
				"Payment: Late payment callback received for expired payment",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				zap.String("current_status", payment.Status),
				lNextStatus,
				zap.Error(ErrPaymentExpired),
			)

			return ErrPaymentExpired
		}

		log.Info(
			"Payment: Callback ignored because payment is already finalized",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
			zap.String("current_status", payment.Status),
			lNextStatus,
		)
		return nil
	}

	var paidAt *time.Time
	if nextStatus == StatusSuccess {
		now := time.Now().UTC()
		paidAt = &now
	}

	if err := paymentRepo.UpdateStatus(ctx, paymentID, nextStatus, paidAt); err != nil {
		log.Error(
			"Payment: Failed to update payment status during callback",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
			lNextStatus,
			zap.Error(err),
		)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	log.Info(
		"Payment: Payment status updated during callback",
		lPaymentID,
		zap.String("order_id", payment.OrderID.String()),
		zap.String("previous_status", payment.Status),
		lNextStatus,
	)

	if afterUpdate != nil {
		if err := afterUpdate(tx, payment); err != nil {
			log.Error(
				"Payment: Failed to run post-callback update",
				lPaymentID,
				zap.String("order_id", payment.OrderID.String()),
				lNextStatus,
				zap.Error(err),
			)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Error(
			"Payment: Failed to commit callback transaction",
			lPaymentID,
			zap.String("order_id", payment.OrderID.String()),
			lNextStatus,
			zap.Error(err),
		)
		return err
	}

	log.Info(
		"Payment: Callback completed",
		lPaymentID,
		zap.String("order_id", payment.OrderID.String()),
		lNextStatus,
	)

	return nil
}

func (s *Service) ExpirePayments(ctx context.Context) ([]ExpiredPayment, error) {
	expiredPayments, err := s.repo.ExpirePayments(ctx)
	if err != nil {
		return nil, err
	}
	return expiredPayments, nil
}

func (s *Service) AddPaymentRecon(ctx context.Context, req PaymentCallbackRequest) error {
	log := helper.LoggerFromCtx(ctx)
	lPaymentID := zap.String("payment_id", req.PaymentID)
	lRequestedStatus := zap.String("requested_status", req.Status)

	log.Info("Payment: Adding payment reconciliation entry", lPaymentID, lRequestedStatus)

	pid, err := uuid.Parse(req.PaymentID)
	if err != nil {
		log.Error("Payment: Failed to parse payment ID for reconciliation", lPaymentID, lRequestedStatus, zap.Error(err))
		return err
	}

	now := time.Now().UTC()

	reconData := &ReconData{
		PaymentID: pid,
		Status:    "PENDING",
		Remark:    "Late Payment",
		CreatedAt: now,
	}

	err = s.repo.AddPaymentRecon(ctx, reconData)
	if err != nil {
		log.Error(
			"Payment: Failed to add payment reconciliation entry",
			lPaymentID,
			lRequestedStatus,
			zap.String("recon_status", reconData.Status),
			zap.String("remark", reconData.Remark),
			zap.Error(err),
		)
		err = errors.New("Something Wrong")
		return err
	}

	log.Info(
		"Payment: Payment reconciliation entry added",
		lPaymentID,
		lRequestedStatus,
		zap.String("recon_status", reconData.Status),
		zap.String("remark", reconData.Remark),
	)

	return nil
}
