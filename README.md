# Doctor Appointment System

Assignment 1 implementation: a two-service medical scheduling platform in Go with MongoDB persistence.

## Services

- One root entry point: `go run .`
- `doctor-service` runs on `:8081`
- `appointment-service` runs on `:8082`
- Both services expose `GET /health`
- `appointment-service` validates doctor existence by calling `doctor-service`
- Data is persisted in MongoDB collections `doctors` and `appointments`

## Docker Deploy

The easiest deployment path is Docker Compose. This stack starts:

- `app`: the Go binary running both services
- `mongo`: MongoDB 8 with a persistent volume
- `nginx`: reverse proxy exposing one public base URL

Start everything:

```bash
docker compose up --build -d
```

Stop everything:

```bash
docker compose down
```

Public endpoints through Nginx:

- `GET /`
- `GET /doctor-health`
- `GET /appointment-health`
- `POST /doctors`
- `GET /doctors`
- `GET /doctors/:id`
- `POST /appointments`
- `GET /appointments`
- `GET /appointments/:id`
- `PATCH /appointments/:id/status`

If you want to use Atlas instead of the bundled Mongo container, override `COMPOSE_MONGODB_URI` when starting Compose:

```bash
COMPOSE_MONGODB_URI='mongodb+srv://USER:PASSWORD@cluster.mongodb.net/' docker compose up --build -d
```

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

Start MongoDB locally first, for example with Docker:

```bash
docker run --name med-go-mongo -p 27017:27017 -d mongo:8
```

Create a local `.env` file from the example:

```bash
cp .env.example .env
```

The app reads `.env` automatically on startup. These are the supported variables:

```bash
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=med_go
DOCTOR_SERVICE_ADDR=:8081
APPOINTMENT_SERVICE_ADDR=:8082
DOCTOR_SERVICE_BASE_URL=http://localhost:8081
```

Then run the app:

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
  "doctor_id": "PUT_REAL_DOCTOR_ID_HERE"
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

- A doctor must exist in `doctor-service` before an appointment can be created for that doctor.
- If MongoDB is unavailable, the application exits during startup.
