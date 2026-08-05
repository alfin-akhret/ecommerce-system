# 🛒 Cart
`api/internal/cart`

## 1. Overview
Cart is an aggregate root represents user's shoping cart.
- Store items `(CartItems)` the user wants to buy.
- Keeping domain invariant (rules) such as `qty > 0`, valid price, must have an owner, etc.
- every state mutation must be done through domain methods (`AddItem, RemoveItem, UpdateQuantity`).
- Price only updated with newest price (from db) only when user ADD item to cart. Updating quantity doesn't update the price, therefore the prices in the cart do not reflect changes in the database. The prices in the cart are only a snapshot to give the user an estimate of the total amount they will pay. perubahan testing git2. The actual price is determined at the time of order creation.
  
testing2......

## 2. Model Structure
```
type Cart struct {
	Owner uuid.UUID         // ID user, mandatory
	Items map[uuid.UUID]CartItem // key = productID, value = CartItem
}

type CartItem struct {
	ID    uuid.UUID // product ID
	Qty   int       // quantity, >0
	Price int64     // snapshot price (in cents)
}
```

**Constructor**
```
func NewCart(owner uuid.UUID) (*Cart, error)
```
- ✅ Validation: owner is mandatory
- ✅ initialize map items
- ✅ keeping invariant: cart cannot be created without owner

## 3. Domain Methods

| Method                                      | Description                                          | Behavior / Rules                                                                 |
| ------------------------------------------- | ---------------------------------------------------- | -------------------------------------------------------------------------------- |
| AddItem(id uuid.UUID, qty int, price int64) | Adding new item to cart or merge with existing item. | - qty > 0, price > 0; merge if item already exists or add new item.              |
| RemoveItem(id uuid.UUID)                    | Remove an item form cart.                            | - `Error` if item doesn't exists (`ErrItemNotFound`)                             |
| UpdateQuantity(id uuid.UUID, qty int)       | Update item's qty                                    | - qty <= 0 → auto remove item- Error if item doesn't exists (`ErrItemNotFound)`) |
| ListItems() []CartItem (optional)           | return list (`slice`) of items                       | - Read Only, doesn't modify data.                                                |
| Total() int64 (optional)                    | Calculate total amount of all items                  | Qty * price per item                                                             |

## 4. Invariants
1. Cart must have an owner. `Cart.Owner != uuid.Nil`
2. Qty per item > 0.
3. Price per item > 0.
4. Cart is mutable before `createOrder`
5. No duplicate items were allowed.
6. UpdateQuantity with qty <= 0 --> automatically remove the item from the cart.

## 5. Errors

| Error            | Condition                                       |
| ---------------- | ----------------------------------------------- |
| ErrInvalidQty    | qty <= 0                                        |
| ErrInvalidPrice  | price <= 0                                      |
| ErrItemNotFound  | item id tidak ada saat remove/update            |
| ErrInvalidItemID | uuid.Nil diberikan sebagai productID            |
| ErrEmptyCart     | Items map nil / kosong (opsional, safety check) |

## 6. Usage example
```
ownerID := uuid.New()
cart, _ := NewCart(ownerID)

productID := uuid.New()

// Add 2 units of a product
cart.AddItem(productID, 2, 10000) 

// Update qty
cart.UpdateQuantity(productID, 3)

// Remove item
cart.RemoveItem(productID)

// Get total
total := cart.Total()

// List items
items := cart.ListItems()
```
## Notes / Best Practice
- ⚡ All state mutations should go through domain methods; do not manipulate the map directly.
- 🔄 AddItem and UpdateQuantity already handle merging and auto-removal → simplifies UX.
- 💰 Price is stored as int64 to avoid floating-point rounding issues.
- 🛠 Optional: you can add helper methods for promo/discount later without changing the core domain.
- 🌐 The domain is unaware of the DB/service/API; that is the responsibility of the service layer.
