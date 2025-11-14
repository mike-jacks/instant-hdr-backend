# Imagen AI Backend API Hub

A comprehensive Go backend service that serves as a complete hub for all Imagen AI API features. Integrated with Supabase for JWT authentication, database, storage, and real-time status updates. Designed for real estate photography where listings (home addresses) are equivalent to projects.

## Features

- **Project Management**: Create, list, get, and delete Imagen projects
- **Image Upload**: Upload bracketed images for HDR processing
- **HDR Processing**: Process images with Imagen AI's HDR merge and other AI tools
- **Real-time Status**: Status updates via Supabase Realtime
- **Automatic Storage**: Automatically stores processed JPEG images in Supabase Storage
- **Webhook Support**: Receives status updates from Imagen AI via webhooks
- **JWT Authentication**: Secure endpoints with Supabase JWT tokens

## Prerequisites

- Go 1.21 or later
- Supabase account and project
- Imagen AI API key
- PostgreSQL database (via Supabase)

## Setup

### 1. Clone the repository

```bash
git clone <repository-url>
cd instant-hdr-backend
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Configure Supabase

#### Create Storage Bucket

1. Go to your Supabase project dashboard
2. Navigate to Storage
3. Create a new bucket named `processed-images`
4. Set it to public (or configure policies for private access)

#### Get Database Connection String

1. Go to Project Settings > Database
2. Copy the Connection string (URI format)
3. This will be used for the `DATABASE_URL` environment variable

#### Get Supabase Credentials

1. Go to Project Settings > API
2. Copy the following:
   - Project URL → `SUPABASE_URL`
   - publishable key → `SUPABASE_PUBLISHABLE_KEY` (replaces the anon key)
   - JWT Secret → `SUPABASE_JWT_SECRET` (for token verification)

**How to get the JWT Secret:**

- In your Supabase dashboard, go to **Project Settings** > **API**
- Scroll down to the **JWT Settings** section
- Copy the **JWT Secret** value (it's a long string)
- This secret is used by Supabase to sign JWT tokens, and your backend uses it to verify those tokens

**Note:** The backend uses the publishable key for Supabase client and storage operations. Ensure your storage bucket has appropriate RLS policies configured if needed. The JWT secret is used for verifying authentication tokens from clients.

### 4. Configure Environment Variables

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

Edit `.env` with your actual credentials.

### 5. Run Migrations

Migrations run automatically on server startup. They are idempotent and safe to run multiple times.

### 6. Run the Server

```bash
go run cmd/server/main.go
```

The server will start on port 8080 (or the port specified in `PORT` environment variable).

## API Documentation

Interactive API documentation is available via Swagger UI when the server is running:

- **Swagger UI**: `http://localhost:8080/swagger/index.html`

The documentation is auto-generated from code annotations. To regenerate after making changes:

```bash
# Install swag CLI (if not already installed)
go install github.com/swaggo/swag/cmd/swag@latest

# Generate documentation
swag init -g cmd/server/main.go -o docs
```

## API Endpoints

All endpoints (except `/health` and `/api/v1/webhooks/imagen`) require JWT authentication via `Authorization: Bearer <token>` header.

### Project Management

- `POST /api/v1/projects` - Create a new project
- `GET /api/v1/projects` - List all projects for authenticated user
- `GET /api/v1/projects/:project_id` - Get project details
- `DELETE /api/v1/projects/:project_id` - Delete a project

### Image Upload & Processing

- `POST /api/v1/projects/:project_id/upload` - Upload bracketed images
- `POST /api/v1/projects/:project_id/process` - Initiate HDR processing

### Status & Files

- `GET /api/v1/projects/:project_id/status` - Get project status (optional/fallback)
- `GET /api/v1/projects/:project_id/files` - List project files

### Webhooks

- `POST /api/v1/webhooks/imagen` - Imagen AI webhook endpoint (no auth, uses HMAC)

### Health

- `GET /health` - Health check endpoint

## Real-time Updates

The iPhone app connects directly to Supabase Realtime (not through this backend) to receive real-time status updates:

- Channels: `project:{project_id}` and `user:{user_id}`
- Events: `upload_started`, `upload_completed`, `processing_started`, `processing_progress`, `processing_completed`, `processing_failed`, `download_ready`

## Deployment to Railway.app

### Configuration

The project includes a `Dockerfile` and `railway.json` for deployment. Railway will automatically detect and use the Dockerfile.

**Docker Build:**

- Multi-stage Docker build for optimized image size
- Uses Go 1.25 Alpine for building
- Final image uses minimal Alpine Linux
- Includes ca-certificates for HTTPS requests

**Alternative (Nixpacks):**
If you prefer to use Nixpacks instead of Docker, you can update `railway.json`:

- **Build Command**: `go build -o server ./cmd/server`
- **Start Command**: `./server`

**Important:** Do NOT use `go build cmd/server/main.go` - this will fail. Use `go build ./cmd/server` or `go build -o server ./cmd/server`.

### Deployment Steps

1. Create a new Railway project
2. Connect your GitHub repository
3. Add all environment variables from `.env.example`:
   - `IMAGEN_API_KEY`
   - `IMAGEN_API_BASE_URL`
   - `IMAGEN_WEBHOOK_SECRET`
   - `SUPABASE_URL`
   - `SUPABASE_PUBLISHABLE_KEY`
   - `SUPABASE_JWT_SECRET`
   - `SUPABASE_STORAGE_BUCKET`
   - `DATABASE_URL`
   - `WEBHOOK_CALLBACK_URL`
   - `ENVIRONMENT=production`
4. Railway automatically sets `PORT` - your app will use it via `os.Getenv("PORT")`
5. Deploy!

Railway will automatically:

- Build the Go application using the correct build command
- Run migrations on startup (if `DATABASE_URL` is set)
- Start the server

### Troubleshooting

If you see `package cmd/server/main.go is not in std`:

- Ensure your build command is `go build -o server ./cmd/server` (not `go build cmd/server/main.go`)
- Check that `railway.json` exists in your project root
- Verify your `go.mod` file is in the project root

## Testing

Run unit tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## Project Structure

```
instant-hdr-backend/
├── cmd/server/          # Main application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP request handlers
│   ├── middleware/      # Middleware (auth, etc.)
│   ├── imagen/          # Imagen API client
│   ├── supabase/        # Supabase clients (storage, realtime, database)
│   ├── models/          # Data models
│   ├── database/        # Database migrations and queries
│   ├── services/        # Business logic services
│   └── test/            # Unit tests
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Configuration

### Required Environment Variables

```bash
# Imagen AI Configuration
IMAGEN_API_KEY=your-imagen-api-key-here
IMAGEN_API_BASE_URL=https://api.imagen-ai.com/v1/
IMAGEN_WEBHOOK_SECRET=your-webhook-secret-here

# Supabase Configuration
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_PUBLISHABLE_KEY=your-publishable-key-here
SUPABASE_JWT_SECRET=your-jwt-secret-here
SUPABASE_STORAGE_BUCKET=processed-images

# Database Configuration
DATABASE_URL=postgresql://postgres:[password]@db.[project-ref].supabase.co:5432/postgres

# Webhook Configuration
WEBHOOK_CALLBACK_URL=https://your-backend-url.com/api/v1/webhooks/imagen

# Server Configuration
PORT=8080
ENVIRONMENT=development
```

**Note:** Create a `.env` file in the project root with these variables. See `.env.example` for a template (if available).

## License

[Your License Here]
