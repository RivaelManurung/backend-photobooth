# 📸 Photo Booth Backend API

Backend API untuk aplikasi Photo Booth dengan fitur lengkap termasuk autentikasi, template management, payment gateway GoPay QRIS, dan banyak lagi.

## 🚀 Fitur Utama

### Core Features
- ✅ **Authentication & Authorization** - JWT-based auth dengan role management (admin/user)
- ✅ **Template Management** - CRUD template foto dengan berbagai layout (single, strip, grid, collage)
- ✅ **Photo Processing** - Upload, edit, dan download foto dengan berbagai filter
- ✅ **Session Management** - Kelola sesi photo booth dengan tracking lengkap
- ✅ **Order & Payment** - Sistem order dengan integrasi payment gateway

### Payment Gateway
- ✅ **GoPay QRIS** - Integrasi lengkap dengan GoPay QRIS sebagai payment method utama
  - Generate QRIS code
  - Real-time payment status checking
  - Webhook callback untuk notifikasi pembayaran
  - HMAC-SHA256 signature verification

### Advanced Features
- ✅ **Two-Factor Authentication (2FA)** - TOTP-based 2FA dengan QR code dan backup codes
- ✅ **Promo Code System** - Sistem promo dengan berbagai tipe diskon (percentage/fixed)
- ✅ **Analytics & Reporting** - Tracking event dan daily statistics
- ✅ **Audit Logs** - Complete audit trail untuk semua aktivitas
- ✅ **WebSocket** - Real-time notifications untuk payment status
- ✅ **Redis Caching** - Caching untuk performa optimal
- ✅ **Rate Limiting** - Protection dari abuse dengan rate limiter
- ✅ **Search & Filter** - Advanced search dengan pagination

## 📋 Tech Stack

- **Framework**: Gin (Go Web Framework)
- **Database**: PostgreSQL 14+
- **Cache**: Redis
- **Authentication**: JWT
- **Payment**: GoPay QRIS
- **Image Processing**: Go Image Libraries
- **Testing**: Testify
- **Documentation**: Swagger (optional)

## 🛠️ Prerequisites

Pastikan sudah terinstall:
- Go 1.21 atau lebih baru
- PostgreSQL 14+
- Redis (optional, untuk caching)
- Git

## ⚙️ Installation & Setup

### 1. Clone Repository

```bash
git clone <repository-url>
cd BackendPhotobooth
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Setup Environment Variables

Copy file `.env.example` ke `.env` dan sesuaikan konfigurasi:

```bash
cp .env.example .env
```

Edit file `.env`:

```env
# Server Configuration
PORT=8080
ENV=development

# Database Configuration
DB_HOST=localhost
DB_PORT=5433
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=photobooth
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production

# Redis Configuration (Optional)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Storage Configuration
STORAGE_TYPE=local
STORAGE_PATH=./uploads

# GoPay QRIS Configuration
GOPAY_MERCHANT_ID=your_merchant_id
GOPAY_TERMINAL_ID=your_terminal_id
GOPAY_SECRET_KEY=your_secret_key
GOPAY_API_URL=https://api.gopay.com/v1

# Email Configuration (Optional)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM=noreply@photobooth.com
```

### 4. Setup Database

#### Windows (PowerShell)

```powershell
# Complete setup (migration + seeding)
.\setup.ps1

# Atau jalankan satu per satu:
.\migrate.ps1  # Migrasi database
.\seed.ps1     # Seed data awal
```

#### Linux/Mac

```bash
# Complete setup
make setup

# Atau jalankan satu per satu:
make migrate  # Migrasi database
make seed     # Seed data awal
```

#### Manual Setup

```bash
# Migrasi database
go run cmd/migrate/main.go

# Seed data awal
go run cmd/seed/main.go
```

### 5. Run Server

```bash
# Development mode
go run main.go

# Atau dengan hot reload (jika sudah install air)
air

# Production mode
go build -o photobooth-api
./photobooth-api
```

Server akan berjalan di `http://localhost:8080`

## 📊 Database Schema

### Tables (14 total)

