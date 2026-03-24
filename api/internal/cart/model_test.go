package cart

import (
	"testing"

	"github.com/google/uuid"
)

func TestCartAddItem(t *testing.T) {
	t.Run("adds new product to cart", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner:     uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.addItem(productID, 2, 15000)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		item, ok := cart.Items[productID]
		if !ok {
			t.Fatalf("expected product %s to exist in cart", productID)
		}

		if item.ID != productID {
			t.Fatalf("expected product id %s, got %s", productID, item.ID)
		}

		if item.Qty != 2 {
			t.Fatalf("expected qty 2, got %d", item.Qty)
		}

		if item.Price != 15000 {
			t.Fatalf("expected price 15000, got %v", item.Price)
		}
	})

	t.Run("increments qty when product already exists", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				productID: {
					ID: productID,
					Qty:       1,
					Price:     15000,
				},
			},
		}

		err := cart.addItem(productID, 3, 20000)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		item := cart.Items[productID]
		if item.Qty != 4 {
			t.Fatalf("expected qty 4, got %d", item.Qty)
		}

		if item.Price != 15000 {
			t.Fatalf("expected price to stay 15000, got %v", item.Price)
		}
	})

	t.Run("initializes cart items map when nil", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
		}

		err := cart.addItem(productID, 1, 5000)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cart.Items == nil {
			t.Fatal("expected cart items map to be initialized")
		}

		if cart.Items[productID].Qty != 1 {
			t.Fatalf("expected qty 1, got %d", cart.Items[productID].Qty)
		}
	})

	t.Run("returns error when qty is invalid", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.addItem(productID, 0, 15000)
		if err != ErrInvalidQty {
			t.Fatalf("expected error %v, got %v", ErrInvalidQty, err)
		}

		if len(cart.Items) != 0 {
			t.Fatalf("expected cart to remain empty, got %d item(s)", len(cart.Items))
		}
	})

	t.Run("returns error when item id is invalid", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.addItem(uuid.Nil, 1, 15000)
		if err != ErrInvalidItemID {
			t.Fatalf("expected error %v, got %v", ErrInvalidItemID, err)
		}

		if len(cart.Items) != 0 {
			t.Fatalf("expected cart to remain empty, got %d item(s)", len(cart.Items))
		}
	})

	t.Run("returns error when price is invalid", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.addItem(productID, 1, 0)
		if err != ErrInvalidPrice {
			t.Fatalf("expected error %v, got %v", ErrInvalidPrice, err)
		}

		if len(cart.Items) != 0 {
			t.Fatalf("expected cart to remain empty, got %d item(s)", len(cart.Items))
		}
	})
}

func TestCartRemoveItem(t *testing.T) {
	t.Run("removes existing product from cart", func(t *testing.T) {
		productID := uuid.New()
		anotherProductID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				productID: {
					ID: productID,
					Qty:       2,
					Price:     15000,
				},
				anotherProductID: {
					ID: anotherProductID,
					Qty:       1,
					Price:     5000,
				},
			},
		}

		err := cart.removeItem(productID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, ok := cart.Items[productID]; ok {
			t.Fatalf("expected product %s to be removed", productID)
		}

		if _, ok := cart.Items[anotherProductID]; !ok {
			t.Fatalf("expected product %s to remain in cart", anotherProductID)
		}
	})

	t.Run("returns error when product does not exist", func(t *testing.T) {
		cart := Cart{
			Owner:     uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.removeItem(uuid.New())
		if err != ErrEmptyCart {
			t.Fatalf("expected error %v, got %v", ErrEmptyCart, err)
		}
	})

	t.Run("returns error when item id is invalid", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.removeItem(uuid.Nil)
		if err != ErrInvalidItemID {
			t.Fatalf("expected error %v, got %v", ErrInvalidItemID, err)
		}
	})
}

func TestCartChangeQuantity(t *testing.T) {
	t.Run("changes qty for existing product", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				productID: {
					ID: productID,
					Qty:       2,
					Price:     15000,
				},
			},
		}

		err := cart.changeQuantity(productID, 5)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		item := cart.Items[productID]
		if item.Qty != 5 {
			t.Fatalf("expected qty 5, got %d", item.Qty)
		}

		if item.Price != 15000 {
			t.Fatalf("expected price to remain 15000, got %v", item.Price)
		}
	})

	t.Run("returns error when qty is invalid", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				productID: {
					ID: productID,
					Qty:       2,
					Price:     15000,
				},
			},
		}

		err := cart.changeQuantity(productID, 0)
		if err != ErrInvalidQty {
			t.Fatalf("expected error %v, got %v", ErrInvalidQty, err)
		}

		if cart.Items[productID].Qty != 2 {
			t.Fatalf("expected qty to remain 2, got %d", cart.Items[productID].Qty)
		}
	})

	t.Run("returns error when product does not exist", func(t *testing.T) {
		cart := Cart{
			Owner:     uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.changeQuantity(uuid.New(), 3)
		if err != ErrEmptyCart {
			t.Fatalf("expected error %v, got %v", ErrEmptyCart, err)
		}
	})

	t.Run("returns error when item id is invalid", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.changeQuantity(uuid.Nil, 3)
		if err != ErrInvalidItemID {
			t.Fatalf("expected error %v, got %v", ErrInvalidItemID, err)
		}
	})
}
