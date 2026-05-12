# Assignment 4 Study Guide - Caching and Background Jobs

This guide explains the current Assignment 4 implementation and gives defense-ready answers for caching, Redis, rate limiting, background jobs, retries, idempotency, and the mock external API.

## 1. Short Project Summary

This project is a Medical Scheduling Platform with four runnable binaries:

- `doctor-service`
- `appointment-service`
- `notification-service`
- `mock-gateway`

Assignment 4 adds infrastructure concerns on top of Assignment 3:

- Redis cache for Doctor and Appointment read operations.
- Redis-backed gRPC rate limiter for Doctor and Appointment endpoints.
- Notification Service background job queue using Go channels and goroutines.
- Idempotency keys stored in Redis.
- Retry with exponential backoff for external gateway calls.
- Mock Notification Gateway over HTTP.

What stayed unchanged:

- Domain models.
- Proto contracts and generated stubs.
- PostgreSQL schemas and migration files.
- Main gRPC behavior and status codes.
- NATS subjects.
- Clean Architecture dependency direction.

Defense one-liner:

The assignment keeps the business model and gRPC APIs stable, then adds Redis, rate limiting, and asynchronous job processing as infrastructure around the existing services.

## 2. Architecture

```mermaid
flowchart LR
    Client["grpcurl / gRPC client"]
    Doctor["Doctor Service :8081"]
    Appointment["Appointment Service :8082"]
    Notification["Notification Service"]
    Gateway["Mock Gateway :8080"]
    DoctorDB[("PostgreSQL doctor_service")]
    AppointmentDB[("PostgreSQL appointment_service")]
    Redis[("Redis")]
    NATS[("NATS Core")]

    Client -->|"gRPC"| Doctor
    Client -->|"gRPC"| Appointment
    Appointment -->|"validate doctor_id over gRPC"| Doctor
    Doctor -->|"SQL"| DoctorDB
    Appointment -->|"SQL"| AppointmentDB
    Doctor -->|"cache + rate limit"| Redis
    Appointment -->|"cache + rate limit"| Redis
    Notification -->|"idempotency keys"| Redis
    Doctor -->|"doctors.created"| NATS
    Appointment -->|"appointments.created"| NATS
    Appointment -->|"appointments.status_updated"| NATS
    NATS -->|"subscribe"| Notification
    Notification -->|"POST /notify"| Gateway
```

## 3. Service Responsibilities

Doctor Service:

- Owns doctor data.
- Uses PostgreSQL as source of truth.
- Caches `GetDoctor` and `ListDoctors` in Redis.
- Invalidates `doctors:list` after `CreateDoctor`.
- Publishes `doctors.created`.
- Applies Redis-backed rate limiting through a gRPC interceptor.

Appointment Service:

- Owns appointment data.
- Uses PostgreSQL as source of truth.
- Calls Doctor Service over gRPC to validate doctor ids.
- Caches `GetAppointment` and `ListAppointments` in Redis.
- Invalidates `appointments:list` after `CreateAppointment`.
- Refreshes `appointment:<id>` and invalidates `appointments:list` after `UpdateAppointmentStatus`.
- Publishes `appointments.created` and `appointments.status_updated`.
- Applies Redis-backed rate limiting through a gRPC interceptor.

Notification Service:

- Subscribes to NATS events.
- Logs every event as JSON.
- Enqueues background jobs only for `appointments.status_updated` with `new_status = "done"`.
- Uses Redis for idempotency.
- Calls Mock Gateway with retry and backoff.
- Writes dead-letter logs to stderr after repeated failures.

Mock Gateway:

- Exposes `POST /notify`.
- Returns `200 {"status":"accepted"}` for new idempotency keys.
- Returns `200 {"status":"duplicate"}` for repeated keys.
- Randomly returns `503` about 20 percent of the time for transient failure simulation.
- Logs every request to stdout as JSON.

## 4. Key Files

Caching:

