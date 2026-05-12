# Medical Scheduling Platform - Assignment 4

This version extends the Assignment 3 Medical Scheduling Platform with Redis caching, Redis-backed gRPC rate limiting, a Notification Service background job queue, and a simulated external notification gateway.

Unchanged from Assignment 3: gRPC contracts, generated stubs, domain models, PostgreSQL schemas/migrations, NATS subjects, and the service-directory startup flow.

## Architecture

```mermaid
flowchart LR
  Client["grpcurl / gRPC client"]
  Doctor["Doctor Service :8081"]
  Appointment["Appointment Service :8082"]
  Notify["Notification Service"]
  Gateway["Mock Notification Gateway :8080"]
  DoctorDB[("PostgreSQL doctor_service")]
  AppointmentDB[("PostgreSQL appointment_service")]
  Redis[("Redis")]
  NATS[("NATS Core")]

  Client -->|"gRPC"| Doctor
  Client -->|"gRPC"| Appointment
  Appointment -->|"gRPC doctor validation"| Doctor
  Doctor -->|"SQL"| DoctorDB
  Appointment -->|"SQL"| AppointmentDB
  Doctor -->|"cache + rate limiter"| Redis
  Appointment -->|"cache + rate limiter"| Redis
  Notify -->|"idempotency keys"| Redis
  Doctor -->|"doctors.created"| NATS
  Appointment -->|"appointments.created / status_updated"| NATS
  NATS -->|"subscribe"| Notify
  Notify -->|"POST /notify jobs"| Gateway
```

## Services

- `doctor-service`: owns doctors, caches `GetDoctor` and `ListDoctors`, publishes `doctors.created`.
- `appointment-service`: owns appointments, validates doctors through gRPC, caches appointment reads, publishes appointment events.
- `notification-service`: subscribes to NATS, logs every event, enqueues completion jobs for `appointments.status_updated` where `new_status = "done"`.
- `mock-gateway`: exposes `POST /notify`, simulates idempotency and random transient HTTP 503 failures.

## Cache Strategy

| Operation | Strategy | Redis key | Invalidation |
| --- | --- | --- | --- |
| `GetDoctor` | Cache-aside | `doctor:<id>` | TTL |
| `ListDoctors` | Cache-aside | `doctors:list` | deleted after `CreateDoctor` |
| `GetAppointment` | Cache-aside | `appointment:<id>` | TTL or refreshed after status update |
| `ListAppointments` | Cache-aside | `appointments:list` | deleted after create/status update |
| `CreateDoctor` | write-through invalidation | `doctors:list` | delete after DB write succeeds |
| `CreateAppointment` | write-around invalidation | `appointments:list` | delete after DB write succeeds |
| `UpdateAppointmentStatus` | write-through update/invalidation | `appointment:<id>`, `appointments:list` | set updated item, delete list |

Cache misses and Redis errors never fail RPCs; the services fall through to PostgreSQL. Cache write/delete failures are logged and the gRPC response still returns. TTL is `CACHE_TTL_SECONDS`, default `60`.

The stale-read window is limited by TTL and explicit invalidation. List reads may be stale until the write invalidation happens; if Redis is unavailable, all reads use PostgreSQL directly.

## Rate Limiting

Doctor and Appointment services use a Redis-backed sliding-window counter implemented as a `UnaryServerInterceptor`. Keys use `rate:<service>:<client_ip>`.

The interceptor stores timestamps in a Redis sorted set, removes entries older than 60 seconds, and allows up to `RATE_LIMIT_RPM` requests per client IP. Default: `100`.

When exceeded, the RPC returns `codes.ResourceExhausted` with a retry-after value. If Redis fails at runtime, the limiter logs the error and fails open so the service remains available.

Central Redis counters avoid two horizontal-scaling problems: per-instance counters can be bypassed by load balancing, and instance restarts reset local counters. Redis gives all instances one shared request window.

## Job Queue

Notification Service has separate packages for subscriber, logger, and job queue. The subscriber logs each event first, then passes done-status events to the queue.

Each job includes:

- idempotency key: SHA-256 hex of `event_type + id + occurred_at`
- appointment id, doctor id, original event timestamp
- channel: `email`
- recipient: `patient@clinic.kz`
- message: `Your appointment <id> with doctor <doctor_id> is complete.`

The queue uses a buffered Go channel sized at `WORKER_POOL_SIZE * 10` and a worker pool sized by `WORKER_POOL_SIZE` (default `3`). If the channel is full, the job is dead-lettered and the idempotency claim is released.

Idempotency keys are stored in Redis as `notification:job:<sha>`. `SETNX processing` claims a job, success stores `done` for 24 hours, and duplicate `done` events are dropped without a second gateway call.

