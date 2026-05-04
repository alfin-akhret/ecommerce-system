# TODO

### 🔴 HIGH PRIORITY (impact ke money / user trust)

1. **Late Payment Handling (Post-Cancel)** ✅
- Current: manual reconciliation (data di insert ke table payment_recon) ✅
- Problem: user udah bayar tapi order CANCELLED
- Future:
    - auto refund atau
    - auto recreate order (conditional)

2. **Payment Idempotency (Webhook)** ✅
- Pastikan callback payment tidak diproses 2x 
- Tambahin:
    - unique constraint di payment_id (dari gateway)  - implementasi request signature verification, jadi request dipastikan valid dari gateway. ✅
    - ~~atau idempotency table khusus payment:~~ ini diimplementasikan menggunakan pesismistic locking (select... for update) di db karena alurnya update jadi bukan create new record. ✅

3. **Payment vs Worker Race Condition Visibility**
- Saat ini: silent ignore
-  Tambahin:
    - logging event LATE_PAYMENT (ini sudah sekalian dengan semua callback dari payment gateway) ✅

4. Monitoring
    - monitoring / alert : status: ongoing
    - todo: add:
        - request counter ✅
	    - latency histogram
	    - error counter
	    - metrics middleware (global) ✅

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


12. ~~benerin semua log:
banyak juga log.Println, log.Printf, fmt.Println di file lain, ini output plain text, jadikan json pake zap~~

13. log retention policy and implementation
14. instrument semua workers:
    1.  payment worker done.
    - benerin log worker: include trace id dan span id
15. containerize fake-payment-api
16. instrument fake-payment-api


Worker checklist:
* ✅ worker pool
* ✅ graceful shutdown (ctx)
* ✅ panic recovery + restart
* ✅ retry with exponential backoff
* ✅ non-blocking retry
* ✅ context-aware retry
* ✅ DLQ non-blocking
* ✅ logging structured
* ✅ safe logger fallback
* ✅ channel close handling (Jobs)

