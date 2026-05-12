# grpcurl Commands - Assignment 4

This file contains defense-ready commands for all runnable services, including the Assignment 4 Redis cache, rate limiter, job queue, and mock gateway checks.

The project has two gRPC services:

- Doctor Service on `localhost:8081`
- Appointment Service on `localhost:8082`

Notification Service has no gRPC API. It is verified by reading Docker logs because it consumes NATS events, prints event logs, and processes background jobs.

## 0. Start The Full Stack

From the repository root:

```bash
docker compose down --remove-orphans
docker compose up --build
```

If you want detached mode:

```bash
docker compose up -d --build
```

Check containers:

```bash
docker compose ps
```

Watch Notification Service logs:

```bash
docker compose logs -f notification-service
```

## 1. Reflection Checks

Reflection is enabled, so these commands should list available services without passing proto files.

Doctor Service:

```bash
grpcurl -plaintext 127.0.0.1:8081 list
```

Expected includes:

```text
doctor.DoctorService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

Appointment Service:

```bash
grpcurl -plaintext 127.0.0.1:8082 list
```

Expected includes:

```text
appointment.AppointmentService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

List Doctor methods:

```bash
grpcurl -plaintext 127.0.0.1:8081 list doctor.DoctorService
```

Expected:

```text
doctor.DoctorService.CreateDoctor
doctor.DoctorService.GetDoctor
doctor.DoctorService.ListDoctors
```

List Appointment methods:

```bash
grpcurl -plaintext 127.0.0.1:8082 list appointment.AppointmentService
```

Expected:

```text
appointment.AppointmentService.CreateAppointment
appointment.AppointmentService.GetAppointment
appointment.AppointmentService.ListAppointments
appointment.AppointmentService.UpdateAppointmentStatus
```

## 2. Doctor Service Commands

### Create Doctor

Using reflection:

```bash
grpcurl -plaintext \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }' \
  127.0.0.1:8081 doctor.DoctorService/CreateDoctor
```

Using proto file:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/doctor/proto/doctor.proto \
  -d '{
    "full_name": "Dr. Aisha Seitkali",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }' \
  127.0.0.1:8081 doctor.DoctorService/CreateDoctor
```

Expected response:

```json
{
  "id": "GENERATED_DOCTOR_ID",
  "fullName": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

Save the returned `id`; use it as `PUT_DOCTOR_ID_HERE`.

Expected Notification Service log:

```json
{"time":"...","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"...","id":"GENERATED_DOCTOR_ID","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

### Get Doctor

```bash
grpcurl -plaintext \
  -d '{"id":"PUT_DOCTOR_ID_HERE"}' \
  127.0.0.1:8081 doctor.DoctorService/GetDoctor
```

Expected response:

```json
{
  "id": "PUT_DOCTOR_ID_HERE",
  "fullName": "Dr. Aisha Seitkali",
  "specialization": "Cardiology",
  "email": "a.seitkali@clinic.kz"
}
```

### List Doctors

```bash
grpcurl -plaintext \
  -d '{}' \
  127.0.0.1:8081 doctor.DoctorService/ListDoctors
```

Expected response contains:

```json
{
  "doctors": [
    {
      "id": "PUT_DOCTOR_ID_HERE",
      "fullName": "Dr. Aisha Seitkali",
      "specialization": "Cardiology",
      "email": "a.seitkali@clinic.kz"
    }
  ]
}
```

### Duplicate Email Error

Run `CreateDoctor` again with the same email:

```bash
grpcurl -plaintext \
  -d '{
    "full_name": "Dr. Duplicate",
    "specialization": "Cardiology",
    "email": "a.seitkali@clinic.kz"
  }' \
  127.0.0.1:8081 doctor.DoctorService/CreateDoctor
```

Expected error:

```text
Code: AlreadyExists
Message: doctor email already exists
```

### Invalid Doctor Input Error

```bash
grpcurl -plaintext \
  -d '{
    "full_name": "",
    "specialization": "Cardiology",
    "email": "bad-email"
  }' \
  127.0.0.1:8081 doctor.DoctorService/CreateDoctor
```

Expected error:

```text
Code: InvalidArgument
Message: invalid doctor input
```

### Doctor Not Found Error

```bash
grpcurl -plaintext \
  -d '{"id":"missing-doctor-id"}' \
  127.0.0.1:8081 doctor.DoctorService/GetDoctor
```

Expected error:

```text
Code: NotFound
Message: doctor not found
```

## 3. Appointment Service Commands

### Create Appointment

Replace `PUT_DOCTOR_ID_HERE` with a real id returned by CreateDoctor.

Using reflection:

```bash
grpcurl -plaintext \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "PUT_DOCTOR_ID_HERE"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/CreateAppointment
```

Using proto file:

```bash
grpcurl -plaintext \
  -import-path . \
  -proto internal/appointment/proto/appointment.proto \
  -d '{
    "title": "Initial cardiac consultation",
    "description": "Patient referred for palpitations and shortness of breath",
    "doctor_id": "PUT_DOCTOR_ID_HERE"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/CreateAppointment
