# flowithgo

A distributed background job queue built in Go — inspired by Sidekiq and BullMQ, built from scratch.

**Live Demo:** https://flowithgo-production.up.railway.app/dashboard

Instead of Redis, flowithgo uses PostgreSQL with `SELECT FOR UPDATE SKIP LOCKED` for concurrency-safe job claiming. This keeps the infrastructure simple (one database, no separate message broker) while remaining correct under concurrent workers.

---

## What it does

- Submit background jobs via HTTP API
- Worker pool of N goroutines polls for and executes jobs concurrently
- Failed jobs are retried automatically with exponential backoff and jitter
- Jobs that exhaust all attempts move to a dead letter queue
- Dead jobs can be inspected and requeued from the dashboard
- Real-time dashboard via Server-Sent Events — no polling, no WebSockets

---

## Architecture

```
HTTP Client
     │
     ▼
┌─────────────┐
│  API Server │  POST /jobs, GET /jobs/:id, GET /stats
│  (net/http) │  GET /dlq, POST /dlq/:id/requeue
└──────┬──────┘
       │
       ▼
┌─────────────┐        ┌─────────────────────┐
│  JobStore   │◄──────►│  PostgreSQL (Neon)  │
│  (store/)   │        │  jobs table         │
└──────┬──────┘        └─────────────────────┘
       │
       ▼
┌─────────────────────────────┐
│  Worker Pool (5 goroutines) │
│  each worker:               │
│  - polls DB every 2s        │
│  - claims job (SKIP LOCKED) │
│  - executes handler         │
│  - updates status           │
└─────────────────────────────┘
       │
       ▼
┌─────────────┐
│ Broadcaster │──► SSE ──► Dashboard (browser)
│   (SSE)     │
└─────────────┘
```

---

## Key Technical Decisions

**PostgreSQL over Redis for the queue**

Most job queues (Sidekiq, BullMQ) use Redis as the queue backend. flowithgo uses PostgreSQL with `SELECT FOR UPDATE SKIP LOCKED` — a pattern used in production by companies like Shopify. This avoids running a separate Redis instance while still being correct under concurrent workers. The tradeoff is throughput — Redis is faster for very high job volumes, but Postgres is sufficient for most real-world workloads.

**`SELECT FOR UPDATE SKIP LOCKED`**

When multiple workers poll simultaneously, naive `SELECT` would return the same row to multiple workers. `FOR UPDATE` locks the selected row; `SKIP LOCKED` skips already-locked rows instead of waiting. Each worker gets a different job atomically, with no double-claiming and no distributed lock needed.

**SSE over WebSockets for the dashboard**

Server-Sent Events are unidirectional (server → client), which is all the dashboard needs. SSE is simpler than WebSockets — no handshake, works over plain HTTP, natively supported in all browsers via `EventSource`. No extra library needed on either side.

**Exponential backoff with jitter on retry**

When a job fails, retrying immediately could hammer a failing downstream service. flowithgo schedules retries with exponentially increasing delays plus random jitter — so retries spread out over time rather than spiking together.

---

## API Reference

### Submit a job
```
POST /jobs
Content-Type: application/json

{
  "type": "send_email",
  "payload": { "to": "user@example.com", "subject": "Welcome" },
  "max_attempts": 3
}
```

### Get job status
```
GET /jobs/:id
```

### Queue stats
```
GET /stats
```

### Dead letter queue
```
GET /dlq
POST /dlq/:id/requeue
```

### Real-time dashboard
```
GET /dashboard
GET /events   (SSE stream)
```

---

## Running locally

### Prerequisites
- Docker
- A [Neon](https://neon.tech) PostgreSQL database (free tier works)

### Setup

**1. Clone the repo**
```bash
git clone https://github.com/krishnabhardwaj25/flowithgo
cd flowithgo
```

**2. Create a Neon project**

Go to [neon.tech](https://neon.tech), create a new project, and copy the **direct** connection string (not the pooled one).

**3. Run migrations**

Open the Neon SQL editor and run the contents of:
- `migrations/001_create_jobs.sql`
- `migrations/002_add_error_column.sql`

**4. Configure environment**
```bash
cp .env.example .env
# paste your Neon DATABASE_URL into .env
```

**5. Start with Docker**
```bash
docker-compose up --build
```

Server starts on `http://localhost:8080`.
Dashboard at `http://localhost:8080/dashboard`.

---

## Adding a new job type

Register a handler in `internal/handlers/handlers.go`:

```go
func MyNewJob(payload []byte) error {
    var data struct {
        Field string `json:"field"`
    }
    if err := json.Unmarshal(payload, &data); err != nil {
        return fmt.Errorf("invalid payload: %w", err)
    }
    log.Printf("[my_new_job] doing work with %s", data.Field)
    return nil
}
```

Then register it in `internal/worker/worker.go`:
```go
w.handlers["my_new_job"] = handlers.MyNewJob
```

---

## What I'd add next

- Cron/scheduled jobs (recurring jobs on a fixed schedule)
- Job priorities (high/medium/low priority queues)
- Prometheus metrics endpoint (`/metrics`)
- Worker concurrency configurable per job type
- Webhook delivery job type with real HTTP calls
