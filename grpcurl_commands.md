# grpcurl Commands

All commands assume:

- `doctor-service` is running on `localhost:8081`
- `appointment-service` is running on `localhost:8082`
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