```

Expected response:

```json
{
  "id": "GENERATED_APPOINTMENT_ID",
  "title": "Initial cardiac consultation",
  "description": "Patient referred for palpitations and shortness of breath",
  "doctorId": "PUT_DOCTOR_ID_HERE",
  "status": "new",
  "createdAt": "...",
  "updatedAt": "..."
}
```

Save the returned `id`; use it as `PUT_APPOINTMENT_ID_HERE`.

Expected Notification Service log:

```json
{"time":"...","subject":"appointments.created","event":{"event_type":"appointments.created","occurred_at":"...","id":"GENERATED_APPOINTMENT_ID","title":"Initial cardiac consultation","doctor_id":"PUT_DOCTOR_ID_HERE","status":"new"}}
```

### Get Appointment

```bash
grpcurl -plaintext \
  -d '{"id":"PUT_APPOINTMENT_ID_HERE"}' \
  127.0.0.1:8082 appointment.AppointmentService/GetAppointment
```

Expected response contains:

```json
{
  "id": "PUT_APPOINTMENT_ID_HERE",
  "title": "Initial cardiac consultation",
  "doctorId": "PUT_DOCTOR_ID_HERE",
  "status": "new"
}
```

### List Appointments

```bash
grpcurl -plaintext \
  -d '{}' \
  127.0.0.1:8082 appointment.AppointmentService/ListAppointments
