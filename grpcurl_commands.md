# grpcurl Commands

All commands assume:

- `doctor-service` is running on `localhost:8081`
- `appointment-service` is running on `localhost:8082`
- `notification-service` is running and subscribed to NATS
- PostgreSQL and NATS are running with `docker compose up -d postgres nats`
- `grpcurl` is installed locally

Because server reflection is not enabled, each command passes the local `.proto` files explicitly.

## Doctor Service

Create doctor:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }' \
  localhost:8081 doctor.DoctorService/CreateDoctor
```

Expected Notification Service output:

```json
{"time":"2026-05-01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"2026-05-01T10:23:44Z","id":"GENERATED_DOCTOR_ID","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

Get doctor by id:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{"id":"PUT_DOCTOR_ID_HERE"}' \
  localhost:8081 doctor.DoctorService/GetDoctor
```

List doctors:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{}' \
  localhost:8081 doctor.DoctorService/ListDoctors
```

Duplicate email test:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{
    "full_name": "Dr. Duplicate",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }' \
  localhost:8081 doctor.DoctorService/CreateDoctor
```

Expected result: `AlreadyExists`.

## Appointment Service

Create appointment:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "PUT_DOCTOR_ID_HERE"
  }' \
  localhost:8082 appointment.AppointmentService/CreateAppointment
```

Expected Notification Service output:

```json
{"time":"2026-05-01T10:24:01Z","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"2026-05-01T10:24:01Z","id":"GENERATED_APPOINTMENT_ID","title":"Initial cardiac consultation","doctor_id":"PUT_DOCTOR_ID_HERE","status":"new"}}
```

Get appointment by id:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{"id":"PUT_APPOINTMENT_ID_HERE"}' \
  localhost:8082 appointment.AppointmentService/GetAppointment
```

List appointments:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{}' \
  localhost:8082 appointment.AppointmentService/ListAppointments
```

Update appointment status:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "in_progress"
  }' \
  localhost:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected Notification Service output:

```json
{"time":"2026-05-01T10:25:10Z","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"2026-05-01T10:25:10Z","id":"PUT_APPOINTMENT_ID_HERE","old_status":"new","new_status":"in_progress"}}
```

Invalid status test:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "invalid"
  }' \
  localhost:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected result: `InvalidArgument`.

Doctor Service unavailable test:

1. Stop `doctor-service`.
2. Run `CreateAppointment`.
3. Expected result: `Unavailable` with a descriptive doctor service error.
