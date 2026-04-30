package cart

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

// just in case we need to move the cart to other type of storage
type CartUpdater interface {
	Get(ctx context.Context, ownerID uuid.UUID) (*Cart, error)
	Save(ctx context.Context, cart *Cart) error
	Delete(ctx context.Context, ownerID uuid.UUID) error
}

// cart retention, this temporary, should be move to env vars.
var retention time.Duration = 24 * time.Hour

type CartRepository struct {
	db *redis.Client
}

func CreateNewCartRepository(db *redis.Client) *CartRepository {
	return &CartRepository{
		db: db,
	}
}

func (c *CartRepository) Get(ctx context.Context, ownerID uuid.UUID) (*Cart, error) {
	logger := helper.LoggerFromCtx(ctx)
	tr := otel.Tracer("cart.repository")

	key := "cart:" + ownerID.String()
	ctx, span := tr.Start(ctx, "cart.repository.Get")
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "redis"),
		attribute.String("db.operation", "GET"),
		attribute.String("redis.key", key),
		attribute.String("user.id", ownerID.String()),
	)

	var storedItems string
	var cartNotFound bool
	attempts := 0
	err := helper.Retry(3, 100*time.Millisecond, func() error {
		attempts++
		span.SetAttributes(attribute.Int("redis.get.attempt", attempts))

		var err error
		storedItems, err = c.db.Get(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			cartNotFound = true
			span.SetAttributes(attribute.Bool("cart.found", false))
			logger.Info("CartRepository.Get: cart not found",
				zap.Int("attempt", attempts),
				zap.String("owner_id", ownerID.String()),
				zap.String("redis_key", key))
			return nil
		}

		if err != nil {
			span.RecordError(err)
			logger.Warn("CartRepository.Get: redis get failed",
				zap.Int("attempt", attempts),
				zap.String("owner_id", ownerID.String()),
				zap.String("redis_key", key),
				zap.Error(err))
		}

		return err
	})

	if cartNotFound {
		return nil, ErrCartNotFound
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		logger.Error("CartRepository.Get: redis get failed after retries",
			zap.Int("attempts", attempts),
			zap.String("owner_id", ownerID.String()),
			zap.String("redis_key", key),
			zap.Error(err))
		return nil, err
	}

	cart := &Cart{}
	err = json.Unmarshal([]byte(storedItems), cart)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logger.Error("CartRepository.Get: failed to decode cart",
			zap.String("owner_id", ownerID.String()),
			zap.String("redis_key", key),
			zap.Error(err))
		return nil, err
	}

	span.SetAttributes(attribute.Bool("cart.found", true))
	logger.Info("CartRepository.Get: cart loaded",
		zap.Int("attempts", attempts),
		zap.String("owner_id", ownerID.String()),
		zap.String("redis_key", key))

	return cart, nil
}

func (c *CartRepository) Save(ctx context.Context, cart *Cart) error {
	key := "cart:" + cart.Owner.String()

	payload, err := json.Marshal(cart)
	if err != nil {
		return err
	}

	if err := c.db.Set(ctx, key, payload, retention).Err(); err != nil {
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
