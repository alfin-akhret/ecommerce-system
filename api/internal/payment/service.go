package payment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db                 *pgxpool.Pool
	repo               *Repository
	orderStatusUpdater OrderUpdater
}

type OrderUpdater interface {
	UpdateOrderStatusWithTx(ctx context.Context, tx pgx.Tx, orderID string, status string) error
	ConfirmOrderStockWithTx(ctx context.Context, tx pgx.Tx, orderID string) error
}

const (
	StatusPending   = "PENDING"
	StatusSuccess   = "SUCCESS"
	StatusFailed    = "FAILED"
	StatusCancelled = "CANCELLED"
)

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

func (s *Service) SetOrderStatusUpdater(updater OrderUpdater) {
	s.orderStatusUpdater = updater
}

// Create payment (return DTO with payment URL)
func (s *Service) CreatePayment(ctx context.Context, orderID string, amount float64, paymentMethod string) (*CreatePaymentResponse, error) {
	return s.createPayment(ctx, s.repo, orderID, amount, paymentMethod)
}

func (s *Service) CreatePaymentWithTx(ctx context.Context, tx pgx.Tx, orderID string, amount float64, paymentMethod string) (*CreatePaymentResponse, error) {
	return s.createPayment(ctx, s.repo.WithTx(tx), orderID, amount, paymentMethod)
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
	p := &Payment{
		ID:            paymentID,
		OrderID:       orderUUID,
		Amount:        amount,
		Status:        StatusPending,
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}

	paymentURL := "localhost:8080/payment-gateway/pay/" + paymentID.String()

	return &CreatePaymentResponse{
		ID:            paymentID.String(),
		OrderID:       orderID,
		Status:        p.Status,
		Amount:        amount,
		PaymentMethod: paymentMethod,
		PaymentURL:    paymentURL,
		CreatedAt:     &p.CreatedAt,
		UpdatedAt:     &p.UpdatedAt,
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

	now := time.Now()

	if err := paymentRepo.UpdateStatus(ctx, paymentID, StatusSuccess, &now); err != nil {
		return err
	}

	if err := s.orderStatusUpdater.ConfirmOrderStockWithTx(ctx, tx, payment.OrderID.String()); err != nil {
		return err
	}

	if err := s.orderStatusUpdater.UpdateOrderStatusWithTx(ctx, tx, payment.OrderID.String(), "PAID"); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