```

Expected response contains an `appointments` array.

### Update Appointment Status

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "in_progress"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected response contains:

```json
{
  "id": "PUT_APPOINTMENT_ID_HERE",
  "status": "in_progress"
}
```

Expected Notification Service log:

```json
{"time":"...","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"...","id":"PUT_APPOINTMENT_ID_HERE","old_status":"new","new_status":"in_progress"}}
```

### Complete Appointment

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "done"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected response contains:

```json
{
  "status": "done"
}
```

### Invalid Status Error

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "invalid"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected error:

```text
Code: InvalidArgument
Message: invalid appointment status
```

### Invalid Transition Error

If the appointment is already `done`, this attempts the forbidden `done -> new` transition:

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "new"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected error:

```text
Code: InvalidArgument
Message: invalid appointment status transition: done -> new
```

### Appointment Not Found Error

```bash
grpcurl -plaintext \
  -d '{"id":"missing-appointment-id"}' \
  127.0.0.1:8082 appointment.AppointmentService/GetAppointment
```

Expected error:

```text
Code: NotFound
Message: appointment not found
```

### Doctor Missing During Appointment Create

```bash
grpcurl -plaintext \
  -d '{
    "title": "Invalid doctor appointment",
    "description": "Doctor id does not exist",
    "doctor_id": "missing-doctor-id"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/CreateAppointment
```

Expected error:

```text
Code: FailedPrecondition
Message: doctor not found
```

### Doctor Service Unavailable Scenario

Stop only Doctor Service:

```bash
docker compose stop doctor-service
```

Call CreateAppointment:

```bash
grpcurl -plaintext \
  -d '{
    "title": "Doctor service down test",
    "description": "Doctor service should be unavailable",
    "doctor_id": "PUT_DOCTOR_ID_HERE"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/CreateAppointment
```

Expected error:

```text
Code: Unavailable
```

Start Doctor Service again:

```bash
docker compose start doctor-service
```

## 4. Notification Service Verification

Notification Service does not expose gRPC, HTTP, or any port.

It is verified through logs:

```bash
docker compose logs -f notification-service
```

Expected event subjects after successful write commands:

```text
doctors.created
appointments.created
appointments.status_updated
```

Each log line must be one JSON object with:

- `time`
- `subject`
- `event`

Example:

```json
{"time":"2026-05-01T10:23:44Z","subject":"doctors.created","event":{"event_type":"doctors.created","occurred_at":"2026-05-01T10:23:44Z","id":"...","full_name":"Dr. Aisha Seitkali","specialization":"Cardiology","email":"a.seitkali@clinic.kz"}}
```

## 5. Database And Migration Checks

Postgres is published on host port `5433` to avoid conflicts with local Postgres on `5432`.

Check databases:

```bash
docker compose exec postgres psql -U postgres -d doctor_service -c '\dt'
docker compose exec postgres psql -U postgres -d appointment_service -c '\dt'
```

Expected tables:

```text
doctors
appointments
```

Manual migration rollback:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  down 1

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  down 1
```

Manual migration apply:

```bash
migrate -path doctor-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable" \
  up

migrate -path appointment-service/migrations \
  -database "postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable" \
  up
```

## 6. One Full Demo Sequence

Use this sequence in defense:

1. Start stack:

```bash
docker compose down --remove-orphans
docker compose up --build
```

2. In another terminal, watch notification logs:

```bash
docker compose logs -f notification-service
```

3. Run `CreateDoctor`.

4. Copy doctor id.

5. Run `CreateAppointment`.

6. Copy appointment id.

7. Run `UpdateAppointmentStatus` to `in_progress`.

8. Show all three notification logs.

9. Run duplicate email test.

10. Run invalid status test.

11. Explain best-effort event publishing and Outbox pattern.

## 7. Assignment 4 Cache Checks

Start Redis monitor in a separate terminal:

```bash
redis-cli MONITOR
```

Run `GetDoctor` twice with the same id:

```bash
grpcurl -plaintext \
  -d '{"id":"PUT_DOCTOR_ID_HERE"}' \
  127.0.0.1:8081 doctor.DoctorService/GetDoctor

grpcurl -plaintext \
  -d '{"id":"PUT_DOCTOR_ID_HERE"}' \
  127.0.0.1:8081 doctor.DoctorService/GetDoctor
```

Expected Redis behavior:

- First call: `GET doctor:PUT_DOCTOR_ID_HERE`, then `SET doctor:PUT_DOCTOR_ID_HERE`.
- Second call: `GET doctor:PUT_DOCTOR_ID_HERE` only.

List cache check:

```bash
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
```

Expected keys:

```text
doctors:list
```

Appointment cache keys:

```text
appointment:PUT_APPOINTMENT_ID_HERE
appointments:list
```

`CreateDoctor`, `CreateAppointment`, and `UpdateAppointmentStatus` should delete the matching list cache key after the database write succeeds.

## 8. Assignment 4 Rate Limiter Check

Start Doctor Service or the full stack with a low limit:

```bash
RATE_LIMIT_RPM=2 docker compose up --build
```

Call the same endpoint more than two times in one minute:

```bash
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
grpcurl -plaintext -d '{}' 127.0.0.1:8081 doctor.DoctorService/ListDoctors
```

Expected third response:

```text
Code: ResourceExhausted
Message: rate limit exceeded, retry after ... seconds
```

Expected Redis key pattern:

```text
rate:doctor-service:127.0.0.1
```

## 9. Assignment 4 Job Queue And Gateway Check

Watch Notification Service logs:

```bash
docker compose logs -f notification-service
```

Watch Mock Gateway logs:

```bash
docker compose logs -f mock-gateway
```

Complete an appointment:

```bash
grpcurl -plaintext \
  -d '{
    "id": "PUT_APPOINTMENT_ID_HERE",
    "status": "done"
  }' \
  127.0.0.1:8082 appointment.AppointmentService/UpdateAppointmentStatus
```

Expected Notification Service event log:

```json
{"time":"...","subject":"appointments.status_updated","event":{"event_type":"appointments.status_updated","occurred_at":"...","id":"PUT_APPOINTMENT_ID_HERE","doctor_id":"PUT_DOCTOR_ID_HERE","old_status":"in_progress","new_status":"done"}}
```

Expected job logs:

```json
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"enqueued"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"processing"}
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"success"}
```

If the gateway returns a simulated 503, expected retry log:

```json
{"time":"...","level":"warn","job_id":"...","attempt":1,"status":"retry","error":"gateway returned 503"}
```

Expected Mock Gateway log:

```json
{"time":"...","path":"/notify","body":{"idempotency_key":"...","channel":"email","recipient":"patient@clinic.kz","message":"Your appointment PUT_APPOINTMENT_ID_HERE with doctor PUT_DOCTOR_ID_HERE is complete."}}
```

## 10. Assignment 4 Idempotency Replay Check

Replay the same `appointments.status_updated` event payload into NATS. Replace the JSON with the exact event from Notification Service logs:

```bash
nats pub appointments.status_updated '{
  "event_type": "appointments.status_updated",
  "occurred_at": "SAME_OCCURRED_AT_FROM_LOG",
  "id": "PUT_APPOINTMENT_ID_HERE",
  "doctor_id": "PUT_DOCTOR_ID_HERE",
  "old_status": "in_progress",
  "new_status": "done"
}'
```

Expected behavior:

- Notification Service prints the event log line.
- Job queue sees `notification:job:<sha>` already set to `done`.
- No second Mock Gateway `/notify` request is sent.

## 11. Assignment 4 Dead Letter Check

Stop the Mock Gateway:

```bash
docker compose stop mock-gateway
```

Trigger a different appointment to `done`.

Expected job logs:

```json
{"time":"...","level":"info","job_id":"...","attempt":1,"status":"processing"}
{"time":"...","level":"warn","job_id":"...","attempt":1,"status":"retry","error":"..."}
{"time":"...","level":"info","job_id":"...","attempt":2,"status":"processing"}
{"time":"...","level":"warn","job_id":"...","attempt":2,"status":"retry","error":"..."}
{"time":"...","level":"info","job_id":"...","attempt":3,"status":"processing"}
{"time":"...","level":"warn","job_id":"...","attempt":3,"status":"retry","error":"..."}
```

Expected stderr dead-letter log:

```json
{"time":"...","level":"error","job_id":"...","attempt":3,"status":"dead_letter","error":"..."}
```

Restart the gateway after the check:

```bash
docker compose start mock-gateway
```
