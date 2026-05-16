Perubahan utama:


redis_queue.go: enqueue sekarang atomik pakai Lua SET NX + LPUSH, jadi dedup key tidak dibuat kalau push gagal.

Job yang sedang diproses sekarang punya lease di queue:processing:deadlines plus heartbeat, jadi job lambat tidak direcovery dobel selama worker masih hidup.

Scheduler delayed job sekarang atomik ZREM -> LPUSH, jadi multi-worker/process tidak mindahin retry yang sama berkali-kali.

Recovery stale job sekarang requeue hanya kalau berhasil LREM dari processing, jadi tidak gampang bikin duplikat.

Handler yang timeout langsung masuk DLQ, supaya job tidak “ga kelar-kelar” atau retry sambil handler lama masih menggantung.

job.go: tambah ID; kalau kosong, queue bikin fingerprint dari Type + Payload buat dedup otomatis.