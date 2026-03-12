package product

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool // for transactions
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) CreateProduct(ctx context.Context, req CreateProductRequest) (*Product, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	repo := NewProductRepository(tx)

	product := &Product{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}

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