1. **users** - User accounts dengan subscription management
2. **templates** - Photo templates dengan berbagai layout
3. **photos** - Uploaded photos dengan metadata
4. **sessions** - Photo booth sessions
5. **orders** - Order management
6. **transactions** - Payment transactions
7. **qris_payments** - GoPay QRIS payment records
8. **promo_codes** - Promo code management
9. **promo_usages** - Promo usage tracking
10. **analytics** - Event analytics
11. **daily_stats** - Daily statistics aggregation
12. **audit_logs** - Complete audit trail
13. **two_factor_auths** - 2FA settings per user
14. **two_factor_logs** - 2FA activity logs

## 🔐 Default Credentials

Setelah seeding, gunakan credentials berikut untuk login:

**Admin Account:**
- Email: `admin@photobooth.com`
- Password: `admin123`
- Role: `admin`
- Subscription: `premium` (valid 1 tahun)

## 📝 API Endpoints

### Authentication
```
POST   /api/v1/auth/register          - Register user baru
POST   /api/v1/auth/login             - Login user
POST   /api/v1/auth/refresh           - Refresh JWT token
GET    /api/v1/auth/me                - Get current user info
PUT    /api/v1/auth/profile           - Update profile
PUT    /api/v1/auth/password          - Change password
```

### Templates
```
GET    /api/v1/templates              - List semua templates
GET    /api/v1/templates/:id          - Get template detail
POST   /api/v1/templates              - Create template (admin)
PUT    /api/v1/templates/:id          - Update template (admin)
DELETE /api/v1/templates/:id          - Delete template (admin)
GET    /api/v1/templates/featured     - Get featured templates
```

### Photos
```
GET    /api/v1/photos                 - List user photos
GET    /api/v1/photos/:id             - Get photo detail
POST   /api/v1/photos                 - Upload photo
PUT    /api/v1/photos/:id             - Update photo
DELETE /api/v1/photos/:id             - Delete photo
GET    /api/v1/photos/:id/download    - Download photo
```

### Sessions
```
GET    /api/v1/sessions               - List sessions
GET    /api/v1/sessions/:id           - Get session detail
POST   /api/v1/sessions               - Create session
PUT    /api/v1/sessions/:id           - Update session
DELETE /api/v1/sessions/:id           - Delete session
POST   /api/v1/sessions/:id/complete  - Complete session
```

### Orders & Payments
```
GET    /api/v1/orders                 - List orders
GET    /api/v1/orders/:id             - Get order detail
POST   /api/v1/orders                 - Create order
PUT    /api/v1/orders/:id             - Update order
POST   /api/v1/orders/:id/cancel      - Cancel order
```

### GoPay QRIS Payment
```
POST   /api/v1/gopay/qris/create      - Create QRIS payment
GET    /api/v1/gopay/qris/:id         - Get QRIS payment detail
GET    /api/v1/gopay/qris/:id/status  - Check payment status
POST   /api/v1/gopay/qris/:id/cancel  - Cancel payment
POST   /api/v1/gopay/webhook          - Webhook callback (dari GoPay)
```

### Promo Codes
```
GET    /api/v1/promo-codes/validate   - Validate promo code
POST   /api/v1/promo-codes/apply      - Apply promo code

# Admin only
GET    /api/v1/admin/promo-codes      - List all promo codes
POST   /api/v1/admin/promo-codes      - Create promo code
PUT    /api/v1/admin/promo-codes/:id  - Update promo code
DELETE /api/v1/admin/promo-codes/:id  - Delete promo code
```

### Two-Factor Authentication
```
POST   /api/v1/2fa/setup              - Setup 2FA
POST   /api/v1/2fa/verify             - Verify 2FA code
POST   /api/v1/2fa/disable            - Disable 2FA
GET    /api/v1/2fa/backup-codes       - Get backup codes
POST   /api/v1/2fa/regenerate-backup  - Regenerate backup codes
```

### Admin - Analytics
```
GET    /api/v1/admin/analytics/overview    - Dashboard overview
GET    /api/v1/admin/analytics/daily       - Daily statistics
GET    /api/v1/admin/analytics/users       - User analytics
GET    /api/v1/admin/analytics/revenue     - Revenue analytics
```

