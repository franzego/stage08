# Wallet Service API

A secure backend wallet service built with Go, featuring Google OAuth authentication, API key management, Paystack payment integration, and wallet-to-wallet transfers.

## Features

-  **Google OAuth 2.0 Authentication** - Secure user authentication with JWT tokens
-  **API Key Management** - Create and manage up to 5 API keys per user with granular permissions
-  **Paystack Integration** - Seamless deposit functionality with webhook support
-  **Wallet Transfers** - Atomic wallet-to-wallet money transfers
-  **Transaction History** - Track all deposits and transfers
-  **Security** - HMAC signature verification, JWT validation, and API key hashing

## Tech Stack

- **Language**: Go 1.21
- **Framework**: Gin
- **Database**: PostgreSQL with sqlx
- **Authentication**: JWT (golang-jwt/v5) + Google OAuth2
- **Payment**: Paystack API
- **Deployment**: Docker, Railway

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 15+
- Docker (optional)
- Google OAuth credentials
- Paystack API keys (test or live)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/franzego/stage08.git
cd stage08
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**

Create a `.env` file in the project root:

```env
# Server Configuration
PORT=8080

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=wallet_service
DB_SSLMODE=disable

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key

# Google OAuth Configuration
GOOGLE_CLIENT_ID=your-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-client-secret
GOOGLE_REDIRECT_URL=http://localhost:8080/auth/google/callback

# Paystack Configuration
PAYSTACK_SECRET_KEY=sk_test_your_secret_key
PAYSTACK_PUBLIC_KEY=pk_test_your_public_key
```

4. **Set up the database**

```bash
# Start PostgreSQL (using Docker)
docker run --name wallet-postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=wallet_service -p 5432:5432 -d postgres:15-alpine

# Migrations will run automatically on startup
```

5. **Build and run**

```bash
go build -o bin/wallet-service
./bin/wallet-service
```

The server will start on `http://localhost:8080`

## API Documentation

### Authentication Flow

#### 1. Google OAuth Login
```http
GET /auth/google
```
Redirects to Google sign-in page.

#### 2. OAuth Callback
```http
GET /auth/google/callback?code=xxx&state=xxx
```
Returns JWT token after successful authentication:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe"
  }
}
```

### API Key Management

All API key endpoints require JWT authentication.

#### Create API Key
```http
POST /keys/create
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "name": "My API Key",
  "permissions": ["deposit", "transfer", "read"],
  "expiry": "1D"
}
```

**Expiry options**: `1H` (1 hour), `1D` (1 day), `1M` (1 month), `1Y` (1 year)

**Response**:
```json
{
  "api_key": "sk_live_xxxxx",
  "expires_at": "2025-12-11T10:00:00Z"
}
```

#### List API Keys
```http
GET /keys/list
Authorization: Bearer {jwt_token}
```

#### Rollover Expired Key
```http
POST /keys/rollover
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "expired_key_id": "uuid",
  "expiry": "1M"
}
```

#### Revoke API Key
```http
POST /keys/revoke
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "key_id": "uuid"
}
```

### Wallet Operations

All wallet endpoints support authentication via JWT token OR API key.

**Authentication options**:
- Header: `Authorization: Bearer {jwt_token}`
- Header: `x-api-key: sk_live_xxxxx`

#### Get Balance
```http
GET /wallet/balance
Authorization: Bearer {jwt_token}
```
**Requires**: `read` permission

**Response**:
```json
{
  "balance": 15000,
  "wallet_number": "4566678954356"
}
```

#### Initialize Deposit
```http
POST /wallet/deposit
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "amount": 10000
}
```
**Requires**: `deposit` permission  
**Amount**: In kobo (100 kobo = 1 Naira), minimum 100

**Response**:
```json
{
  "reference": "DEP_12345678_abcd1234",
  "authorization_url": "https://checkout.paystack.com/xxxxx"
}
```

#### Check Deposit Status
```http
GET /wallet/deposit/{reference}/status
Authorization: Bearer {jwt_token}
```
**Requires**: `read` permission

**Response**:
```json
{
  "reference": "DEP_12345678_abcd1234",
  "status": "success",
  "amount": 10000
}
```

#### Transfer Money
```http
POST /wallet/transfer
Authorization: Bearer {jwt_token}
Content-Type: application/json

