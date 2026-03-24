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
	Price int64
}

func NewCart(owner uuid.UUID) (*Cart, error) {
	if owner == uuid.Nil {
		return nil, errors.New("cart must have an owner")
	}

	return &Cart{
		Owner: owner,
		Items: make(map[uuid.UUID]CartItem),
	}, nil
}

var ErrInvalidQty = errors.New("quantity must be greater than 0")
var ErrInvalidPrice = errors.New("price must be greater than 0")
var ErrEmptyCart = errors.New("cart is empty")
var ErrItemNotFound = errors.New("item not found")
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

func ensurePrice(price int64) func() error {
	return func() error {
		if price <= 0 {
			return ErrInvalidPrice
		}
		return nil
	}
}

func (c *Cart) AddItem(id uuid.UUID, qty int, price int64) error {

	if err := validate(ensureId(id), ensurePrice(price), ensureQty(qty)); err != nil {
		return err
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

func (c *Cart) RemoveItem(id uuid.UUID) error {
	if err := validate(ensureId(id), ensureCartItem(c)); err != nil {
		return err
	}

	_, ok := c.Items[id]
	if ok {
		delete(c.Items, id)
		return nil
	}
	return ErrItemNotFound
}

func (c *Cart) UpdateQuantity(id uuid.UUID, qty int) error {

	if err := validate(ensureId(id), ensureCartItem(c)); err != nil {
		return err
	}

	item, ok := c.Items[id]
	if ok {
		if qty <= 0 {
			delete(c.Items, id)
			return nil
		}
		item.Qty = qty
		c.Items[id] = item
		return nil
	}

	return ErrItemNotFound

}

func (c *Cart) Total() int64 {
	total := int64(0)
	for _, item := range c.Items {
		total += item.Price * int64(item.Qty)
	}
	return total
}

func (c *Cart) ListItem() []CartItem {
	items := make([]CartItem, 0, len(c.Items))
	for _, item := range c.Items {
		items = append(items, item)
	}
	return items
}
