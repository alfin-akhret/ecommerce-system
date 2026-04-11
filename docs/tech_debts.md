# TODO

### 🔴 HIGH PRIORITY (impact ke money / user trust)

1. **Late Payment Handling (Post-Cancel)**
- Current: manual reconciliation
- Problem: user udah bayar tapi order CANCELLED
- Future:
    - auto refund atau
    - auto recreate order (conditional)

2. **Payment Idempotency (Webhook)** ✅
- Pastikan callback payment tidak diproses 2x 
- Tambahin:
    - unique constraint di payment_id (dari gateway)
    - atau idempotency table khusus payment

3. **Payment vs Worker Race Condition Visibility**
- Saat ini: silent ignore
-  Tambahin:
    - logging event LATE_PAYMENT
    - monitoring / alert

### 🟠 MEDIUM PRIORITY (scalability & ops)
4. **Payment Reconciliation System**
- Table khusus:
    ```
    payment_reconciliation
    order_id
    payment_id
    status (PENDING, REVIEWED, RESOLVED)
    ```
- Flow:
    - late payment → masuk sini
    - diproses manual / semi-auto

5. **Cart Cleanup Reliability**
- Problem:
    - order sukses tapi cart gagal di-clear
- Improvement:
    - async cleanup (worker / outbox)
    - atau mark cart sebagai “checked_out”

7. **Order Expiration Handling Improvement**
- Saat ini: worker-based cancel
- Improvement:
    - tambahin expired_at
    - optimize query:
  ```
  WHERE status = 'PENDING' AND expired_at < now()
  ```
  Setelah itu tambahkan worker untuk update order status menjadi cancel.
  jangan lupa release stock abis itu.

### 🟡 LOW PRIORITY (future-proof / optimization)

8. Move Stock Validation Fully to SQL
- Dari:
    - SELECT FOR UPDATE + logic di app
    - Ke:
    ```
    UPDATE ... WHERE stock - reserved >= qty
    ```

9. Optimistic Locking (version column)
- 	•	Untuk future scaling tanpa heavy locking, tambahkan
    ```
    version INT
    ```

10. Auto Recovery Strategy
- Kalau:
    - payment success tapi order gagal dibuat
- Future:
  - retry / recreate order otomatis

11. Event / Outbox Pattern
- Untuk:
    - stock update
    - payment event
    - order lifecycle
- Biar:
    - reliable async processing
    - ga tergantung sync flow