- `internal/doctor/usecase/service.go`
- `internal/doctor/cache/redis.go`
- `internal/appointment/usecase/service.go`
- `internal/appointment/cache/redis.go`

Rate limiting:

- `internal/platform/middleware/ratelimit.go`
- `internal/doctor/app/app.go`
- `internal/appointment/app/app.go`

Redis setup:

- `internal/platform/redis/client.go`
- `internal/platform/bootstrap/config.go`

Events:

- `internal/doctor/event/nats.go`
- `internal/appointment/event/nats.go`

Notification Service:

- `internal/notification/subscriber/nats.go`
- `internal/notification/logger/logger.go`
- `internal/notification/jobqueue/jobqueue.go`
- `notification-service/main.go`

Mock Gateway:

- `mock-gateway/main.go`

Infrastructure and docs:

- `docker-compose.yml`
- `Dockerfile`
- `Makefile`
- `README.md`
- `grpcurl_commands.md`

## 5. Redis Cache Strategy

Doctor Service:

| Operation | Strategy | Key | Behavior |
| --- | --- | --- | --- |
| `GetDoctor` | Cache-aside | `doctor:<id>` | Try Redis first, fall back to DB, then set cache |
| `ListDoctors` | Cache-aside | `doctors:list` | Try Redis first, fall back to DB, then set cache |
| `CreateDoctor` | Write-through invalidation | `doctors:list` | Delete list key after DB write succeeds |

Appointment Service:

| Operation | Strategy | Key | Behavior |
| --- | --- | --- | --- |
| `GetAppointment` | Cache-aside | `appointment:<id>` | Try Redis first, fall back to DB, then set cache |
| `ListAppointments` | Cache-aside | `appointments:list` | Try Redis first, fall back to DB, then set cache |
| `CreateAppointment` | Write-around invalidation | `appointments:list` | Delete list key after DB write succeeds |
| `UpdateAppointmentStatus` | Write-through update/invalidation | `appointment:<id>`, `appointments:list` | Set updated item, delete list key |

Defense answer:

Cache-aside means the application checks Redis on reads, loads from PostgreSQL on a miss, and then writes the result back to Redis. PostgreSQL remains the source of truth.

## 6. Why Cache-Aside Was Used

Cache-aside is simple and safe for this project:

- If Redis is unavailable, reads still work through PostgreSQL.
- Cache data can expire naturally through TTL.
- The application controls when keys are invalidated.
- No business logic depends on Redis.

Trade-off:

There can be short stale-read windows until invalidation or TTL expiry. The implementation reduces this by deleting list keys immediately after successful writes and refreshing updated appointment items.

## 7. Cache Failure Behavior

Required behavior:

- Redis unavailable on startup: log warning and continue.
- Cache miss: fall through to PostgreSQL.
- Cache read failure: log and fall through to PostgreSQL.
- Cache write/delete failure: log and return the gRPC response anyway.

Defense answer:

Caching is a performance optimization, not a correctness dependency. The database remains authoritative, so Redis failures must not crash the service or fail normal RPC responses.

## 8. CacheRepository Interface

The use cases define cache interfaces:

- Doctor: `CacheRepository` for doctor item/list get/set/delete.
- Appointment: `CacheRepository` for appointment item/list get/set/delete.

Redis implementations live in infrastructure packages:

- `internal/doctor/cache`
- `internal/appointment/cache`

Defense answer:

The use case depends only on an interface, not Redis. Redis is injected from the app wiring layer, which preserves Clean Architecture dependency direction.

## 9. Rate Limiter

Implementation:

- File: `internal/platform/middleware/ratelimit.go`
- Type: gRPC `UnaryServerInterceptor`
- Algorithm: Redis-backed sliding window.
- Redis data structure: sorted set.
- Key pattern: `rate:<service>:<client_ip>`
- Default limit: `100` requests per minute.
- Env var: `RATE_LIMIT_RPM`

How it works:

1. Extract client IP from gRPC peer info.
2. Remove sorted-set entries older than 60 seconds.
3. Add the current request timestamp.
4. Count entries in the window.
5. If count exceeds limit, return `codes.ResourceExhausted`.
6. If Redis fails, log and allow the request.

Defense answer:

The sliding-window counter is more accurate than a fixed window because it always checks the last 60 seconds, not just the current clock minute.

## 10. Why Rate Limiting Is an Interceptor

Defense answer:

The rate limiter is a gRPC interceptor so endpoint handlers stay clean. It protects all unary RPCs consistently without duplicating rate-limit code in every handler.

## 11. Redis-Backed Rate Limiting in Distributed Systems

Problem with local per-instance limiters:

- A user can bypass limits if traffic is load-balanced across many service instances.
- Counters reset when an instance restarts.
- Each instance has only a partial view of traffic.

Why Redis helps:

- Redis gives all service instances one shared counter.
- Limits remain consistent across horizontal scaling.
- The counter survives individual application restarts.

## 12. Background Job Trigger

The job queue is triggered by:

- Subject: `appointments.status_updated`
- Condition: `new_status == "done"`

The subscriber always logs the event first, then calls the job queue.

Defense answer:

The event consumer stays responsible for receiving and routing messages. The job queue separately owns idempotency, retries, workers, and gateway calls.

## 13. Job Contract

Each notification job contains:

- `idempotency_key`
- `appointment_id`
- `doctor_id`
- `occurred_at`
- `channel`
- `recipient`
- `message`

Example message:

```text
Your appointment appt-1 with doctor doc-1 is complete.
```

The idempotency key is:

```text
SHA-256(event_type + id + occurred_at)
```

Defense answer:

The key is deterministic, so replaying the same event produces the same key and prevents duplicate external gateway calls.

## 14. Event Doctor ID Note

Assignment 4 says the job needs `doctor_id` from the status-updated event payload. The original Assignment 3 event did not include it.

This implementation keeps proto/domain/database unchanged and adds `doctor_id` as an extra JSON field in `appointments.status_updated`.

Defense answer:

This is backwards-compatible because JSON consumers can ignore unknown fields. It avoids adding a new cross-service lookup from Notification Service and keeps the job self-contained.

## 15. Job Queue Lifecycle

```mermaid
stateDiagram-v2
    [*] --> EventLogged
    EventLogged --> Ignored: new_status != done
    EventLogged --> Duplicate: Redis key == done
    EventLogged --> Enqueued: SETNX processing
    Enqueued --> Processing: worker receives job
    Processing --> Success: gateway HTTP 200
    Processing --> Retry: gateway HTTP 503 or network error
    Retry --> Processing: backoff expires
    Retry --> DeadLetter: attempts exhausted
    Success --> DoneKey: Redis key = done, TTL 24h
    DoneKey --> [*]
    Duplicate --> [*]
    DeadLetter --> [*]
```

Log statuses:

- `enqueued`
- `processing`
- `success`
- `retry`
- `dead_letter`

## 16. Worker Pool

Implementation:

- File: `internal/notification/jobqueue/jobqueue.go`
- Queue: buffered Go channel.
- Workers: goroutines reading from the channel.
- Pool size env var: `WORKER_POOL_SIZE`
- Default pool size: `3`
- Channel size: `WORKER_POOL_SIZE * 10`

Defense answer:

The worker pool decouples event consumption from slow external HTTP calls. The Notification Service can receive events quickly while workers process gateway calls asynchronously.

## 17. Backpressure

If the job channel is full:

- The job is not blocked forever.
- The idempotency processing claim is deleted.
- A `dead_letter` log is written.

Defense answer:

This makes overload visible and prevents the service from hanging. A production system would use a durable queue or broker dead-letter queue.

## 18. Idempotency

Redis key:

```text
notification:job:<sha256>
```

Lifecycle:

