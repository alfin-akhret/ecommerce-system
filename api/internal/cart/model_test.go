package cart

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewCart(t *testing.T) {
	t.Run("creates cart with owner and initialized items", func(t *testing.T) {
		ownerID := uuid.New()

		cart, err := NewCart(ownerID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if cart.Owner != ownerID {
			t.Fatalf("expected owner %s, got %s", ownerID, cart.Owner)
		}

		if cart.Items == nil {
			t.Fatal("expected cart items map to be initialized")
		}
	})

	t.Run("returns error when owner is invalid", func(t *testing.T) {
		cart, err := NewCart(uuid.Nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if cart != nil {
			t.Fatalf("expected nil cart, got %+v", cart)
		}
	})
}

func TestCartAddItem(t *testing.T) {
	t.Run("adds new product to cart", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner:     uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.AddItem(productID, 2, 15000)
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

		err := cart.AddItem(productID, 3, 20000)
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

	t.Run("panics when items map is nil", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
		}

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic when adding item to nil map")
			}
		}()

		if err := cart.AddItem(productID, 1, 5000); err != nil {
			t.Fatalf("expected panic before error, got %v", err)
		}
	})

	t.Run("returns error when qty is invalid", func(t *testing.T) {
		productID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.AddItem(productID, 0, 15000)
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

		err := cart.AddItem(uuid.Nil, 1, 15000)
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

		err := cart.AddItem(productID, 1, 0)
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

		err := cart.RemoveItem(productID)
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

		err := cart.RemoveItem(uuid.New())
		if err != ErrItemNotFound {
			t.Fatalf("expected error %v, got %v", ErrItemNotFound, err)
		}
	})

	t.Run("returns error when item id is invalid", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.RemoveItem(uuid.Nil)
		if err != ErrInvalidItemID {
			t.Fatalf("expected error %v, got %v", ErrInvalidItemID, err)
		}
	})

	t.Run("returns error when items map is nil", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
		}

		err := cart.RemoveItem(uuid.New())
		if err != ErrEmptyCart {
			t.Fatalf("expected error %v, got %v", ErrEmptyCart, err)
		}
	})
}

func TestCartUpdateQuantity(t *testing.T) {
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

		err := cart.UpdateQuantity(productID, 5)
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

	t.Run("removes item when qty is zero or less", func(t *testing.T) {
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

		err := cart.UpdateQuantity(productID, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, ok := cart.Items[productID]; ok {
			t.Fatalf("expected product %s to be removed from cart", productID)
		}
	})

	t.Run("returns error when product does not exist", func(t *testing.T) {
		cart := Cart{
			Owner:     uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.UpdateQuantity(uuid.New(), 3)
		if err != ErrItemNotFound {
			t.Fatalf("expected error %v, got %v", ErrItemNotFound, err)
		}
	})

	t.Run("returns error when item id is invalid", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		err := cart.UpdateQuantity(uuid.Nil, 3)
		if err != ErrInvalidItemID {
			t.Fatalf("expected error %v, got %v", ErrInvalidItemID, err)
		}
	})

	t.Run("returns error when items map is nil", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
		}

		err := cart.UpdateQuantity(uuid.New(), 3)
		if err != ErrEmptyCart {
			t.Fatalf("expected error %v, got %v", ErrEmptyCart, err)
		}
	})
}

func TestCartTotal(t *testing.T) {
	t.Run("returns total for all items in cart", func(t *testing.T) {
		firstProductID := uuid.New()
		secondProductID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				firstProductID: {
					ID: firstProductID,
					Qty:       2,
					Price:     15000,
				},
				secondProductID: {
					ID: secondProductID,
					Qty:       3,
					Price:     5000,
				},
			},
		}

		total := cart.Total()

		expected := int64(45000)
		if total != expected {
			t.Fatalf("expected total %d, got %d", expected, total)
		}
	})

	t.Run("returns zero for empty cart", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		total := cart.Total()
		if total != 0 {
			t.Fatalf("expected total 0, got %d", total)
		}
	})
}

func TestCartListItem(t *testing.T) {
	t.Run("returns all items in cart", func(t *testing.T) {
		firstProductID := uuid.New()
		secondProductID := uuid.New()
		cart := Cart{
			Owner: uuid.New(),
			Items: map[uuid.UUID]CartItem{
				firstProductID: {
					ID: firstProductID,
					Qty:       2,
					Price:     15000,
				},
				secondProductID: {
					ID: secondProductID,
					Qty:       1,
					Price:     5000,
				},
			},
		}

		items := cart.ListItem()

		if len(items) != 2 {
			t.Fatalf("expected 2 items, got %d", len(items))
		}

		foundItems := make(map[uuid.UUID]CartItem, len(items))
		for _, item := range items {
			foundItems[item.ID] = item
		}

		if foundItems[firstProductID].Qty != 2 {
			t.Fatalf("expected qty 2 for product %s, got %d", firstProductID, foundItems[firstProductID].Qty)
		}

		if foundItems[secondProductID].Price != 5000 {
			t.Fatalf("expected price 5000 for product %s, got %d", secondProductID, foundItems[secondProductID].Price)
		}
	})

	t.Run("returns empty slice when cart has no items", func(t *testing.T) {
		cart := Cart{
			Owner: uuid.New(),
			Items: make(map[uuid.UUID]CartItem),
		}

		items := cart.ListItem()

		if len(items) != 0 {
			t.Fatalf("expected no items, got %d", len(items))
		}

		if items == nil {
			t.Fatal("expected empty slice, got nil")
		}
	})
}