{
  "wallet_number": "4566678954356",
  "amount": 5000
}
```
**Requires**: `transfer` permission  
**Amount**: In kobo

**Response**:
```json
{
  "status": "success",
  "message": "Transfer completed"
}
```

#### Get Transaction History
```http
GET /wallet/transactions
Authorization: Bearer {jwt_token}
```
**Requires**: `read` permission

**Response**:
```json
[
  {
    "type": "deposit",
    "amount": 10000,
    "status": "success",
    "reference": "DEP_12345678_abcd1234",
    "created_at": "2025-12-10T10:00:00Z"
  },
  {
    "type": "transfer_out",
    "amount": 5000,
    "status": "success",
    "created_at": "2025-12-10T11:00:00Z"
  }
]
```

### Webhook

#### Paystack Webhook
```http
POST /wallet/paystack/webhook
x-paystack-signature: {signature}
Content-Type: application/json
```

This endpoint is called automatically by Paystack when a payment succeeds. It verifies the signature and credits the wallet.

**No authentication required** - validated by HMAC signature.

## Swagger Documentation

Interactive API documentation is available at:
```
http://localhost:8080/swagger
```

Or download the OpenAPI spec:
```
http://localhost:8080/swagger.yaml
```

## Database Schema

### Users Table
- Stores Google OAuth user information
- Automatically creates a wallet on user creation

### Wallets Table
- One wallet per user
- Balance stored in kobo (smallest currency unit)
- Unique 13-digit wallet number

### Transactions Table
- Records all deposits and transfers
- Types: `deposit`, `transfer_in`, `transfer_out`
- Statuses: `pending`, `success`, `failed`
- Idempotent processing using unique references

### API Keys Table
- Up to 5 active keys per user (enforced by DB trigger)
- SHA256 hashed keys for security
- Granular permissions: `deposit`, `transfer`, `read`
- Expiration and revocation support

## Security Features

1. **Authentication**
   - Google OAuth 2.0 with state parameter for CSRF protection
   - JWT tokens with 24-hour expiration
   - API keys with SHA256 hashing

2. **Authorization**
   - Permission-based access control
   - Middleware validates JWT or API key on protected routes

3. **Payment Security**
   - Paystack webhook signature verification (HMAC SHA-512)
   - Idempotent transaction processing
   - Amount validation

4. **Database Security**
   - Atomic wallet operations using transactions
   - Balance checks prevent negative balances
   - Unique constraints on references and wallet numbers

## Testing

### Using Paystack Test Cards

For testing deposits without real money:

- **Card Number**: 4084084084084081
- **Expiry**: Any future date (e.g., 12/25)
- **CVV**: 408
- **PIN**: 0000 or 1234

### Testing Webhooks Locally

Paystack webhooks require a public URL. Use ngrok:

```bash
# Terminal 1: Start the server
./bin/wallet-service

# Terminal 2: Expose with ngrok
ngrok http 8080

# Copy the ngrok URL (e.g., https://abc123.ngrok-free.app)
# Add to Paystack Dashboard: https://abc123.ngrok-free.app/wallet/paystack/webhook
```

## Deployment

### Railway

1. **Push to GitHub**
```bash
git push origin main
```

2. **Connect to Railway**
   - Create new project in Railway
   - Connect your GitHub repository
   - Add PostgreSQL plugin

3. **Set Environment Variables**
   - Add all variables from `.env`
   - Update `GOOGLE_REDIRECT_URL` to your Railway domain
   - Use production Paystack keys

4. **Configure Paystack Webhook**
   - Add webhook URL: `https://your-app.up.railway.app/wallet/paystack/webhook`

### Docker

```bash
# Build image
docker build -t wallet-service .

# Run container
docker run -p 8080:8080 --env-file .env wallet-service
```

## Project Structure

```
stage08/
├── cmd/                    # Application entry points
├── config/                 # Configuration management
│   └── config.go
├── internal/
│   ├── database/          # Database connection and migrations
│   ├── handlers/          # HTTP request handlers
│   │   ├── auth_handler.go
│   │   ├── apikey_handler.go
│   │   ├── wallet_handler.go
│   │   └── paystack_handler.go
│   ├── middleware/        # Authentication and authorization
│   │   ├── jwt_auth.go
│   │   └── auth.go
│   ├── models/            # Data models
│   │   └── models.go
│   ├── repository/        # Database operations
│   │   ├── user_repository.go
│   │   ├── wallet_repository.go
│   │   ├── transaction_repository.go
│   │   └── apikey_repository.go
│   ├── paystack/          # Paystack API client
│   │   └── client.go
│   └── utils/             # Utility functions
│       ├── jwt.go
│       ├── random.go
│       └── expiry.go
├── migrations/            # SQL migration files
│   ├── 001_create_users_table.up.sql
│   ├── 002_create_wallets_table.up.sql
│   ├── 003_create_transactions_table.up.sql
│   └── 004_create_api_keys_table.up.sql
├── scripts/               # Helper scripts
│   └── generate_token.go
├── Dockerfile
├── swagger.yaml           # OpenAPI specification
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Configuration Guide

### Google OAuth Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project
3. Enable Google+ API
4. Create OAuth 2.0 credentials
5. Add authorized redirect URIs:
   - `http://localhost:8080/auth/google/callback`
   - `https://your-production-domain.com/auth/google/callback`
6. Copy Client ID and Client Secret to `.env`

### Paystack Setup

1. Sign up at [Paystack](https://paystack.com/)
2. Get API keys from Settings → API Keys & Webhooks
3. For testing, use test keys (starts with `sk_test_` and `pk_test_`)
4. Add webhook URL in Settings → Webhooks
5. Copy keys to `.env`

## Troubleshooting

### Database Connection Issues
- Ensure PostgreSQL is running
- Check database credentials in `.env`
- Verify database name exists

### Webhook Not Working
- Webhooks don't work on localhost without ngrok
- Verify webhook URL is configured in Paystack Dashboard
- Check server logs for signature verification errors

### API Key Limit
- Maximum 5 active keys per user
- Revoke unused keys to free up slots
- Expired keys still count towards the limit until revoked

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Support

For issues and questions:
- Create an issue on GitHub
- Email: daverecords02@gmail.com

## Acknowledgments

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Paystack API](https://paystack.com/docs/api/)
- [Google OAuth 2.0](https://developers.google.com/identity/protocols/oauth2)

