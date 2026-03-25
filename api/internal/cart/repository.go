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
	storedItems, err := c.db.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(storedItems) == 0 {
		return nil, redis.Nil
	}

	cart := &Cart{
		Owner: ownerID,
		Items: make(map[uuid.UUID]CartItem, len(storedItems)),
	}

	for rawProductID, rawItem := range storedItems {
		productID, err := uuid.Parse(rawProductID)
		if err != nil {
			return nil, err
		}

		var item CartItem
		if err := json.Unmarshal([]byte(rawItem), &item); err != nil {
			return nil, err
		}

		cart.Items[productID] = item
	}

	return cart, nil
}

func (c *CartRepository) Save(ctx context.Context, cart *Cart) error {
	if cart == nil {
		return errors.New("cart is nil")
	}

	key := "cart:" + cart.Owner.String()

	if len(cart.Items) == 0 {
		return c.db.Del(ctx, key).Err()
	}

	serializedItems := make(map[string]string, len(cart.Items))
	for productID, item := range cart.Items {
		rawItem, err := json.Marshal(item)
		if err != nil {
			return err
		}

		serializedItems[productID.String()] = string(rawItem)
	}

	pipe := c.db.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, serializedItems)
	if _, err := pipe.Exec(ctx); err != nil {
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