- Before enqueue: `SETNX key processing`.
- On success: `SET key done EX 24h`.
- On duplicate event: if value is `done`, drop without another gateway call.
- On dead-letter: delete `processing` so replay can retry.

Defense answer:

Idempotency protects the external API from duplicate calls caused by retries, replays, or duplicate broker messages.

## 19. Retry and Backoff

Defaults:

- `JOB_MAX_RETRIES=3`
- `JOB_BACKOFF_SECONDS=1,2,4`

Transient failures:

- HTTP 503.
- Network errors.

Behavior:

- Attempt 1 fails, log `retry`, wait 1 second.
- Attempt 2 fails, log `retry`, wait 2 seconds.
- Attempt 3 fails, log `retry`, wait 4 seconds.
- Then write `dead_letter` to stderr.

Defense answer:

Exponential backoff gives transient systems time to recover and avoids hammering the external API during an outage.

## 20. Dead-Letter Strategy

Current implementation:

- Writes a structured JSON line to stderr.
- Does not crash the worker.
- Deletes the processing idempotency key so manual replay can retry later.

Production improvement:

- Send failed jobs to a durable dead-letter queue.
- Add alerting.
- Add an admin replay tool.
- Store failure metadata for inspection.

Defense answer:

For the assignment, stderr JSON is enough to prove dead-letter behavior. In production, stderr would be replaced by a durable DLQ and monitoring alerts.

## 21. Mock Gateway

Endpoint:

```http
POST /notify
```

Request:

```json
{
  "idempotency_key": "...",
  "channel": "email",
  "recipient": "patient@clinic.kz",
  "message": "Your appointment appt-1 with doctor doc-1 is complete."
}
```

Responses:

- New key: `200 {"status":"accepted"}`
- Duplicate key: `200 {"status":"duplicate"}`
- Random transient failure: `503 {"status":"unavailable"}`

Defense answer:

The gateway simulates a third-party API. It supports idempotency and transient failures so retry behavior can be demonstrated during defense.

## 22. Environment Variables

Core:

```text
DATABASE_URL
NATS_URL
REDIS_URL
```

Doctor and Appointment:

```text
CACHE_TTL_SECONDS=60
RATE_LIMIT_RPM=100
DOCTOR_SERVICE_ADDR=:8081
APPOINTMENT_SERVICE_ADDR=:8082
GRPC_PORT=8081 or 8082
```

Appointment:

```text
DOCTOR_SERVICE_GRPC_TARGET=127.0.0.1:8081
```

Notification:

```text
GATEWAY_URL=http://localhost:8080
WORKER_POOL_SIZE=3
JOB_MAX_RETRIES=3
JOB_BACKOFF_SECONDS=1,2,4
```

Gateway:

```text
GATEWAY_PORT=8080
```

## 23. How To Run Everything

Full stack:

```bash
docker compose up --build
```

Infrastructure only:

```bash
docker compose up -d postgres nats redis
```

Gateway:

```bash
cd mock-gateway
GATEWAY_PORT=8080 go run .
```

Doctor Service:

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

Appointment Service:

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

Notification Service:

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

Startup order:

1. PostgreSQL.
2. NATS.
3. Redis.
4. Mock Gateway.
5. Doctor Service.
6. Appointment Service.
7. Notification Service.

## 24. Defense Checkpoint 1 - Cache Hit

Run Redis monitor:

```bash
redis-cli MONITOR
```

Call `GetDoctor` twice:

```bash
grpcurl -plaintext \
  -d '{"id":"PUT_DOCTOR_ID_HERE"}' \
  127.0.0.1:8081 doctor.DoctorService/GetDoctor
```

Expected:

- First call: Redis `GET doctor:<id>`, then database read, then Redis `SET doctor:<id>`.
- Second call: Redis `GET doctor:<id>` returns cached value.

Defense answer:

The second call is faster and avoids a PostgreSQL query because the data is already in Redis.

