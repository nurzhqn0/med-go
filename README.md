# Doctor Appointment System

Two HTTP microservices in Go:

- `doctor-service` owns doctor data
- `appointment-service` owns appointment data
- `appointment-service` validates doctor existence through REST, not through shared storage

The project stays in one repository and one Go module for coursework convenience, but each service now has its own runnable entrypoint:

- `go run ./cmd/doctor-service`
- `go run ./cmd/appointment-service`

For local convenience there is also a combined runner:

- `go run .`

## Architecture

```mermaid
flowchart LR
    Client["Client / Postman / curl"] --> DoctorHTTP["doctor-service HTTP transport"]
    Client --> AppointmentHTTP["appointment-service HTTP transport"]

    DoctorHTTP --> DoctorUC["doctor use case"]
    AppointmentHTTP --> AppointmentUC["appointment use case"]

    DoctorUC --> DoctorRepo["doctor repository"]
    AppointmentUC --> AppointmentRepo["appointment repository"]
    AppointmentUC --> DoctorLookup["doctor-service REST client"]

    DoctorRepo --> DoctorDB["MongoDB doctors collection"]
    AppointmentRepo --> AppointmentDB["MongoDB appointments collection"]
    DoctorLookup --> DoctorHTTP
```

## Service Responsibilities

`doctor-service`

- creates doctors
- lists doctors
- gets doctor by id
- enforces required `full_name`
- enforces required valid `email`
- enforces unique email across all doctors

`appointment-service`

- creates appointments
- lists appointments
- gets appointment by id
- updates appointment status
- validates that `doctor_id` exists by calling `doctor-service`
- enforces appointment status rules

## Dependency Direction

The dependency flow is:

- `transport -> usecase -> repository/client`
- handlers depend on interfaces, not concrete use case structs
- domain models no longer contain HTTP JSON tags
- MongoDB document mapping stays inside repositories

This keeps HTTP concerns in transport and persistence concerns in repository code.

## Data Ownership

There is no shared doctor table read from `appointment-service`.

- `doctor-service` owns doctor records
- `appointment-service` stores only `doctor_id`
- doctor existence is checked through `GET /doctors/:id`

This makes the service boundary explicit and avoids hidden coupling through a shared database query.

## Failure Behavior

Doctor lookup from `appointment-service` uses:

- context-aware outbound requests
- a `3s` HTTP client timeout
- internal logging when doctor lookup fails
- `503 Service Unavailable` when `doctor-service` cannot be reached or returns `5xx`

Validation failures still return `400`, missing resources return `404`, and duplicate doctor emails return `409`.

## Project Structure

```text
.
├── cmd
│   ├── appointment-service
│   └── doctor-service
├── internal
│   ├── appointment
│   │   ├── app
│   │   ├── client
│   │   ├── model
│   │   ├── repository
│   │   ├── transport/http
│   │   └── usecase
│   ├── doctor
│   │   ├── app
│   │   ├── model
│   │   ├── repository
│   │   ├── transport/http
│   │   └── usecase
│   └── platform
├── main.go
└── README.md
```

## Configuration

Supported environment variables:

```bash
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=med_go
DOCTOR_SERVICE_ADDR=:8081
APPOINTMENT_SERVICE_ADDR=:8082
DOCTOR_SERVICE_BASE_URL=http://localhost:8081
```

The app reads `.env` automatically if present.

## Run Locally

Start MongoDB, for example:

```bash
docker run --name med-go-mongo -p 27017:27017 -d mongo:8
```

Run services separately:

```bash
go run ./cmd/doctor-service
go run ./cmd/appointment-service
```

Or run both from one process:

```bash
go run .
```

## Docker

Start the default stack:

```bash
docker compose up --build -d
```

This exposes:

- `127.0.0.1:8081` for `doctor-service`
- `127.0.0.1:8082` for `appointment-service`

Stop it:

```bash
docker compose down
```

Optional reverse proxy profile:

```bash
docker compose --profile proxy up --build -d
```

## API

### Doctor Service

`POST /doctors`

```json
{
  "full_name": "Dr. Alice Brown",
  "specialization": "Cardiology",
  "email": "alice.brown@example.com"
}
```

`specialization` is optional. `full_name` and `email` are required.

Example:

```bash
curl -s -X POST http://localhost:8081/doctors \
  -H 'Content-Type: application/json' \
  -d '{
    "full_name":"Dr. Alice Brown",
    "specialization":"Cardiology",
    "email":"alice.brown@example.com"
  }'
```

List doctors:

```bash
curl -s http://localhost:8081/doctors
```

Get doctor by id:

```bash
curl -s http://localhost:8081/doctors/<doctor_id>
```

### Appointment Service

`POST /appointments`

```json
{
  "title": "Initial Consultation",
  "description": "Review chest pain symptoms",
  "doctor_id": "<doctor_id>"
}
```

Example:

```bash
curl -s -X POST http://localhost:8082/appointments \
  -H 'Content-Type: application/json' \
  -d '{
    "title":"Initial Consultation",
    "description":"Review chest pain symptoms",
    "doctor_id":"<doctor_id>"
  }'
```

List appointments:

```bash
curl -s http://localhost:8082/appointments
```

Get appointment by id:

```bash
curl -s http://localhost:8082/appointments/<appointment_id>
```

Update status:

```bash
curl -s -X PATCH http://localhost:8082/appointments/<appointment_id>/status \
  -H 'Content-Type: application/json' \
  -d '{"status":"in_progress"}'
```

Supported statuses:

- `new`
- `in_progress`
- `done`

Forbidden transition:

- `done -> new`

## Observability

Start the observability profile:

```bash
docker compose --profile observability up --build -d
```

Available endpoints:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- Doctor metrics: `http://localhost:8081/metrics`
- Appointment metrics: `http://localhost:8082/metrics`

## Notes

- if MongoDB is unavailable, startup fails fast
- doctor emails are normalized to lowercase before storing
- `doctor-service` creates a unique MongoDB index on `email`
