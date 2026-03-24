package cart

import (
	"errors"

	"github.com/google/uuid"
)

type Cart struct {
	Owner uuid.UUID
	Items map[uuid.UUID]CartItem
}

type CartItem struct {
	ID    uuid.UUID
	Qty   int
	Price float64
}

var ErrInvalidQty = errors.New("quantity must be greater than 0")
var ErrInvalidPrice = errors.New("price must be greater than 0")
var ErrEmptyCart = errors.New("cart is empty")
var ErrInvalidItemID = errors.New("invalid item id")

// validator helper
func validate(checks ...func() error) error {
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func ensureId(id uuid.UUID) func() error {
	return func() error {
		if id == uuid.Nil {
			return ErrInvalidItemID
		}
		return nil
	}
}

func ensureQty(qty int) func() error {
	return func() error {
		if qty <= 0 {
			return ErrInvalidQty
		}
		return nil
	}
}

func ensureCartItem(c *Cart) func() error {
	return func() error {
		if c.Items == nil {
			return ErrEmptyCart
		}
		return nil
	}
}

func ensurePrice(price float64) func() error {
	return func() error {
		if price <= 0 {
			return ErrInvalidPrice
		}
		return nil
	}
}

func (c *Cart) addItem(id uuid.UUID, qty int, price float64) error {

	if err := validate(ensureId(id), ensurePrice(price), ensureQty(qty)); err != nil {
		return err
	}

	if c.Items == nil {
		c.Items = make(map[uuid.UUID]CartItem)
	}

	item, ok := c.Items[id]
	if ok {
		item.Qty += qty
	} else {
		item = CartItem{
			ID:    id,
			Qty:   qty,
			Price: price,
		}
	}

	c.Items[id] = item

	return nil

}

func (c *Cart) removeItem(id uuid.UUID) error {
	if err := validate(ensureId(id), ensureCartItem(c)); err != nil {
		return err
	}

	_, ok := c.Items[id]
	if ok {
		delete(c.Items, id)
		return nil
	}
	return ErrEmptyCart
}

func (c *Cart) changeQuantity(id uuid.UUID, qty int) error {

	if err := validate(ensureId(id), ensureCartItem(c), ensureQty(qty)); err != nil {
		return err
	}

	item, ok := c.Items[id]
	if ok {
		item.Qty = qty
		c.Items[id] = item
		return nil
	}

	return ErrEmptyCart

}