### Admin - Audit Logs
```
GET    /api/v1/admin/audit-logs            - List audit logs
GET    /api/v1/admin/audit-logs/:id        - Get audit log detail
```

### Search
```
GET    /api/v1/search/templates       - Search templates
GET    /api/v1/search/photos          - Search photos
GET    /api/v1/search/users           - Search users (admin)
```

### WebSocket
```
WS     /ws                         - WebSocket connection untuk real-time notifications
```

## 💳 GoPay QRIS Integration

### Flow Pembayaran

1. **Create QRIS Payment**
   ```bash
   POST /api/v1/gopay/qris/create
   {
     "order_id": 1,
     "amount": 50000,
     "customer_name": "John Doe",
     "customer_phone": "081234567890",
     "customer_email": "john@example.com"
   }
   ```

2. **Response dengan QRIS Code**
   ```json
   {
     "id": 1,
     "qris_string": "00020101021226...",
     "qris_image_url": "/uploads/qris/xxx.png",
     "amount": 50000,
     "status": "pending",
     "expires_at": "2024-01-01T12:30:00Z"
   }
   ```

3. **Customer Scan QRIS** - Customer scan QR code dengan aplikasi GoPay

4. **Check Payment Status**
   ```bash
   GET /api/v1/gopay/qris/:id/status
   ```

5. **Webhook Notification** - GoPay akan kirim notifikasi ke webhook endpoint
   ```bash
   POST /api/v1/gopay/webhook
   ```

6. **WebSocket Notification** - Frontend akan menerima real-time notification via WebSocket

### Testing GoPay (Development)

Untuk testing di development, gunakan GoPay Sandbox:
- Merchant ID: Dari GoPay Developer Portal
- Terminal ID: Dari GoPay Developer Portal
- Secret Key: Dari GoPay Developer Portal
- API URL: `https://api-sandbox.gopay.com/v1`

## 🧪 Testing

### Run All Tests

```bash
# Run semua tests
go test ./...

# Run dengan coverage
go test -cover ./...

# Run dengan verbose
go test -v ./...

# Run specific test file
go test ./tests/auth_test.go
```

### Test Files

- `tests/auth_test.go` - Authentication tests
- `tests/template_test.go` - Template management tests
- `tests/integration_test.go` - Integration tests

## 🎯 Promo Codes (Default)

Setelah seeding, tersedia 3 promo codes:

1. **WELCOME10**
   - Type: Percentage
   - Discount: 10%
   - Max Discount: Rp 50,000
   - Valid: 3 bulan
   - Applicable: basic, premium plans

2. **FIRST50**
   - Type: Fixed
   - Discount: Rp 50,000
   - Min Purchase: Rp 100,000
   - Valid: 1 bulan
   - First-time users only
   - Applicable: premium plan

3. **YEARLY20**
   - Type: Percentage
   - Discount: 20%
   - Max Discount: Rp 200,000
   - Valid: 1 tahun
   - Unlimited uses
   - Applicable: premium plan

## 📦 Sample Templates

8 sample templates tersedia setelah seeding:

1. **Classic Frame** (free) - Single photo, portrait
2. **Birthday Party** (basic) - Single photo, portrait
3. **Wedding Elegance** (premium) - Single photo, portrait
4. **Photo Strip 4x** (free) - 4 photos, portrait strip
5. **Collage 2x2** (basic) - 4 photos, square grid
6. **Vintage Polaroid** (premium) - Single photo, portrait
7. **Modern Minimal** (basic) - Single photo, portrait
8. **Holiday Special** (premium) - Single photo, portrait

## 🔧 Development Tools

### Hot Reload dengan Air

Install Air untuk hot reload:

```bash
go install github.com/cosmtrek/air@latest
```

Jalankan dengan:

```bash
air
```

### Database Migration Commands

```bash
# Create new migration
go run cmd/migrate/main.go

# Rollback migration (manual)
# Edit database/database.go dan comment out tables yang ingin di-rollback
```

