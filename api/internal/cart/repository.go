package cart

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// just in case we need to move the cart to other type of storage
type CartUpdater interface {
	Get(ctx context.Context, userID uuid.UUID) (*Cart, error)
	Save(ctx context.Context, cart *Cart) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

type CartRepository struct {
	db *redis.Client
}

func CreateNewCartRepository(db *redis.Client) *CartRepository {
	return &CartRepository{
		db: db,
	}
}

func (c *CartRepository) Get(ctx context.Context, userID uuid.UUID) (*Cart, error) {
	// not implemented yet
	return nil, nil
}

func (c *CartRepository) Save(ctx context.Context, cart *Cart) error {
	ownerID := cart.Owner.String()

	for productID, val := range cart.Items {
		payload, err := json.Marshal(val)
		if err != nil {
			return err
		}

		if err := c.db.HSet(ctx, "cart:"+ownerID, productID.String(), payload).Err(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CartRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	// not implemented yet
	return nil
}