## 25. Defense Checkpoint 2 - Rate Limiter

Start with low limit:

```bash
RATE_LIMIT_RPM=2 docker compose up --build
```

Call an endpoint three times within one minute:

```bash
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
```

Expected:

```text
Code: ResourceExhausted
Message: rate limit exceeded, retry after ... seconds
```

Defense answer:

The limiter is enforced before the handler is called because it is a unary server interceptor.

## 26. Defense Checkpoint 3 - Job Queue and Gateway

Complete an appointment:

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "done"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected Notification Service logs:

```json
{"time":"...","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"...","id":"PUT_APPOINTMENT_ID_HERE","doctor_id":"PUT_DOCTOR_ID_HERE","old_status":"in_progress","new_status":"done"}}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"enqueued"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"processing"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"success"}
```

Expected Gateway log:

```json
{"time":"...","path":"/notify","body":{"idempotency_key":"...","channel":"email","recipient":"patient@clinic.kz","message":"Your appointment PUT_APPOINTMENT_ID_HERE with doctor PUT_DOCTOR_ID_HERE is complete."}}
```

## 27. Defense Checkpoint 4 - Idempotency

Replay the same event with the same `event_type`, `id`, and `occurred_at`.

Expected:

- Notification Service logs the event.
- Redis key already has value `done`.
- Job is dropped as duplicate.
- Mock Gateway is not called again.

Defense answer:

The same event creates the same SHA-256 idempotency key, so duplicate processing is prevented.

## 28. Defense Checkpoint 5 - Dead Letter

Stop the gateway:

```bash
docker compose stop mock-gateway
```

Complete a different appointment.

Expected:

- Attempt 1 processing.
- Attempt 1 retry.
- Attempt 2 processing.
- Attempt 2 retry.
- Attempt 3 processing.
- Attempt 3 retry.
- Dead-letter JSON line to stderr.
- Notification Service keeps running.

Defense answer:

The worker treats gateway outage as transient, retries with backoff, and dead-letters only after the configured attempts are exhausted.

## 29. Important Error Handling

Redis unavailable:

- Doctor/Appointment services log warning.
- Cache and rate limiter are disabled.
- PostgreSQL reads/writes still work.

NATS unavailable:

- Service logs warning.
- RPC still succeeds.
- Event publishing is best effort.

Gateway unavailable:

- Notification worker retries.
- Dead-letter after max attempts.
- Worker does not crash.

Cache miss:

- Not an error.
- Fall through to database.

Rate limit exceeded:

- Return `codes.ResourceExhausted`.

## 30. Clean Architecture Defense

Domain layer:

- No Redis.
- No NATS.
- No HTTP.
- No PostgreSQL-specific code.

Use case layer:

- Defines repository and cache interfaces.
- Contains business workflow.
- Does not import go-redis or net/http.

Infrastructure layer:

- PostgreSQL repository implementations.
- Redis cache implementations.
- NATS publishers/subscribers.
- HTTP gateway client inside job queue.
- gRPC server/interceptor wiring.

Defense answer:

Infrastructure dependencies point inward through interfaces. The use cases define what they need, and infrastructure provides implementations.

## 31. Common Oral Defense Questions

Question: Why use Redis?

Answer: Redis reduces database load and improves latency for repeated read operations. It is fast, supports TTL, and works well for shared counters and idempotency keys.

Question: What happens if Redis is down?

Answer: The services log a warning and continue. Caching and rate limiting are disabled, but PostgreSQL remains the source of truth.

Question: What is cache-aside?

Answer: The application checks cache first. On a miss, it reads from the database and writes the result into the cache.

Question: Why invalidate list keys on writes?

Answer: List caches become stale when new data is created or an appointment status changes. Deleting the list key forces the next list read to reload from PostgreSQL.

Question: Why is the rate limiter not inside handlers?

Answer: An interceptor applies the same protection to all unary RPCs without duplicating logic or modifying handler code.

