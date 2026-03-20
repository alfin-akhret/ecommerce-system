package product

import (
	"context"
	"errors"

	"github.com/alfin-akhret/ecommerce-system/internal/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotEnoughStock = errors.New("not enough stock")
var ErrInvalidReserved = errors.New("invalid reserved quantity")

type Service struct {
	db   *pgxpool.Pool // for queries that require transactions, we create a new repository with the transaction as DBTX
	repo *Repository   // for simple queries that don't require transactions, we can use the repository with the main DB connection
}

func NewService(db *pgxpool.Pool) *Service {

	repo := NewProductRepository(db)

	return &Service{
		db:   db,
		repo: repo,
	}
}

func (s *Service) CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repo := s.repo.WithTx(tx)

	product := toProduct(req)

	if err := repo.CreateProduct(ctx, product); err != nil {
		return nil, err
	}

	inv := &Inventory{
		ProductID: product.ID,
		Stock:     req.Stock,
		Reserved:  0,
	}

	if err := repo.CreateInventory(ctx, inv); err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return product, nil

}

func (s *Service) ListProducts(ctx context.Context) ([]ProductListItem, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) GetProductByID(ctx context.Context, id string) (*ProductDetailResponse, error) {
	return s.repo.GetProductByID(ctx, id)
}

func (s *Service) GetProductByIDWithTx(ctx context.Context, tx pgx.Tx, id string) (contracts.ProductView, error) {
	repo := s.repo.WithTx(tx)
	detail, err := repo.GetProductByID(ctx, id)
	if err != nil {
		return nil, err
	}

	productID, err := uuid.Parse(detail.ID)
	if err != nil {
		return nil, err
	}

	return &Product{
		ID:          productID,
		Name:        detail.Name,
		Description: detail.Description,
		Price:       detail.Price,
	}, nil
}

func (s *Service) UpdateStock(ctx context.Context, productID string, qty int) error {
	return s.repo.UpdateStock(ctx, productID, qty)
}

func (s *Service) ReserveStock(ctx context.Context, productID string, qty int) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.ReserveStockWithTx(ctx, tx, productID, qty); err != nil {
		return err
	}

	return tx.Commit(ctx)

}

func (s *Service) ReserveStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	repo := s.repo.WithTx(tx)
	inv, err := repo.GetInventoryForUpdate(ctx, productID)
	if err != nil {
		return err
	}

	available := inv.Stock - inv.Reserved

	if available < qty {
		return ErrNotEnoughStock
	}

	if err := repo.UpdateReserved(ctx, productID, qty); err != nil {
		return err
	}

	return nil
}

func (s *Service) ReleaseStock(ctx context.Context, productID string, qty int) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.ReleaseStockWithTx(ctx, tx, productID, qty); err != nil {
		return err
	}

	return tx.Commit(ctx)

}

func (s *Service) ReleaseStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	repo := s.repo.WithTx(tx)

	inv, err := repo.GetInventoryForUpdate(ctx, productID)
	if err != nil {
		return err
	}

	if inv.Reserved < qty {
		return ErrInvalidReserved
	}

	if err := repo.ReleasedReserved(ctx, productID, qty); err != nil {
		return err
	}

	return nil
}

func (s *Service) ConfirmStock(ctx context.Context, productID string, qty int) error {

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.ConfirmStockWithTx(ctx, tx, productID, qty); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) ConfirmStockWithTx(ctx context.Context, tx pgx.Tx, productID string, qty int) error {
	repo := s.repo.WithTx(tx)

	inv, err := repo.GetInventoryForUpdate(ctx, productID)
	if err != nil {
		return err
	}

	if inv.Reserved < qty {
		return ErrInvalidReserved
	}

	if err := repo.ConfirmStock(ctx, productID, qty); err != nil {
		return err
	}

	return nil
}
