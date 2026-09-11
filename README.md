# GoLink — Modern URL Shortener & Analytics

[![Go Report Card](https://goreportcard.com/badge/github.com/jaskiratmiglani07/url-shortener)](https://goreportcard.com/report/github.com/jaskiratmiglani07/url-shortener)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A high-performance, production-ready URL shortener service built with idiomatic Go, PostgreSQL persistence, Redis cache-aside, IP-based rate limiting, and link click analytics. Includes an embedded, responsive web dashboard.

---

## Architecture Overview

```
                          +------------------------+
                          |      HTTP Client       |
                          | (Browser / cURL / App) |
                          +-----------+------------+
                                      |
                                      v
                       +-------------------------------+
                       |      Go HTTP Middleware       |
                       | - Panic Recovery              |
                       | - Structured Logging (slog)   |
                       | - Rate Limiter (Token Bucket) |
                       +--------------+----------------+
                                      |
                                      v
                       +-------------------------------+
                       |        net/http Router        |
                       +--------------+----------------+
                                      |
                 +--------------------+--------------------+
                 |                                         |
                 v                                         v
      +---------------------+                   +---------------------+
      |  Shortener Service  |                   |  Analytics Service  |
      | - URL Validation    |                   | - Total Clicks      |
      | - Base62 Encoding   |                   | - Clicks Today      |
      | - Expiry Filtering  |                   | - Last 7 Days       |
      +----------+----------+                   +----------+----------+
                 |                                         |
                 +--------------------+--------------------+
                                      |
                                      v
                       +-------------------------------+
                       |       Cache-Aside Layer       |
                       |   (Redis Key: url:{code})     |
                       +--------------+----------------+
                                      | (Cache Miss / Fallback)
                                      v
                       +-------------------------------+
                       |      PostgreSQL Database      |
                       | - urls (indexed short_code)   |
                       | - url_clicks (indexed url_id) |
                       +-------------------------------+
```

---

## Tech Stack

- **Language**: Go (`1.24+`) using standard library `net/http` and `log/slog`
- **Database**: PostgreSQL 16 (relational schema, auto-migrations, foreign keys, composite indexes)
- **Cache**: Redis 7 (`github.com/redis/go-redis/v9`) with TTL & graceful degradation
- **Driver**: `github.com/lib/pq`
- **Frontend**: Embedded single-page dashboard (`embed.FS`, semantic HTML5, Vanilla CSS & JavaScript)
- **Containers**: Docker & Docker Compose

---

## Features

1. **URL Shortening**: Accepts valid `http` or `https` destination URLs and generates unique Base62 short codes.
2. **Custom Aliases**: Allows users to specify human-readable custom aliases (`3–32` alphanumeric characters, hyphens, underscores). Checks for collisions and rejects reserved paths (`/api`, `/health`, `/analytics`, etc.).
3. **URL Expiration**: Supports optional expiration via relative minutes (`expires_in_minutes`) or ISO timestamps (`expires_at`). Expired URLs cleanly return `410 Gone` and are invalidated from cache.
4. **Redis Cache-Aside**: URL lookups check Redis first (`url:<short_code>`). On a cache miss, data is read from PostgreSQL and written back to Redis with a configurable TTL.
5. **Graceful Degradation**: If Redis becomes temporarily unavailable or disconnected, the service automatically falls back to direct database reads and in-memory rate limiting without dropping user requests.
6. **Configurable Rate Limiting**: Per-IP rate limiting (configurable requests-per-second and burst capacity) backed by Redis sliding windows, with in-memory token bucket fallback.
7. **Link Click Analytics (Distinctive Addition)**:
   - Tracks total clicks per link
   - Tracks clicks today (since midnight UTC)
   - Tracks clicks during the last 7 days
   - Records the most recent click timestamp
   - Returns aggregated statistics via `GET /api/v1/analytics/{shortCode}`
8. **Interactive Web Dashboard**: Embedded modern web interface to shorten links, copy short URLs with one click, and monitor live click statistics.

---

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go                 # Entrypoint, dependency wiring, graceful shutdown
├── internal/
│   ├── api/
│   │   ├── handler.go              # REST handlers: Shorten, Redirect, Preview, Analytics
│   │   ├── handler_test.go         # API endpoint unit tests
│   │   ├── middleware.go           # Logging and panic recovery middleware
│   │   ├── middleware_test.go      # Middleware unit tests
│   │   ├── response.go             # JSON envelopes and error formatting
│   │   └── router.go               # HTTP ServeMux route registration
│   ├── config/
│   │   ├── config.go               # Environment loader and .env parser
│   │   └── config_test.go          # Config unit tests
│   ├── database/
│   │   └── redis.go                # Redis client connection factory
│   ├── model/
│   │   └── url.go                  # Domain entities and DTOs
│   ├── ratelimit/
│   │   ├── limiter.go              # Redis sliding window and in-memory token bucket
│   │   └── limiter_test.go         # Rate limiter unit tests
│   ├── service/
│   │   ├── base62.go               # Base62 encoder/decoder and random code generator
│   │   ├── base62_test.go          # Base62 test cases
│   │   ├── expiration_alias_test.go# Expiration and custom alias boundary tests
│   │   ├── shortener.go            # Shortener orchestration and click tracking
│   │   ├── shortener_test.go       # Shortener service test cases
│   │   ├── validator.go            # URL syntax and alias safety validation
│   │   └── validator_test.go       # Validator test cases
│   └── store/
│       ├── cached_store.go         # Redis cache-aside wrapper
│       ├── cached_store_test.go    # Cache hit, miss, and degradation tests
│       ├── memory.go               # In-memory URLStore implementation for tests
│       ├── memory_test.go          # MemoryStore unit tests
│       ├── postgres.go             # PostgreSQL implementation and auto-migration
│       └── store.go                # URLStore interface and error definitions
├── migrations/
│   ├── 000001_create_urls_table.up.sql
│   ├── 000001_create_urls_table.down.sql
│   ├── 000002_create_clicks_table.up.sql
│   └── 000002_create_clicks_table.down.sql
├── tests/
│   └── integration_test.go         # Full end-to-end integration tests
├── web/
│   ├── static/
│   │   ├── css/style.css           # Modern, responsive styles
│   │   └── js/app.js               # Frontend interactive controller
│   ├── index.html                  # Dashboard HTML template
│   ├── web.go                      # Go embed.FS filesystem wrapper
│   └── web_test.go                 # Asset embedding tests
├── .env.example
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Environment Variables

Copy `.env.example` to `.env` to configure your environment:

```bash
cp .env.example .env
```

| Variable | Default | Description |
| :--- | :--- | :--- |
| `SERVER_PORT` | `8080` | Port the HTTP server listens on |
| `BASE_URL` | `http://localhost:8080` | Base domain used when generating shortened URLs |
| `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable` | PostgreSQL connection string |
| `DB_MAX_OPEN_CONNS` | `25` | Maximum active DB connections in pool |
| `DB_MAX_IDLE_CONNS` | `10` | Maximum idle DB connections in pool |
| `DB_CONN_MAX_LIFETIME_MINUTES` | `30` | Connection reuse lifetime in minutes |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection URL |
| `CACHE_TTL_MINUTES` | `60` | Duration to cache URL lookups in Redis |
| `RATE_LIMIT_ENABLED`| `true` | Toggle client IP rate limiting |
| `RATE_LIMIT_RPS` | `10` | Allowed requests per second per IP |
| `RATE_LIMIT_BURST` | `20` | Maximum request burst capacity per IP |

---

## Quickstart & Local Setup

### Option 1: Running with Docker Compose (Recommended)

Start PostgreSQL, Redis, and the URL Shortener application with a single command:

```bash
docker compose up --build -d
```

View application logs:
```bash
docker compose logs -f app
```

The service and dashboard will be available at [http://localhost:8080](http://localhost:8080).

To stop all services and remove volumes:
```bash
docker compose down -v
```

---

### Option 2: Running Locally with Go

If running locally without Docker:
1. Ensure PostgreSQL is running on port `5432` with a database named `urlshortener`.
2. Ensure Redis is running on port `6379`.
3. Run the application:

```bash
go run ./cmd/server
```

*(Note: If PostgreSQL or Redis are not reachable, the application will automatically run in local in-memory fallback mode for quick evaluation.)*

---

## API Endpoints & Usage

### 1. Health Probe

```bash
curl -i http://localhost:8080/health
```

**Response (`200 OK`)**:
```json
{
  "status": "healthy"
}
```

---

### 2. Shorten a URL

`POST /api/v1/shorten` (or `POST /shorten`)

#### Request Body
```json
{
  "url": "https://en.wikipedia.org/wiki/Go_(programming_language)",
  "custom_alias": "golang-wiki",
  "expires_in_minutes": 1440
}
```

#### Example cURL
```bash
curl -X POST http://localhost:8080/api/v1/shorten \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://en.wikipedia.org/wiki/Go_(programming_language)",
    "custom_alias": "golang-wiki",
    "expires_in_minutes": 1440
  }'
```

**Response (`201 Created`)**:
```json
{
  "short_code": "golang-wiki",
  "short_url": "http://localhost:8080/golang-wiki",
  "original_url": "https://en.wikipedia.org/wiki/Go_(programming_language)",
  "created_at": "2026-09-12T04:10:00Z",
  "expires_at": "2026-09-13T04:10:00Z",
  "is_expired": false
}
```

---

### 3. URL Redirection

`GET /{shortCode}`

```bash
curl -i http://localhost:8080/golang-wiki
```

**Response (`302 Found`)**:
```http
HTTP/1.1 302 Found
Location: https://en.wikipedia.org/wiki/Go_(programming_language)
Cache-Control: no-cache, no-store, must-revalidate
Pragma: no-cache
Expires: 0
```
*(Every redirect visit increments the click counter asynchronously).*

---

### 4. Link Preview (Metadata without Redirect)

`GET /api/v1/preview/{shortCode}` (or `GET /preview/{shortCode}`)

```bash
curl -i http://localhost:8080/api/v1/preview/golang-wiki
```

**Response (`200 OK`)**:
```json
{
  "id": 1,
  "short_code": "golang-wiki",
  "original_url": "https://en.wikipedia.org/wiki/Go_(programming_language)",
  "created_at": "2026-09-12T04:10:00Z",
  "expires_at": "2026-09-13T04:10:00Z"
}
```

---

### 5. Click Analytics (Distinctive Addition)

`GET /api/v1/analytics/{shortCode}` (or `GET /analytics/{shortCode}`)

```bash
curl -i http://localhost:8080/api/v1/analytics/golang-wiki
```

**Response (`200 OK`)**:
```json
{
  "short_code": "golang-wiki",
  "original_url": "https://en.wikipedia.org/wiki/Go_(programming_language)",
  "created_at": "2026-09-12T04:10:00Z",
  "expires_at": "2026-09-13T04:10:00Z",
  "is_expired": false,
  "total_clicks": 14,
  "clicks_today": 14,
  "clicks_last_7_days": 14,
  "last_clicked_at": "2026-09-12T04:10:25Z"
}
```

---

## Testing

Run the full test suite with data race detector:

```bash
make test
```
or
```bash
go test -v -race ./...
```

The test suite includes:
- Base62 encoding/decoding edge cases and collision handling
- URL validator (scheme constraints, valid hostnames, loopback prevention, maximum length)
- Custom alias validation (boundary lengths, allowed character set, reserved route filtering)
- URL expiration calculation and resolution rejection
- Redis cache-aside read/write behavior and graceful degradation
- IP-based rate limiting under burst load
- End-to-end integration flow (Shorten &rarr; Redirect &rarr; Click Recording &rarr; Analytics Verification)

---

## Design Decisions & Trade-offs

1. **Standard Library net/http over Third-Party Web Frameworks**:
   - Go 1.22+ introduced enhanced HTTP route matching (`METHOD /path/{param}`), eliminating the necessity of heavier frameworks like Gin, Echo, or Fiber.
   - Standard library usage minimizes external attack surfaces, lowers binary size, and ensures maximum compatibility.

2. **Decoupled Relational Click Analytics**:
   - Rather than introducing complex distributed queues (Kafka) or dedicated analytics databases (ClickHouse), clicks are recorded with relational integrity in PostgreSQL (`url_clicks` table).
   - A single composite index `(url_id, clicked_at DESC)` allows instantaneous aggregation for `total_clicks`, `clicks_today`, and `clicks_last_7_days` using standard SQL `COUNT(*) FILTER (...)`.
   - Recording occurs in a non-blocking goroutine so user redirection latency is completely decoupled from database insert performance.

3. **Multi-Layered Cache-Aside**:
   - Read-heavy redirect workloads hit Redis directly.
   - If Redis fails or is restarting, the application logs a warning and transparently queries PostgreSQL, maintaining zero service downtime.

4. **In-Memory Store for Hermetic Tests**:
   - Providing both `PostgresStore` and `MemoryStore` behind the same `URLStore` interface allows the entire unit and integration test suite to run in milliseconds without requiring external container dependencies.

---

## Known Limitations

- **Geo & User-Agent Analytics**: The analytics feature tracks temporal click volume (total, today, last 7 days, last timestamp) rather than geographic IP resolution or device/browser user-agent breakdown.
- **Single-Node Rate Limiting Fallback**: When Redis is offline, rate limiting falls back to an in-memory token bucket. In multi-instance deployments, each instance maintains its own in-memory quota until Redis reconnects.
- **Analytics Background Writes**: Clicks are dispatched asynchronously in background goroutines. In the rare event of an ungraceful hard server crash, clicks currently queued in memory before database commit could be lost.

---

## License

This project is licensed under the MIT License.