Question: Why use a Redis sorted set for rate limiting?

Answer: Sorted sets store timestamps as scores. This makes it easy to remove old requests and count only requests inside the last 60 seconds.

Question: What is idempotency?

Answer: Idempotency means processing the same event multiple times has the same effect as processing it once. Here, duplicate events do not create duplicate external gateway calls.

Question: Why use SHA-256 for idempotency keys?

Answer: It creates a deterministic, compact key from event fields. The same event always produces the same key.

Question: Why use retries?

Answer: External APIs can fail temporarily. Retrying with backoff gives the gateway time to recover.

Question: What is dead-letter logging?

Answer: It records jobs that could not be completed after retries. In this assignment it writes JSON to stderr; in production it would go to a DLQ.

Question: Why is the Mock Gateway needed?

Answer: It simulates an external notification provider and lets us test success, duplicate idempotency, and transient 503 failures.

Question: Why add `doctor_id` to `appointments.status_updated`?

Answer: The job contract requires doctor id from the event payload. Adding it as an extra JSON field is backwards-compatible and avoids a new lookup dependency in Notification Service.

Question: What consistency issues does caching introduce?

Answer: Cached data can be stale. TTL and invalidation reduce the stale window, but Redis is eventually consistent with PostgreSQL from the application perspective.

Question: How would Redis Cluster change things?

Answer: It improves availability and scalability, but network partitions and replication lag can still cause stale reads. Key naming and hash slots also become operational concerns.

## 32. Quick Defense Script

1. Start stack:

```bash
docker compose up --build
```

2. Create a doctor.

3. Call `GetDoctor` twice while Redis monitor is open.

4. Explain cache-aside and TTL.

5. Lower `RATE_LIMIT_RPM` and show `ResourceExhausted`.

6. Create an appointment.

7. Update appointment to `done`.

8. Show Notification Service event log.

9. Show job queue logs.

10. Show Mock Gateway `/notify` log.

11. Replay the same event and show no second gateway call.

12. Stop gateway, complete another appointment, and show retry plus dead-letter logs.

## 33. Final Grade Checklist

Caching:

- Redis client: `github.com/redis/go-redis/v9`.
- `REDIS_URL`.
- `CACHE_TTL_SECONDS`.
- Cache-aside reads.
- Correct key names.
- Invalidation after DB writes.
- Cache failures are best effort.

Rate limiting:

- Unary interceptor.
- Redis sorted-set sliding window.
- `RATE_LIMIT_RPM`.
- `codes.ResourceExhausted`.

Background jobs:

- Worker pool with goroutines and channel.
- `WORKER_POOL_SIZE`.
- Redis idempotency.
- SHA-256 deterministic key.
- Retry with backoff.
- Dead-letter JSON stderr.
- Mock Gateway HTTP integration.

Clean Architecture:

- No Redis in domain models.
- No Redis in gRPC handlers.
- Cache injected through interfaces.
- Subscriber, logger, and job queue are separate packages.

Documentation:

- README explains architecture.
- Cache strategy documented.
- Rate limiter algorithm documented.
- Job lifecycle documented.
- grpcurl commands include expected logs.

## 34. Most Important Sentences To Memorize

Caching:

Redis is used as a best-effort performance layer. PostgreSQL remains the source of truth, so Redis failures never break normal RPC behavior.

Rate limiting:

The limiter is a Redis-backed sliding window implemented as a gRPC unary interceptor, which keeps handler code clean and works across multiple service instances.

Job queue:

The Notification Service logs events, then enqueues only completed appointment events into a worker pool. Workers call the external gateway with Redis idempotency, retry transient failures, and dead-letter failed jobs.

Idempotency:

The idempotency key is deterministic from `event_type + id + occurred_at`, so replaying the same event cannot call the gateway twice after a successful job.

Clean Architecture:

Infrastructure implementations depend on interfaces defined by the use case layer; domain and gRPC contracts remain unchanged.
