# Assignment 3 Defense Guide

Use this file during defense to avoid missing required steps.

## 1. Start Infrastructure

```bash
docker compose up -d postgres nats
docker compose ps
```

Full Docker Compose mode:

```bash
docker compose up --build
```

Expected:

- `postgres` is running and healthy.
- `nats` is running and healthy.
- If using full Docker Compose mode, all three services are also running.

## 2. Start Services

Quick local mode:

```bash
go run .
```

This starts Doctor Service, Appointment Service, and Notification Service in one process. For defense, separate terminals are usually easier to show, especially for Notification Service logs.

Terminal 1:

```bash
cd doctor-service
DATABASE_URL="postgres://postgres:postgres@localhost:5432/doctor_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
DOCTOR_SERVICE_ADDR=":8081" \
go run .
```

Terminal 2:

```bash
cd appointment-service
DATABASE_URL="postgres://postgres:postgres@localhost:5432/appointment_service?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
APPOINTMENT_SERVICE_ADDR=":8082" \
DOCTOR_SERVICE_GRPC_TARGET="127.0.0.1:8081" \
go run .
```

Terminal 3:

```bash
cd notification-service
NATS_URL="nats://localhost:4222" go run .
```

## 3. Run grpcurl Demo

Create doctor:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{"full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}' \
  localhost:8081 doctor.DoctorService/CreateDoctor
```

Expected notification log:

```json
{"time":"...","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"...","id":"...","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

Create appointment:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{"title":"Initial cardiac consultation","description":"Patient referred for palpitations","doctor_id":"PUT_DOCTOR_ID_HERE"}' \
  localhost:8082 appointment.AppointmentService/CreateAppointment
```

Expected notification log:

```json
{"time":"...","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"...","id":"...","title":"Initial cardiac consultation","doctor_id":"PUT_DOCTOR_ID_HERE","status":"new"}}
```

Update appointment status:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{"id":"PUT_APPOINTMENT_ID_HERE","status":"in_progress"}' \
  localhost:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected notification log:

```json
{"time":"...","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"...","id":"PUT_APPOINTMENT_ID_HERE","old_status":"new","new_status":"in_progress"}}
```

## 4. Migration Rollback Demo

Stop services before rollback.

Rollback:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  down 1

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  down 1
```

Apply again:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  up

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  up
```

## 5. Required Explanation Answers

Why NATS Core?

NATS Core is simple Pub/Sub and fits this assignment because notification events are best-effort and stateless. The trade-off is no persistence, so subscribers must be online and events can be lost.

Pub/Sub vs Point-to-Point:

Pub/Sub broadcasts events to all subscribers interested in a subject. Point-to-Point sends work to one consumer from a queue. Notifications are Pub/Sub because multiple future subscribers could react to the same domain event.

Why publish after DB write?

The event should represent a state change that actually succeeded. Publishing before commit can announce data that was never saved.

What consistency problem exists?

The DB write can succeed and the process can crash before publishing. Then the event is lost. This is acceptable here because publishing is best-effort.

Production fix:

Use the Outbox pattern: write the domain change and event record in the same DB transaction, then a background relay publishes the event and marks it sent. NATS JetStream or RabbitMQ publisher confirms can add durable broker-side delivery.

ACID explanation:

- Atomicity: a transaction fully commits or rolls back.
- Consistency: constraints keep data valid.
- Isolation: concurrent operations do not corrupt each other.
- Durability: committed changes survive process crashes.

Where transaction is used:

`UpdateAppointmentStatus` uses a PostgreSQL transaction with `SELECT ... FOR UPDATE`, then updates the row and commits. This protects the read-modify-write status change.

What happens if broker is down?

Doctor and Appointment services still start and RPC writes still succeed. Publish failures are logged and do not affect the response. Notification Service is different: it retries broker connection with exponential backoff and exits non-zero after repeated failure.
