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

The default Docker Compose deployment is meant for servers that already run host Nginx on port `80`.

Default stack:

- `app`: the Go binary running both services
- `mongo`: MongoDB 8 with a persistent volume

Start it:

```bash
docker compose up --build -d
```

This publishes the app only on loopback:

- `127.0.0.1:8081`
- `127.0.0.1:8082`

Stop it:

```bash
docker compose down
```

If you want the bundled Docker Nginx proxy too, start the optional `proxy` profile:

```bash
docker compose --profile proxy up --build -d
```

Use that profile only when port `80` is free on the host.

If you want to use Atlas instead of the bundled Mongo container, override `COMPOSE_MONGODB_URI` when starting Compose:

```bash
COMPOSE_MONGODB_URI='mongodb+srv://USER:PASSWORD@cluster.mongodb.net/' docker compose up --build -d
```

If your server already runs Nginx, use [host-med-go.conf](/Users/myrzanizimbetov/Desktop/med-go/deploy/nginx/host-med-go.conf) as the site config and proxy to:

- `127.0.0.1:8081` for `/doctors`
- `127.0.0.1:8082` for `/appointments`

## Assignment 3 Observability

Start Prometheus, Grafana, and node-exporter with:

```bash
docker compose --profile observability up --build -d
```

Available endpoints:

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000`
- Doctor metrics: `http://localhost:8081/metrics`
- Appointment metrics: `http://localhost:8082/metrics`

Assignment files:

- writeup: [ASSIGNMENT_3.md](/Users/myrzanizimbetov/Desktop/med-go/ASSIGNMENT_3.md)
- Prometheus config: [prometheus.yml](/Users/myrzanizimbetov/Desktop/med-go/monitoring/prometheus/prometheus.yml)
- alert rules: [alerts.yml](/Users/myrzanizimbetov/Desktop/med-go/monitoring/prometheus/alerts.yml)
- Grafana dashboard: [med-go-observability.json](/Users/myrzanizimbetov/Desktop/med-go/monitoring/grafana/dashboards/med-go-observability.json)

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
