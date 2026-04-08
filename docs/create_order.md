# Create Order Checklist

## 1. Input & Request Validation
- User ID valid (uuid parse) ✅
- Cart tidak kosong ✅
- Qty > 0 ✅
- Price snapshot valid (produk masih ada di DB) ✅

## 2. Stock & Inventory
- Stock cukup sebelum reserve ✅
- Reserve stock atomic dengan create order ✅
- Jangan oversell kalau ada race condition ✅ 
- Jika stock kurang, return error → user bisa update cart ✅

## 3. Price & Promo Validation
- Hitung ulang harga terbaru saat CreateOrder ✅
- Snapshot price ke order_items (immutable) ✅
- Apply promo / discount rules (jika ada)
- Pastikan GrandTotal = Total + Shipping – Promo

## 4. Idempotency
- Endpoint harus aman jika user klik “Place Order” berkali-kali ✅
- Bisa pakai idempotency key atau constraint unik per cart ✅
- Jangan double reserve stock / double insert order ✅

## 5. Order Status & Lifecycle
- Set status awal → PENDING ✅
- Payment belum bayar → tetap PENDING ✅
- Payment callback → update → PAID ✅
- Worker untuk expired orders → update → CANCELLED + release stock ✅
- Pastikan worker hanya cancel PENDING, jangan sentuh PAID ✅

## 6. Concurrency & Race Conditions
- Transaction boundary jelas untuk:
- Reserve stock
- Insert order & order_items
- Handle payment callback vs cancellation worker race
- Locking / SELECT ... FOR UPDATE di DB jika perlu

## 7. Error Handling
- Product not found → return error ✅
- Stock insufficient → return error ✅
- DB error → rollback order & stock ✅
- Payment error → user bisa retry 
- Timeout / worker cancel → release stock ✅

## 8. Logging & Monitoring
- Log create order attempt & result
- Log stock reservation / release
- Log payment callback & final status
- Alert / metrics untuk failed/cancelled orders

## 9. Future-proof fields
- Shipping info placeholder
- Payment method placeholder
- Promo / discount info placeholder
- Metadata / notes field untuk ekstensi