## 📁 Project Structure

```
BackendPhotobooth/
├── cmd/
│   ├── migrate/          # Database migration
│   └── seed/             # Database seeding
├── config/               # Configuration management
├── database/             # Database connection & setup
├── handlers/             # HTTP request handlers
│   ├── admin_handler.go
│   ├── auth_handler.go
│   ├── gopay_handler.go
│   ├── payment_handler.go
│   ├── photo_handler.go
│   ├── promo_handler.go
│   ├── search_handler.go
│   ├── session_handler.go
│   ├── template_handler.go
│   ├── twofa_handler.go
│   └── websocket_handler.go
├── middleware/           # HTTP middlewares
│   ├── audit.go
│   ├── auth.go
│   ├── logger.go
│   └── rate_limit.go
├── models/               # Database models
│   ├── analytics.go
│   ├── audit.go
│   ├── order.go
│   ├── photo.go
│   ├── promo.go
│   ├── qris_payment.go
│   ├── session.go
│   ├── template.go
│   ├── twofa.go
│   └── user.go
├── services/             # Business logic services
│   ├── email.go
│   ├── gopay_qris.go
│   ├── image_processor.go
│   ├── redis.go
│   ├── storage.go
│   └── websocket.go
├── tests/                # Test files
│   ├── auth_test.go
│   ├── integration_test.go
│   └── template_test.go
├── uploads/              # File uploads (gitignored)
├── .env                  # Environment variables
├── .env.example          # Environment template
├── .gitignore
├── go.mod
├── go.sum
├── main.go               # Application entry point
├── Makefile              # Make commands
├── migrate.ps1           # Windows migration script
├── seed.ps1              # Windows seeding script
├── setup.ps1             # Windows setup script
└── README.md             # This file
```

## 🚀 Deployment

### Build untuk Production

```bash
# Build binary
go build -o photobooth-api

# Run binary
./photobooth-api
```

### Docker (Optional)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o photobooth-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/photobooth-api .
COPY --from=builder /app/.env .
EXPOSE 8080
CMD ["./photobooth-api"]
```

### Environment Variables untuk Production

Pastikan set environment variables berikut di production:

```env
ENV=production
JWT_SECRET=<strong-random-secret>
DB_SSLMODE=require
GOPAY_API_URL=https://api.gopay.com/v1
```

## 🔒 Security Best Practices

1. **JWT Secret** - Gunakan secret key yang kuat dan random
2. **Database Password** - Gunakan password yang kuat
3. **SSL/TLS** - Aktifkan SSL untuk database connection di production
4. **Rate Limiting** - Sudah diimplementasikan untuk mencegah abuse
5. **Input Validation** - Semua input sudah divalidasi
6. **SQL Injection** - Menggunakan GORM ORM untuk mencegah SQL injection
7. **CORS** - Configure CORS sesuai kebutuhan
8. **2FA** - Aktifkan 2FA untuk admin accounts

## 📊 Performance Optimization

1. **Redis Caching** - Cache untuk templates, users, dan sessions
2. **Database Indexing** - Index pada kolom yang sering di-query
3. **Connection Pooling** - Database connection pool sudah dikonfigurasi
4. **Rate Limiting** - Mencegah overload server
5. **Pagination** - Semua list endpoints menggunakan pagination

## 🐛 Troubleshooting

### Database Connection Error

```bash
# Check PostgreSQL service
sudo systemctl status postgresql

# Check connection
psql -h localhost -p 5433 -U postgres -d photobooth
```

### Migration Error

```bash
# Drop dan recreate database
dropdb photobooth
createdb photobooth

# Run migration lagi
go run cmd/migrate/main.go
```

### Port Already in Use

```bash
# Check process using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>
```

## 📞 Support

Untuk pertanyaan atau issue, silakan buat issue di repository atau hubungi tim development.

## 📄 License

[Your License Here]

## 👥 Contributors

- [Your Name] - Initial work

---

**Version**: 1.0.0  
**Last Updated**: May 2, 2026  
**Status**: ✅ Production Ready
#   b a c k e n d - p h o t o b o o t h  
 