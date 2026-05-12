# Feedback Backend Photobooth

Tanggal analisa: 2026-05-12

## Ringkasan

Backend `BackendPhotobooth` sudah cukup matang dan bukan sekadar skeleton. Struktur project sudah mencakup server Go/Gin, database PostgreSQL dengan GORM, model domain utama, handler API, service layer, middleware, dokumentasi, Docker, seed/migration command, dan folder testing.

Namun masih ada beberapa hal yang perlu dirapikan sebelum backend bisa dianggap siap dipakai penuh: build test saat ini masih gagal, beberapa handler sudah dibuat tetapi belum didaftarkan ke router, dan dokumentasi endpoint belum sepenuhnya konsisten dengan route aktual.

## Yang Sudah Dibuat

### Core backend

- Entry point aplikasi di `main.go`.
- Load konfigurasi dari environment lewat `config/config.go`.
- Init logger Zap lewat `utils/logger.go`.
- Koneksi database PostgreSQL dan auto migration lewat `database/database.go`.
- Graceful shutdown HTTP server.
- Static file serving untuk folder `uploads`.
- Health check endpoint di `/health`.

### Routing API

Router utama ada di `routes/routes.go`.

Prefix API aktual:

```text
/api/v1
```

Route yang sudah aktif antara lain:

- Auth public:
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/logout`
- Profile protected:
  - `GET /api/v1/profile`
  - `PUT /api/v1/profile`
  - `POST /api/v1/profile/change-password`
- Templates public/optional auth:
  - `GET /api/v1/templates`
  - `GET /api/v1/templates/categories`
  - `GET /api/v1/templates/:id`
  - `POST /api/v1/templates/:id/usage`
- Photos protected:
  - `POST /api/v1/photos`
  - `GET /api/v1/photos`
  - `GET /api/v1/photos/:id`
  - `DELETE /api/v1/photos/:id`
  - `GET /api/v1/photos/:id/download`
  - `POST /api/v1/photos/:id/favorite`
  - `POST /api/v1/photos/strip`
- Public photo strip:
  - `POST /api/v1/photos/strip-public`
- Orders:
  - `POST /api/v1/orders/subscription`
  - `GET /api/v1/orders`
  - `GET /api/v1/orders/:id`
  - `POST /api/v1/orders/:id/cancel`
- GoPay QRIS:
  - `POST /api/v1/payment/qris/create`
  - `GET /api/v1/payment/qris/:order_id`
  - `GET /api/v1/payment/qris/:order_id/status`
  - `POST /api/v1/payment/qris/:order_id/cancel`
- Sessions:
  - `POST /api/v1/sessions`
  - `GET /api/v1/sessions`
  - `GET /api/v1/sessions/:session_id`
  - `PUT /api/v1/sessions/:session_id`
  - `POST /api/v1/sessions/:session_id/end`
  - `POST /api/v1/sessions/:session_id/extend`
  - `GET /api/v1/sessions/:session_id/photos`
  - `DELETE /api/v1/sessions/:session_id`
- Promo:
  - `POST /api/v1/promo/validate`
- Admin:
  - Dashboard stats
  - System health
  - User list/detail/status/delete
  - Revenue report
  - User growth analytics
  - Template analytics
  - Template CRUD
  - Promo CRUD dan usage history
- Search:
  - Global search
  - Template search
  - Suggestions
  - Popular searches
- Webhooks:
  - Stripe
  - Midtrans
  - GoPay
- WebSocket:
  - `GET /api/v1/ws`
- Docs:
  - `/docs`
  - `/swagger`
  - `/api-docs`
  - `/api/v1/docs`
  - `/api/v1/docs/swagger.json`

### Model database

Model utama sudah tersedia:

- `User`
- `Template`
- `Photo`
- `Session`
- `Order`
- `Transaction`
- `QRISPayment`
- `PromoCode`
- `PromoUsage`
- `Analytics`
- `DailyStats`
- `AuditLog`
- `TwoFactorAuth`
- `TwoFactorLog`

Auto migration sudah mendaftarkan semua model tersebut.

### Handler

Handler yang sudah ada:

- `AuthHandler`
- `TemplateHandler`
- `TemplateAdminHandler`
- `PhotoHandler`
- `PaymentHandler`
- `GoPayHandler`
- `AdminHandler`
- `SessionHandler`
- `SearchHandler`
- `PromoHandler`
- `DocsHandler`
- `AuditHandler`
- `TwoFAHandler`
- `WebSocketHandler`

Catatan: tidak semua handler yang sudah dibuat sudah aktif di router.

### Services

Service layer yang sudah ada:

- `StorageService` untuk upload, delete, public URL, thumbnail.
- `ImageProcessor` untuk proses foto, filter, watermark, photo strip, optimasi gambar.
- `TemplateProcessor` untuk apply template, photo zones, text element, border, shadow, opacity, thumbnail.
- `GoPayQRISService` untuk create QRIS, check status, verify callback, cancel QRIS, simulate payment.
- `RedisService` dan `CacheService`.
- `EmailService`.
- `WebSocket Hub` dan `NotificationService`.

### Middleware

Middleware yang sudah dibuat:

- JWT auth middleware.
- Optional auth middleware.
- Admin middleware.
- Recovery middleware.
- Error handler middleware.
- Security headers middleware.
- Rate limit middleware.
- Zap request logger.
- Audit middleware.

### Dokumentasi dan tooling

- `README.md` sudah berisi fitur, setup, env, schema, endpoint, dan credential seed.
- `docs/swagger.json` tersedia.
- `Dockerfile` tersedia.
- `docker-compose.yml` tersedia.
- `Makefile` tersedia.
- Command migration dan seed tersedia di `cmd/migrate` dan `cmd/seed`.
- Script seed tambahan ada di `scripts/seed.go`.
- Folder tests sudah tersedia.

## Yang Masih Kurang / Perlu Diperbaiki

### 1. Build/test masih gagal

Saat menjalankan:

```bash
go test ./...
```

hasilnya gagal build karena:

```text
handlers/promo_handler.go:7:2: "backendphotobooth/utils" imported and not used
```

Perbaikannya sederhana: hapus import `backendphotobooth/utils` jika memang tidak dipakai, atau gunakan logger dari `utils` jika memang dibutuhkan di handler promo.

### 2. Ada handler yang belum tersambung ke router

Beberapa fitur sudah dibuat di handler, tetapi belum terlihat aktif di `routes/routes.go`:

- `TwoFAHandler`
  - Setup 2FA
  - Verify and enable 2FA
  - Disable 2FA
  - Verify 2FA
  - Get 2FA status
  - Regenerate backup codes
- `AuditHandler`
  - Get audit logs
  - User audit trail
  - Resource audit trail
  - Audit stats
  - Export audit logs
- Beberapa fungsi `TemplateAdminHandler`:
  - Toggle template status
  - Toggle template featured
  - Duplicate template
  - Template analytics versi handler template admin
- Beberapa fungsi `SearchHandler`:
  - Search photos
  - Search users
- Beberapa fungsi `AdminHandler`:
  - Export users

Artinya fitur-fitur tersebut sudah dibuat secara kode, tetapi belum bisa dipanggil dari API utama kecuali nanti route-nya ditambahkan.

### 3. README belum konsisten dengan route aktual

README mencantumkan beberapa endpoint dengan prefix:

```text
/api/...
```

Sedangkan router aktual menggunakan:

```text
/api/v1/...
```

Ini perlu disamakan agar frontend atau developer lain tidak salah target endpoint.

### 4. Audit middleware belum terlihat dipakai secara global

`middleware/audit.go` sudah ada, tetapi di router utama belum terlihat dipasang sebagai middleware global atau untuk route admin/protected tertentu.

Perlu diputuskan:

- audit semua request penting,
- audit hanya perubahan data,
- atau audit khusus admin/payment/security action.

### 5. Redis service sudah ada tetapi belum tampak aktif di main flow

`RedisService` dan `CacheService` sudah dibuat, tapi `main.go` belum menginisialisasi Redis/cache dan belum mengoper ke handler/service lain.

Jika caching memang masuk scope, perlu dihubungkan ke:

- template list cache,
- user/session cache,
- rate limit berbasis Redis,
- stats/dashboard cache.

### 6. Email service sudah ada tetapi belum tampak dipakai

`EmailService` sudah mendukung welcome email, verification, reset password, order confirmation, reminder, dan photo ready notification.

Namun belum terlihat tersambung ke flow:

- register,
- email verification,
- forgot password,
- order paid,
- photo processed.

### 7. WebSocket handler ada, tetapi router memakai service langsung

Ada `handlers/websocket_handler.go`, tetapi route `/api/v1/ws` langsung memanggil:

```go
services.ServeWs(wsHub, c.Writer, c.Request)
```

Ini tidak salah, tapi membuat `WebSocketHandler` belum dimanfaatkan. Perlu dipilih salah satu pola agar konsisten.

### 8. Payment provider masih campuran

Ada handler webhook untuk Stripe dan Midtrans, serta GoPay QRIS yang lebih lengkap.

Perlu diperjelas strategi payment:

- Apakah GoPay QRIS menjadi payment utama?
- Apakah Stripe/Midtrans hanya placeholder?
- Apakah frontend harus memakai semua provider atau hanya QRIS?

### 9. Order number berpotensi collision

`GenerateOrderNumber()` memakai timestamp sampai detik:

```go
ORD-YYYYMMDD-HHMMSS
```

Jika ada lebih dari satu order dalam detik yang sama, ada risiko bentrok pada unique index. Sebaiknya ditambah random suffix atau UUID pendek.

### 10. Test coverage sudah ada, tapi belum bisa diverifikasi

Folder `tests` sudah berisi:

- admin test
- auth test
- integration test
- payment test
- template test
- session test
- promo test
- photo test

Namun karena build masih gagal, test suite belum bisa dijadikan indikator kualitas sampai compile error dibereskan.

### 11. Ada perubahan git yang belum dicommit

Status terakhir di folder backend menunjukkan:

```text
M handlers/promo_handler.go
M routes/routes.go
```

Sebelum perubahan besar berikutnya, sebaiknya dicek apakah ini perubahan sengaja dan perlu disimpan.

## Prioritas Perbaikan

### Prioritas 1 - Bikin backend build hijau

1. Hapus unused import di `handlers/promo_handler.go`.
2. Jalankan `go test ./...`.
3. Perbaiki error compile berikutnya jika muncul.

### Prioritas 2 - Sinkronisasi route dan dokumentasi

1. Update README agar semua endpoint memakai `/api/v1`.
2. Pastikan `docs/swagger.json` sesuai router aktual.
3. Tambahkan route untuk handler yang sudah ada tetapi belum aktif.

### Prioritas 3 - Aktifkan fitur yang sudah setengah jadi

1. Daftarkan route 2FA.
2. Daftarkan route audit log untuk admin.
3. Daftarkan route export users.
4. Daftarkan route search photos/users sesuai kebutuhan role.
5. Daftarkan route toggle/duplicate template admin.

### Prioritas 4 - Integrasi service pendukung

1. Hubungkan Redis/cache jika memang akan digunakan.
2. Hubungkan EmailService ke flow auth, payment, dan photo processing.
3. Rapikan WebSocket agar memakai handler atau service secara konsisten.

### Prioritas 5 - Hardening sebelum production

1. Perkuat order number agar tidak collision.
2. Review security untuk webhook signature, file upload, dan CORS.
3. Pastikan rate limit production-ready.
4. Pastikan `.env` tidak ikut commit.
5. Tambahkan test untuk route yang baru diaktifkan.

## Kesimpulan

Backend sudah memiliki fondasi dan fitur besar yang lengkap: auth, template, photo processing, session, order/payment QRIS, promo, admin, search, docs, middleware, dan test structure.

Yang paling penting sekarang adalah merapikan koneksi antar bagian: build harus hijau, route harus mengekspos handler yang sudah dibuat, dokumentasi harus mengikuti route aktual, lalu service pendukung seperti Redis, email, audit, dan 2FA perlu diaktifkan secara konsisten.
