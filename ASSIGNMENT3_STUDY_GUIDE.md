# Assignment 3 Study Guide - Message Queue and Migrations

This guide explains the current Assignment 3 implementation and gives short defense-ready answers.

## 1. What Changed From Assignment 2

Assignment 2 had two gRPC services:

- Doctor Service
- Appointment Service

Assignment 3 adds two infrastructure changes:

- PostgreSQL replaces in-memory/Mongo-style storage.
- NATS Core adds asynchronous domain events.

It also adds a third binary:

- Notification Service

The proto contracts and gRPC methods stay unchanged.

## 2. Current Architecture

```mermaid
flowchart LR
    Client["grpcurl / Postman gRPC reference"] --> Doctor["Doctor Service gRPC :8081"]
    Client --> Appointment["Appointment Service gRPC :8082"]

    Appointment -- "GetDoctor gRPC" --> Doctor

    Doctor --> DoctorDB["PostgreSQL doctor_service DB"]
    Appointment --> AppointmentDB["PostgreSQL appointment_service DB"]

    Doctor -- "doctors.created" --> NATS["NATS Core"]
    Appointment -- "appointments.created" --> NATS
    Appointment -- "appointments.status_updated" --> NATS

    NATS --> Notification["Notification Service stdout JSON logs"]
```

## 3. Services

Doctor Service:

- Owns doctor data.
- Creates doctors.
- Lists doctors.
- Gets doctor by id.
- Enforces valid `full_name`, valid `email`, and unique email.
- Publishes `doctors.created` after successful doctor creation.

Appointment Service:

- Owns appointment data.
- Creates appointments.
- Lists appointments.
- Gets appointment by id.
- Updates appointment status.
- Calls Doctor Service over gRPC to validate `doctor_id`.
- Publishes `appointments.created` and `appointments.status_updated`.

Notification Service:

- Has no database.
- Has no HTTP server.
- Has no gRPC server.
- Subscribes to NATS subjects.
- Prints one structured JSON log per event.

## 4. How To Run

Full Docker Compose stack:

```bash
docker compose up --build
```

If port `5432` is already used locally, this project publishes Postgres on host port `5433`. Inside Docker, services still use `postgres:5432`.

Manual local mode:

```bash
docker compose up -d postgres nats
go run .
```

Separate service mode:

```bash
cd doctor-service
go run .
```

```bash
cd appointment-service
go run .
```

```bash
cd notification-service
go run .
```

## 5. PostgreSQL

Each service has its own database:

- `doctor_service`
- `appointment_service`

Doctor table:

```sql
CREATE TABLE doctors (
    id TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    specialization TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Appointment table:

```sql
CREATE TABLE appointments (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    doctor_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 6. Migrations

Migration files:

- `doctor-service/migrations/000001_create_doctors.up.sql`
- `doctor-service/migrations/000001_create_doctors.down.sql`
- `appointment-service/migrations/000001_create_appointments.up.sql`
- `appointment-service/migrations/000001_create_appointments.down.sql`

Migrations run automatically on service startup before gRPC starts listening.

Manual rollback:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  down 1

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  down 1
```

Manual apply:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  up

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  up
```

Defense answer:

Versioned migrations make schema changes repeatable, auditable, and reversible. Application code does not contain raw table DDL.

## 7. Transactions

`UpdateAppointmentStatus` uses a PostgreSQL transaction:

- Reads the appointment row.
- Locks it with `SELECT ... FOR UPDATE`.
- Updates the status and `updated_at`.
- Commits.

Defense answer:

This protects the read-modify-write operation from concurrent updates. It demonstrates isolation and atomicity.

## 8. Events

All events are JSON.

`doctors.created`:

```json
{
  "event_type": "doctors.created",
  "occurred_at": "2026-05-01T10:23:44Z",
  "id": "doc-id",
  "full_name": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

`appointments.created`:

```json
{
  "event_type": "appointments.created",
  "occurred_at": "2026-05-01T10:24:01Z",
  "id": "appointment-id",
  "title": "Initial cardiac consultation",
  "doctor_id": "doctor-id",
  "status": "new"
}
```

`appointments.status_updated`:

```json
{
  "event_type": "appointments.status_updated",
  "occurred_at": "2026-05-01T10:25:10Z",
  "id": "appointment-id",
  "old_status": "new",
  "new_status": "in_progress"
}
```

Events are published after successful database writes.

## 9. Why NATS Core

NATS Core is simple Pub/Sub.

Why it fits:

- Easy local setup.
- Good for stateless notifications.
- Matches best-effort event publishing required by the assignment.

Trade-off:

- NATS Core does not persist messages.
- If Notification Service is offline, it misses events.
- If a service crashes after DB commit but before publish, the event is lost.

Production fix:

- Outbox pattern.
- NATS JetStream.
- RabbitMQ durable queues and publisher confirms.

## 10. Pub/Sub vs Point-to-Point

Pub/Sub:

- One event can be received by many subscribers.
- Good for notifications, analytics, audit logs, and future integrations.

Point-to-Point:

- One message is consumed by one worker.
- Good for background jobs and task queues.

This project uses Pub/Sub because events describe domain facts that multiple future services may care about.

## 11. Error Handling

Expected gRPC statuses:

- Invalid doctor input: `InvalidArgument`.
- Duplicate doctor email: `AlreadyExists`.
- Missing doctor by id: `NotFound`.
- Missing appointment by id: `NotFound`.
- Invalid appointment status: `InvalidArgument`.
- Invalid status transition: `InvalidArgument`.
- Doctor Service unavailable during appointment creation: `Unavailable`.
- Runtime DB failure: `Internal`.

Broker failures:

- Doctor and Appointment services log publish failures.
- RPC responses are not affected.

Notification Service broker failure:

- Retries with exponential backoff.
- Exits non-zero after repeated startup failure.

## 12. Defense Demo Checklist

1. Start stack:

```bash
docker compose up --build
```

2. Create doctor with grpcurl.

3. Copy returned doctor id.

4. Create appointment using doctor id.

5. Copy returned appointment id.

6. Update appointment status to `in_progress`.

7. Show Notification Service logs for all three event subjects.

8. Show migration rollback and re-apply if asked.

9. Explain consistency trade-off and Outbox solution.

## 13. Common Problems

Port 5432 already used:

- This repo maps container Postgres to host port `5433`.
- Use `localhost:5433` from your machine.
- Docker services use `postgres:5432` internally.

Old containers warning:

```bash
docker compose down --remove-orphans
docker compose up --build
```

Existing old Postgres volume:

If the database init script did not create `doctor_service` and `appointment_service`, reset the compose volume:

```bash
docker compose down -v --remove-orphans
docker compose up --build
```

Use this only when you are okay deleting local demo data.

## 14. Short Defense Script

Say this:

This system has two bounded contexts, Doctor and Appointment, communicating synchronously with gRPC for doctor validation. Each service owns its own PostgreSQL database and schema migrations. After successful writes, services publish domain events to NATS Core. Notification Service subscribes to those events and logs structured JSON. Event publishing is best-effort, so a crash between DB commit and publish can lose an event. In production, I would use the Outbox pattern or a durable broker feature like NATS JetStream or RabbitMQ publisher confirms.

## 15. If Instructor Opens Files

Use this section when the instructor opens code and asks, “What is this file?”

### `docker-compose.yml`

What it does:

- Starts PostgreSQL.
- Starts NATS.
- Builds and starts Doctor Service.
- Builds and starts Appointment Service.
- Builds and starts Notification Service.

What to say:

This file runs the full Assignment 3 system. PostgreSQL and NATS are infrastructure. The three Go services are started as separate containers. Doctor and Appointment services depend on PostgreSQL and NATS. Appointment also depends on Doctor Service because it calls it through gRPC.

Important detail:

- Host machine uses Postgres port `5433`.
- Docker containers use internal address `postgres:5432`.
- Doctor Service is reachable from host on `localhost:8081`.
- Appointment Service is reachable from host on `localhost:8082`.

### `Dockerfile`

What it does:

- Builds four binaries:
  - root combined app
  - doctor-service
  - appointment-service
  - notification-service
- Copies migrations into the container.

What to say:

This is a multi-stage Dockerfile. The first stage compiles Go binaries. The second stage is a smaller Alpine runtime image. Service binaries are stored under `/app/bin`, while migration files are stored under `/app/doctor-service/migrations` and `/app/appointment-service/migrations`.

### `main.go`

What it does:

- Starts Doctor Service.
- Starts Appointment Service.
- Starts Notification Service subscriber.
- Runs all three in one process for local convenience.

What to say:

The assignment requires each service to run separately from its own directory, and those entrypoints exist. This root `main.go` is only a convenience runner for local testing. It starts both gRPC servers and the notification subscriber together.

### `doctor-service/main.go`

What it does:

- Loads `.env`.
- Connects Doctor Service to PostgreSQL.
- Runs Doctor migrations.
- Connects to NATS if available.
- Starts the Doctor gRPC server.

What to say:

This is the real Doctor Service entrypoint for defense. It uses `DATABASE_URL`, `NATS_URL`, and `DOCTOR_SERVICE_ADDR`. Migrations run before the server starts accepting gRPC requests.

### `appointment-service/main.go`

What it does:

- Loads `.env`.
- Connects Appointment Service to PostgreSQL.
- Runs Appointment migrations.
- Connects to Doctor Service through gRPC.
- Connects to NATS if available.
- Starts the Appointment gRPC server.

What to say:

Appointment Service owns appointment data. It does not read the Doctor database directly. It validates doctors by calling Doctor Service over gRPC using `DOCTOR_SERVICE_GRPC_TARGET`.

### `notification-service/main.go`

What it does:

- Loads `.env`.
- Connects to NATS.
- Subscribes to events.
- Keeps running until stopped.

What to say:

Notification Service is intentionally simple. It has no HTTP server, no gRPC server, and no database. Its only job is to consume events and print structured JSON logs.

### `internal/platform/bootstrap/config.go`

What it does:

- Reads environment variables.
- Provides defaults.

What to say:

Configuration is not hardcoded in services. URLs and ports are read from environment variables, which is required by the assignment. Defaults exist only for local development.

Important variables:

- `DATABASE_URL`
- `DOCTOR_DATABASE_URL`
- `APPOINTMENT_DATABASE_URL`
- `NATS_URL`
- `DOCTOR_SERVICE_ADDR`
- `APPOINTMENT_SERVICE_ADDR`
- `DOCTOR_SERVICE_GRPC_TARGET`

### `internal/platform/bootstrap/bootstrap.go`

What it does:

- Loads `.env` files.
- Starts one or more gRPC servers.
- Handles graceful shutdown.

What to say:

This file contains shared startup utilities. It keeps main files small. `RunGRPCServices` starts gRPC servers and stops them gracefully on `SIGINT` or `SIGTERM`.

### `internal/platform/postgres/client.go`

What it does:

- Creates a PostgreSQL connection pool using `pgxpool`.
- Pings the database on startup.

What to say:

This is infrastructure code. If the database is unavailable on startup, the service exits with an error, which matches the assignment requirement.

### `internal/platform/migrations/migrate.go`

What it does:

- Runs `golang-migrate` automatically.
- Uses `file://...` migration paths.
- Ignores `migrate.ErrNoChange`.

What to say:

The services apply migrations on startup before accepting gRPC calls. `ErrNoChange` is not a real failure; it just means the database is already up to date.

### `doctor-service/migrations/000001_create_doctors.up.sql`

What it does:

- Creates the `doctors` table.

What to say:

This file contains the Doctor Service schema. The table has `id`, `full_name`, `specialization`, `email`, and `created_at`. The `email` column is unique, so duplicate emails are rejected at the database level.

### `doctor-service/migrations/000001_create_doctors.down.sql`

What it does:

- Drops the `doctors` table.

What to say:

This is the reversible rollback for the doctor migration. The instructor can test `migrate down 1`, and this file undoes the matching up migration.

### `appointment-service/migrations/000001_create_appointments.up.sql`

What it does:

- Creates the `appointments` table.

What to say:

This schema belongs only to Appointment Service. It stores `doctor_id` as text, but it does not use a foreign key to the Doctor database because services do not share tables or directly depend on each other’s database.

### `appointment-service/migrations/000001_create_appointments.down.sql`

What it does:

- Drops the `appointments` table.

What to say:

This is the reversible rollback for the appointment migration.

### `internal/doctor/model/doctor.go`

What it does:

- Defines the Doctor domain model.

What to say:

This is pure domain code. It does not import PostgreSQL, NATS, gRPC, or protobuf. That preserves Clean Architecture.

### `internal/appointment/model/appointment.go`

What it does:

- Defines Appointment model.
- Defines valid statuses.
- Defines status transition rule.

What to say:

The valid statuses are `new`, `in_progress`, and `done`. The domain rule prevents transition from `done` back to `new`.

### `internal/doctor/usecase/service.go`

What it does:

- Validates doctor input.
- Checks duplicate email.
- Creates doctor in repository.
- Publishes `doctors.created` after successful write.

What to say:

This is the business use case. It depends on interfaces, not concrete Postgres or NATS types. That means infrastructure can be changed without changing domain logic.

Important flow:

1. Trim and validate fields.
2. Check email exists.
3. Generate id.
4. Save doctor.
5. Publish event best-effort.

### `internal/appointment/usecase/service.go`

What it does:

- Validates appointment input.
- Calls Doctor Service through `DoctorLookup`.
- Creates appointment.
- Updates appointment status.
- Publishes appointment events after successful writes.

What to say:

Appointment Service does not access Doctor database. It uses the `DoctorLookup` interface, implemented by a gRPC client. This preserves service boundaries.

Important flow for create:

1. Validate title and doctor id.
2. Call Doctor Service to check doctor exists.
3. Save appointment.
4. Publish `appointments.created`.

Important flow for status update:

1. Load appointment.
2. Parse requested status.
3. Validate transition.
4. Update status in repository transaction.
5. Publish `appointments.status_updated`.

### `internal/doctor/repository/postgres.go`

What it does:

- Implements Doctor repository using PostgreSQL.
- Maps duplicate email constraint to `ErrDoctorEmailAlreadyExists`.
- Maps missing rows to `ErrDoctorNotFound`.

What to say:

This file is infrastructure. It uses `pgx` directly, not an ORM. It satisfies the repository interface used by the Doctor use case.

### `internal/appointment/repository/postgres.go`

What it does:

- Implements Appointment repository using PostgreSQL.
- Uses transaction for status update.
- Uses `SELECT ... FOR UPDATE`.

What to say:

The transaction protects the status update from race conditions. `SELECT ... FOR UPDATE` locks the row until commit, so two concurrent updates cannot silently overwrite each other.

### `internal/doctor/event/nats.go`

What it does:

- Connects Doctor Service to NATS.
- Publishes `doctors.created`.
- Serializes event as JSON.

What to say:

This publisher is behind an interface in the use case. If NATS publish fails, the error is logged and the RPC still succeeds, because event publishing is best-effort.

### `internal/appointment/event/nats.go`

What it does:

- Publishes `appointments.created`.
- Publishes `appointments.status_updated`.
- Serializes events as JSON.

What to say:

Each event includes `event_type`, `occurred_at`, and the required entity fields from the assignment.

### `internal/notification/subscriber/nats.go`

What it does:

- Connects to NATS with retry/backoff.
- Subscribes to:
  - `doctors.created`
  - `appointments.created`
  - `appointments.status_updated`
- Decodes JSON payload.
- Prints structured JSON log lines.

What to say:

NATS Core has no explicit ack. For RabbitMQ, we would acknowledge messages after processing. Here the service simply receives the message and logs it.

### `internal/doctor/transport/grpc/server.go`

What it does:

- Implements Doctor gRPC handlers.
- Maps protobuf requests to use-case input.
- Maps use-case errors to gRPC status codes.

What to say:

The gRPC layer is thin. Business rules are not implemented here; they are in the use case.

### `internal/appointment/transport/grpc/server.go`

What it does:

- Implements Appointment gRPC handlers.
- Maps errors to gRPC status codes.

What to say:

If Doctor Service is unreachable, Appointment Service returns `Unavailable`. If the doctor does not exist, appointment creation returns `FailedPrecondition`.

### `internal/appointment/client/doctor_service.go`

What it does:

- Implements the gRPC client used by Appointment Service to call Doctor Service.
- Adds timeout.
- Converts `NotFound` to `false`.
- Converts unavailable errors to descriptive errors.

What to say:

This file is the synchronous inter-service communication from Assignment 2. It is preserved in Assignment 3.

### `internal/doctor/proto/doctor.proto`

What it does:

- Defines Doctor Service gRPC contract.

What to say:

The proto contract is unchanged from Assignment 2. Generated Go stubs are committed.

### `internal/appointment/proto/appointment.proto`

What it does:

- Defines Appointment Service gRPC contract.

What to say:

The contract is unchanged. Assignment 3 changes infrastructure, not public gRPC API.

### `Makefile`

What it does:

- Provides shortcuts for running tests, compose stack, migrations, services, and verification.

What to say:

This is not part of business logic. It exists to make defense commands repeatable and reduce manual mistakes.

### `scripts/verify_assignment3.sh`

What it does:

- Checks no active Mongo references.
- Checks DDL is only in migrations.
- Checks required service entrypoints.
- Checks event subjects.
- Runs Go tests.
- Validates Docker Compose config.

What to say:

This is my local quality gate before submission.

### `postman_collection.json`

What it does:

- Documents gRPC request payloads and grpcurl commands.

What to say:

The project is gRPC-only, not REST. This collection is a Postman reference artifact. The safest executable demo remains `grpcurl` unless Postman gRPC mode is configured with the proto files.

## 16. File Opening Quick Answers

If you forget everything, use these short answers.

`usecase` files:

Business logic. They depend on interfaces.

`repository` files:

PostgreSQL implementation of data access.

`event` files:

NATS publisher code.

`transport/grpc` files:

gRPC handlers and error-code mapping.

`client` files:

Outbound gRPC client from Appointment Service to Doctor Service.

`model` files:

Pure domain objects and rules.

`proto` files:

Public gRPC contracts.

`migrations` files:

Versioned schema changes.

`notification` files:

NATS subscriber and structured logger.

`platform` files:

Shared infrastructure helpers: config, PostgreSQL, migrations, server startup.

## 17. Questions Instructor May Ask

Why no foreign key from appointments to doctors?

Because Doctor and Appointment are separate bounded contexts. Appointment stores `doctor_id`, but validates it by gRPC. It must not directly depend on Doctor database tables.

Why publish event after DB write?

Because the event should represent a real saved state change. If publishing happened before the DB write, an event could announce data that does not exist.

What happens if publish fails?

The error is logged. RPC success is not affected. This is best-effort asynchronous communication.

What is the weakness of best-effort publish?

A service can commit to the DB and crash before publishing. The event is lost.

How to fix event loss?

Use the Outbox pattern. Save the event in the same DB transaction as the business change, then a background worker publishes it reliably.

Why NATS instead of RabbitMQ?

NATS Core is simpler and enough for stateless notifications. RabbitMQ is better when durable queues and stronger delivery guarantees are required.

What would change if switching to RabbitMQ?

Publisher would declare a fanout exchange, publish to that exchange, and Notification Service would bind an exclusive queue and acknowledge messages after logging.

What does `SELECT ... FOR UPDATE` do?

It locks the selected row inside a transaction so concurrent updates wait instead of overwriting each other.

What are ACID properties?

Atomicity means all-or-nothing. Consistency means constraints keep valid data. Isolation means transactions do not corrupt each other. Durability means committed data survives crashes.

Why use `pgx` directly?

The assignment allows `database/sql` or `pgx/v5` directly and forbids ORMs. This project uses `pgx/v5`.

Why does Notification Service have no database?

The assignment says its only responsibility is to subscribe to events and log them.

Why does Notification Service not expose a port?

It is not an API service. It only consumes broker messages.

What is `occurred_at`?

It is the UTC RFC3339 timestamp when the publishing service created the event.

What is `time` in notification logs?

It is the UTC RFC3339 timestamp when Notification Service received and processed the message.
