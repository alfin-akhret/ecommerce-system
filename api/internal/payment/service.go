package payment

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db   *pgxpool.Pool
	repo *Repository
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{
		db:   db,
		repo: NewRepository(db),
	}
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
		return nil, errors.New("amount must be greater than 0")
	}

	paymentID := uuid.New().String()
	now := time.Now()
	p := &Payment{
		ID:            uuid.MustParse(paymentID),
		OrderID:       uuid.MustParse(orderID),
		Amount:        amount,
		Status:        "PENDING",
		PaymentMethod: paymentMethod,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := repo.CreatePayment(ctx, p); err != nil {
		return nil, err
	}

	paymentURL := "localhost:8080/payment-gateway/pay" + paymentID

	return &CreatePaymentResponse{
		ID:            paymentID,
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
	paidAt := paidAtForStatus(status)

	if err := repo.UpdateStatus(ctx, paymentID, status, paidAt); err != nil {
		return nil, err
	}

	return repo.GetByID(ctx, paymentID)
}

func paidAtForStatus(status string) *time.Time {
	if status == "paid" {
		t := time.Now()
		return &t
	}

	return nil
}
