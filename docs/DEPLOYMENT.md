# Halisi Deployment Guide

## Overview

Halisi can be deployed using Docker or directly via Railway (the current deployment platform). The application is a single binary that serves both HTML pages and static files on port 8080.

## Deployment Options

| Option | Description | Best For |
|--------|-------------|----------|
| **Railway** | Platform-as-a-Service with nixpacks build | Quick deployment, managed infrastructure |
| **Docker** | Containerized deployment | Self-hosted, Kubernetes, Docker Compose |
| **Bare Metal** | Direct Go binary deployment | VPS, minimal overhead |

## Railway Deployment

Railway is the current deployment platform. The project includes a `nixpacks.toml` configuration for building the Go binary with CGO support (required for the PostgreSQL driver).

### nixpacks.toml Configuration

```toml
[phases.setup]
nixPkgs = ["go", "gcc", "sqlite"]

[variables]
CGO_ENABLED = "1"
```

### Steps

1. **Connect** your repository to Railway:
   ```bash
   railway init
   railway link
   railway up
   ```

2. **Set** the required environment variables in Railway's dashboard or CLI:
   ```bash
   railway variables set DATABASE_URL="postgres://..."
   railway variables set GOOGLE_CLIENT_ID="..."
   railway variables set GOOGLE_CLIENT_SECRET="..."
   railway variables set SMTP_HOST="smtp.gmail.com"
   railway variables set SMTP_PORT="587"
   railway variables set SMTP_USERNAME="..."
   railway variables set SMTP_PASSWORD="..."
   railway variables set SMTP_FROM="noreply@halisi.com"
   railway variables set APP_URL="https://your-app.railway.app"
   ```

3. **Update** the Google OAuth2 redirect URI:
   - In Google Cloud Console, add `https://your-app.railway.app/auth/google/callback` to the authorized redirect URIs.

4. **Deploy**:
   ```bash
   railway deploy
   ```

### Railway Database

Railway provides a managed PostgreSQL database. Link it to your service:

```bash
railway plugin add postgresql
railway link
```

The `DATABASE_URL` is automatically set by Railway.

## Docker Deployment

### Build the Image

```bash
docker build -t halisi:latest .
```

### Run the Container

```bash
docker run -d \
  --name halisi \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@db:5432/halisi?sslmode=disable" \
  -e GOOGLE_CLIENT_ID="your-client-id" \
  -e GOOGLE_CLIENT_SECRET="your-client-secret" \
  -e SMTP_HOST="smtp.gmail.com" \
  -e SMTP_PORT="587" \
  -e SMTP_USERNAME="your-email" \
  -e SMTP_PASSWORD="your-password" \
  -e SMTP_FROM="noreply@halisi.com" \
  -e APP_URL="https://your-domain.com" \
  halisi:latest
```

### Docker Compose

For a complete setup with PostgreSQL:

```yaml
version: "3.8"

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: "postgres://halisi:password@db:5432/halisi?sslmode=disable"
      GOOGLE_CLIENT_ID: "${GOOGLE_CLIENT_ID}"
      GOOGLE_CLIENT_SECRET: "${GOOGLE_CLIENT_SECRET}"
      SMTP_HOST: "${SMTP_HOST}"
      SMTP_PORT: "${SMTP_PORT}"
      SMTP_USERNAME: "${SMTP_USERNAME}"
      SMTP_PASSWORD: "${SMTP_PASSWORD}"
      SMTP_FROM: "${SMTP_FROM}"
      APP_URL: "https://your-domain.com"
    depends_on:
      - db

  db:
    image: postgres:15
    environment:
      POSTGRES_USER: halisi
      POSTGRES_PASSWORD: password
      POSTGRES_DB: halisi
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

Then run:

```bash
docker compose up -d
```

### Multi-Stage Dockerfile (Optimized)

The current `Dockerfile` is simple. For production, consider a multi-stage build to reduce image size:

```dockerfile
# Build stage
FROM golang:1.26 AS builder
WORKDIR /app
ENV CGO_ENABLED=1
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/api

# Runtime stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

## Bare Metal Deployment

### Build the Binary

```bash
CGO_ENABLED=1 go build -o server ./cmd/api
```

### Run as a Systemd Service

Create `/etc/systemd/system/halisi.service`:

```ini
[Unit]
Description=Halisi Gas Order Application
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/halisi
ExecStart=/opt/halisi/server
Environment=DATABASE_URL=postgres://user:pass@localhost:5432/halisi?sslmode=disable
Environment=GOOGLE_CLIENT_ID=your-client-id
Environment=GOOGLE_CLIENT_SECRET=your-client-secret
Environment=SMTP_HOST=smtp.gmail.com
Environment=SMTP_PORT=587
Environment=SMTP_USERNAME=your-email
Environment=SMTP_PASSWORD=your-password
Environment=SMTP_FROM=noreply@halisi.com
Environment=APP_URL=https://your-domain.com
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable halisi
sudo systemctl start halisi
sudo systemctl status halisi
```

## Reverse Proxy (Nginx)

For production, place Nginx in front of the application:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### SSL with Let's Encrypt

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## Environment Variables Summary

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `DATABASE_URL` | Yes | PostgreSQL connection string | `postgres://user:pass@host:5432/halisi?sslmode=disable` |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth2 Client ID | `123456.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth2 Client Secret | `GOCSPX-...` |
| `SMTP_HOST` | No* | SMTP server host | `smtp.gmail.com` |
| `SMTP_PORT` | No* | SMTP server port | `587` |
| `SMTP_USERNAME` | No* | SMTP username | `user@gmail.com` |
| `SMTP_PASSWORD` | No* | SMTP password | `app-password` |
| `SMTP_FROM` | No* | Sender email address | `noreply@halisi.com` |
| `APP_URL` | No | Base URL for email links | `https://halisi.com` |

\* Required if email verification is enabled.

## File Uploads

KYC document uploads are stored on the server's filesystem. In production:

1. **Ensure** the upload directory exists and is writable.
2. **Consider** using object storage (e.g., AWS S3, Google Cloud Storage) instead of local filesystem for scalability.
3. **Configure** the reverse proxy to allow large file uploads:
   ```nginx
   client_max_body_size 10M;
   ```

## Monitoring

- Check application logs: `journalctl -u halisi` (systemd) or `docker logs halisi`.
- Monitor database connection pool health.
- Set up health checks on `/` endpoint.

## Backup

Back up the PostgreSQL database regularly:

```bash
pg_dump -h localhost -U halisi halisi > backup_$(date +%Y%m%d).sql
```

For Railway, use Railway's built-in backup features or set up automated PostgreSQL dumps.
