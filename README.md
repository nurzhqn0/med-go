# Doctor Appointment System

Assignment 1 implementation: a two-service medical scheduling platform in Go.

## Services

- One root entry point: `go run .`
- `doctor-service` runs on `:8081`
- `appointment-service` runs on `:8082`
- Both services expose `GET /health`
- `appointment-service` validates doctor existence by calling `doctor-service`

## Structure

```text
.
├── go.mod
├── go.sum
├── main.go
├── internal
│   ├── appointment
│   │   ├── app
│   │   ├── model
│   │   ├── repository
│   │   ├── transport/http
│   │   └── usecase
│   └── doctor
│       ├── app
│       ├── model
│       ├── repository
│       ├── transport/http
│       └── usecase
└── README.md
```

## Run

```bash
go run .
```

## API

### Doctor service

- `POST /doctors`
- `GET /doctors`
- `GET /doctors/:id`

Create doctor payload:

```json
{
  "full_name": "Dr. Alice Brown",
  "specialization": "Cardiology",
  "email": "alice.brown@example.com"
}
```

### Appointment service

- `POST /appointments`
- `GET /appointments`
- `GET /appointments/:id`
- `PATCH /appointments/:id/status`

Create appointment payload:

```json
{
  "title": "Initial Consultation",
  "description": "Review chest pain symptoms",
  "doctor_id": "doc-1"
}
```

Update status payload:

```json
{
  "status": "in_progress"
}
```

Supported statuses:

- `new`
- `in_progress`
- `done`

## Notes

- Data is stored in memory, so restarting the app resets doctors and appointments.
- A doctor must exist in `doctor-service` before an appointment can be created for that doctor.
