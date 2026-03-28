package cart

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// just in case we need to move the cart to other type of storage
type CartUpdater interface {
	Get(ctx context.Context, ownerID uuid.UUID) (*Cart, error)
	Save(ctx context.Context, cart *Cart) error
	Delete(ctx context.Context, ownerID uuid.UUID) error
}

type CartRepository struct {
	db *redis.Client
}

func CreateNewCartRepository(db *redis.Client) *CartRepository {
	return &CartRepository{
		db: db,
	}
}

func (c *CartRepository) Get(ctx context.Context, ownerID uuid.UUID) (*Cart, error) {
	key := "cart:" + ownerID.String()
	storedItems, err := c.db.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCartNotFound
		}
		return nil, err
	}

	cart := &Cart{}
	err = json.Unmarshal([]byte(storedItems), cart)
	if err != nil {
		return nil, err
	}

	return cart, nil
}

func (c *CartRepository) Save(ctx context.Context, cart *Cart) error {
	key := "cart:" + cart.Owner.String()

	payload, err := json.Marshal(cart)
	if err != nil {
		return err
	}

	if err := c.db.Set(ctx, key, payload, 0).Err(); err != nil {
		return err
	}

	return nil
}

func (c *CartRepository) Delete(ctx context.Context, ownerID uuid.UUID) error {

	if err := c.db.Del(ctx, "cart:"+ownerID.String()).Err(); err != nil {
		return err
	}

	return nil
}