Gateway failures with HTTP 503 or network errors retry with exponential backoff. Defaults are `JOB_MAX_RETRIES=3` and `JOB_BACKOFF_SECONDS=1,2,4`. After the configured attempts fail, the worker writes a JSON `dead_letter` line to stderr and deletes the processing key so a manual replay can retry.

```mermaid
stateDiagram-v2
  [*] --> Logged: NATS event received
  Logged --> Ignored: new_status != done
  Logged --> Duplicate: Redis key == done
  Logged --> Enqueued: SETNX processing
  Enqueued --> Processing: worker receives job
  Processing --> Success: POST /notify HTTP 200
  Processing --> Retry: HTTP 503 or network error
  Retry --> Processing: backoff expires
  Retry --> DeadLetter: attempts exhausted
  Success --> [*]: Redis key = done for 24h
  Duplicate --> [*]
  DeadLetter --> [*]
```

## Infrastructure

Start everything:

```bash
docker compose up --build
```

Start only infrastructure for local `go run`:

```bash
docker compose up -d postgres nats redis
```

Run gateway locally:

```bash
cd mock-gateway
GATEWAY_PORT=8080 go run .
```

Run Doctor Service:

```bash
cd doctor-service
DATABASE_URL="postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
REDIS_URL="redis://localhost:6379" \
CACHE_TTL_SECONDS=60 \
RATE_LIMIT_RPM=100 \
DOCTOR_SERVICE_ADDR=":8081" \
go run .
```

For assignment-style environment names, `GRPC_PORT=8081` is also supported when `DOCTOR_SERVICE_ADDR` is not set.

Run Appointment Service:

```bash
cd appointment-service
DATABASE_URL="postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
REDIS_URL="redis://localhost:6379" \
CACHE_TTL_SECONDS=60 \
RATE_LIMIT_RPM=100 \
APPOINTMENT_SERVICE_ADDR=":8082" \
DOCTOR_SERVICE_GRPC_TARGET="127.0.0.1:8081" \
go run .
```

For assignment-style environment names, `GRPC_PORT=8082` is also supported when `APPOINTMENT_SERVICE_ADDR` is not set.

Run Notification Service:

```bash
cd notification-service
NATS_URL="nats://localhost:4222" \
REDIS_URL="redis://localhost:6379" \
GATEWAY_URL="http://localhost:8080" \
WORKER_POOL_SIZE=3 \
JOB_MAX_RETRIES=3 \
JOB_BACKOFF_SECONDS=1,2,4 \
go run .
```

Startup order for defense: PostgreSQL, NATS, Redis, Mock Gateway, Doctor Service, Appointment Service, Notification Service.

## Environment Variables

| Service | Variable | Default / example |
| --- | --- | --- |
| all | `REDIS_URL` | `redis://localhost:6379` |
| Doctor / Appointment | `CACHE_TTL_SECONDS` | `60` |
| Doctor / Appointment | `RATE_LIMIT_RPM` | `100` |
| Doctor / Appointment | `DATABASE_URL` | PostgreSQL URL |
| all publishers/subscribers | `NATS_URL` | `nats://localhost:4222` |
| Doctor | `DOCTOR_SERVICE_ADDR` | `:8081` |
| Doctor / Appointment | `GRPC_PORT` | assignment-compatible alias when service-specific address is unset |
| Appointment | `APPOINTMENT_SERVICE_ADDR` | `:8082` |
| Appointment | `DOCTOR_SERVICE_GRPC_TARGET` | `127.0.0.1:8081` |
| Notification | `GATEWAY_URL` | `http://localhost:8080` |
| Notification | `WORKER_POOL_SIZE` | `3` |
| Notification | `JOB_MAX_RETRIES` | `3` |
| Notification | `JOB_BACKOFF_SECONDS` | `1,2,4` |
| Mock Gateway | `GATEWAY_PORT` | `8080` |

## Event Contract

The NATS subjects stay unchanged:

- `doctors.created`
- `appointments.created`
- `appointments.status_updated`

The gRPC/proto/domain models stay unchanged. `appointments.status_updated` keeps the original fields and adds one backwards-compatible JSON field, `doctor_id`, because the Assignment 4 job contract requires the Notification Service to read doctor id from the event payload. Existing consumers can ignore the extra field.

## Testing

Run automated tests:

```bash
GOCACHE=/private/tmp/med-go-gocache go test ./...
```

Validate Docker Compose:

```bash
docker compose config
```

Use `grpcurl_commands.md` for defense commands, including cache-hit, rate-limit, job queue, idempotency, and dead-letter checks.
