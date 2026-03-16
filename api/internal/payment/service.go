package payment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
const paymentExpiry = 5 * time.Minute

var ErrInvalidAmount = errors.New("amount must be greater than 0")
var ErrInvalidPaymentMethod = errors.New("payment method is required")
var ErrInvalidStatus = errors.New("invalid payment status")
var ErrOrderStatusUpdaterNotSet = errors.New("order status updater is not configured")

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
func (s *Service) CreatePayment(ctx context.Context, orderID string, amount float64, paymentMethod string) (*CreatePaymentResponse, error) {
	return s.createPayment(ctx, s.repo, orderID, amount, paymentMethod)
}

func (s *Service) CreatePaymentWithTx(ctx context.Context, tx pgx.Tx, orderID string, amount float64, paymentMethod string) (*contracts.PaymentCreateResult, error) {
	resp, err := s.createPayment(ctx, s.repo.WithTx(tx), orderID, amount, paymentMethod)
	if err != nil {
		return nil, err
	}

	return &contracts.PaymentCreateResult{
		ID:         resp.ID,
		PaymentURL: resp.PaymentURL,
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

func (s *Service) createPayment(ctx context.Context, repo *Repository, orderID string, amount float64, paymentMethod string) (*CreatePaymentResponse, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if strings.TrimSpace(paymentMethod) == "" {
		return nil, ErrInvalidPaymentMethod
	}

	paymentID := uuid.New()
	now := time.Now()
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}

	expiredAt := time.Now().Add(paymentExpiry)

	p := &Payment{
		ID:            paymentID,
		OrderID:       orderUUID,
		Amount:        amount,
		Status:        StatusPending,
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
		ExpiredAt:     expiredAt,
	}

	if err := repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}

	paymentURL := paymentURL + paymentID.String()

	return &CreatePaymentResponse{
		ID:            paymentID.String(),
		OrderID:       orderID,
		Status:        p.Status,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		PaymentURL:    paymentURL,
		CreatedAt:     &p.CreatedAt,
		UpdatedAt:     &p.UpdatedAt,
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
	return s.processPayment(ctx, paymentID, StatusSuccess, func(tx pgx.Tx, payment *Payment) error {
		if err := s.orderStatusUpdater.ConfirmOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
			return err
		}

		return s.orderStatusUpdater.UpdateOrderStatusWithTx(
			ctx,
			tx,
			payment.OrderID.String(),
			StatusPaid,
		)
	})
}

func (s *Service) ProcessPaymentFailed(ctx context.Context, paymentID string) error {
	return s.processPayment(ctx, paymentID, StatusFailed, func(tx pgx.Tx, payment *Payment) error {
		if err := s.orderStatusUpdater.ReleaseOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
			return err
		}

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
	if s.orderStatusUpdater == nil {
		return ErrOrderStatusUpdaterNotSet
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	paymentRepo := s.repo.WithTx(tx)

	payment, err := paymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return err
	}

	if payment.Status != StatusPending {
		return ErrInvalidStatus
	}

	var paidAt *time.Time
	if nextStatus == StatusSuccess {
		now := time.Now()
		paidAt = &now
	}

	if err := paymentRepo.UpdateStatus(ctx, paymentID, nextStatus, paidAt); err != nil {
		return err
	}

	if afterUpdate != nil {
		if err := afterUpdate(tx, payment); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) HandleCallback(req PaymentCallbackRequest) error {
	ctx := context.Background()

	payment, err := s.repo.GetByID(ctx, req.PaymentID)
	if err != nil {
		return err
	}

	if payment.Status == StatusSuccess || payment.Status == StatusFailed {
		return nil
	}

	status, err := normalizeStatus(req.Status)
	if err != nil {
		return err
	}

	switch status {
	case StatusSuccess:
		return s.ProcessPaymentSuccess(ctx, req.PaymentID)
	case StatusFailed:
		return s.ProcessPaymentFailed(ctx, req.PaymentID)
	default:
		return ErrInvalidStatus
	}
}
