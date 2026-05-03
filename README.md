# Medical Scheduling Platform - Assignment 3

This version extends the Assignment 2 gRPC-only Medical Scheduling Platform with PostgreSQL persistence, versioned golang-migrate migrations, NATS Core events, and a standalone Notification Service.

The proto contracts, generated stubs, domain models, business rules, and gRPC status mapping are intentionally unchanged from Assignment 2. The infrastructure layer changed from MongoDB/in-memory storage to PostgreSQL repositories and asynchronous event publishing.

## Architecture

![alt text](/docs/image.png)

## Broker Choice

This project uses **NATS Core**.

Reason: Assignment 3 allows best-effort, fire-and-forget event publishing. NATS Core is simpler to run locally, has minimal setup, and is enough for the required Notification Service demo.

Trade-off: NATS Core does not persist messages. If a source service commits to PostgreSQL and then crashes before publishing, the event is lost. In production, this would be addressed with the Outbox pattern, NATS JetStream, or RabbitMQ with durable queues and publisher confirms.

## Services

- `doctor-service`: owns doctor profiles and publishes `doctors.created`.
- `appointment-service`: owns appointments, validates doctors through Doctor Service gRPC, and publishes appointment events.
- `notification-service`: subscribes to all events and prints one structured JSON log line per message.

Notification Service does not expose HTTP/gRPC, does not call other services, and does not use a database.

## Environment Variables

Doctor Service:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable
NATS_URL=nats://localhost:4222
DOCTOR_SERVICE_ADDR=:8081
```

Appointment Service:

```bash
DATABASE_URL=postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable
NATS_URL=nats://localhost:4222
APPOINTMENT_SERVICE_ADDR=:8082
DOCTOR_SERVICE_GRPC_TARGET=127.0.0.1:8081
```

Notification Service:

```bash
NATS_URL=nats://localhost:4222
```

The root combined binary also supports `DOCTOR_DATABASE_URL` and `APPOINTMENT_DATABASE_URL`, but the assignment defense flow should use the three service directories.

## Infrastructure Setup

Start the full stack:

```bash
docker compose up --build
```

The compose file starts:

- `doctor_service`
- `appointment_service`
- `doctor-service`
- `appointment-service`
- `notification-service`
- `nats`

## Migrations

Migrations are stored inside each service directory:

- `doctor-service/migrations/000001_create_doctors.up.sql`
- `doctor-service/migrations/000001_create_doctors.down.sql`
- `appointment-service/migrations/000001_create_appointments.up.sql`
- `appointment-service/migrations/000001_create_appointments.down.sql`

Migrations run automatically on service startup before the gRPC server starts listening.

Manual rollback examples:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  down 1

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  down 1
```

Manual apply examples:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  up

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  up
```

## Startup Order

Start infrastructure first:

```bash
docker compose up -d postgres nats
```

Or start the complete Docker Compose stack:

```bash
docker compose up --build
```

For a local all-in-one run from the repository root:

```bash
go run .
```

This starts Doctor Service, Appointment Service, and the Notification Service subscriber in one process. For defense, the three-terminal service startup below is still clearer because the Notification Service logs are isolated.

Start Doctor Service:

```bash
cd doctor-service
DATABASE_URL="postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
DOCTOR_SERVICE_ADDR=":8081" \
go run .
```

Start Appointment Service:

```bash
cd appointment-service
DATABASE_URL="postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
APPOINTMENT_SERVICE_ADDR=":8082" \
DOCTOR_SERVICE_GRPC_TARGET="127.0.0.1:8081" \
go run .
```

Start Notification Service:

```bash
cd notification-service
NATS_URL="nats://localhost:4222" go run .
```

Doctor Service should be available before Appointment Service handles `CreateAppointment`, because Appointment Service validates `doctor_id` synchronously over gRPC.

## Event Contract

| Subject | Publisher | Trigger | JSON fields |
| --- | --- | --- | --- |
| `doctors.created` | Doctor Service | `CreateDoctor` succeeds | `event_type`, `occurred_at`, `id`, `full_name`, `specialization`, `email` |
| `appointments.created` | Appointment Service | `CreateAppointment` succeeds | `event_type`, `occurred_at`, `id`, `title`, `doctor_id`, `status` |
| `appointments.status_updated` | Appointment Service | `UpdateAppointmentStatus` succeeds | `event_type`, `occurred_at`, `id`, `old_status`, `new_status` |

Example `appointments.created` event:

```json
{
  "event_type": "appointments.created",
  "occurred_at": "2026-05-01T10:23:44Z",
  "id": "appt-1",
  "title": "Initial cardiac consultation",
  "doctor_id": "doc-1",
  "status": "new"
}
```

## Notification Logs

Each consumed event is printed to stdout as one JSON object:

```json
{"time":"2026-05-01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"2026-05-01T10:23:44Z","id":"doc-1","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

The `time` field is when Notification Service received and processed the event. The `event` field is the full JSON payload from the publishing service.

## gRPC Behavior

Existing Assignment 2 behavior is preserved:

- Duplicate doctor email returns `AlreadyExists`.
- Missing doctor by id returns `NotFound`.
- Missing appointment by id returns `NotFound`.
- Invalid input returns `InvalidArgument`.
- Invalid appointment status transitions return `InvalidArgument`.
- If Doctor Service is unreachable during appointment creation, Appointment Service returns `Unavailable`.
- Runtime database failures return `Internal`.

Broker failures do not affect RPC responses. Doctor and Appointment services log NATS connection or publish failures and continue serving requests.

## Transactions

Appointment status updates use a PostgreSQL transaction with `SELECT ... FOR UPDATE`, then update the row and commit. This protects the read-modify-write operation from concurrent status changes. Single-row inserts are atomic by PostgreSQL statement semantics.

## NATS vs RabbitMQ

- NATS Core is lightweight pub/sub with no message persistence; choose it for simple stateless notifications and local demos.
- RabbitMQ uses exchanges and queues; choose it when subscribers need durable queues, acknowledgements, and stronger delivery guarantees.
- With NATS Core, subscribers must be online to receive messages. With RabbitMQ durable queues, a subscriber can consume queued messages after reconnecting.

## Testing

Run automated tests:

```bash
go test ./...
```

Use the commands in `grpcurl_commands.md` for live defense testing. Watch the Notification Service terminal after each write RPC to verify the expected JSON event log appears.
