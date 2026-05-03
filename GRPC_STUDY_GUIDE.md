# gRPC Study Guide For This Project

This file is written for a beginner.

If you feel like "I do not know anything in this code", start here.

The goal of this guide is to explain:

- what gRPC is
- what Protocol Buffers (`.proto`) are
- how requests move through your code
- what each layer does
- how your two services talk to each other
- what you should say during defense

## 1. Big Picture

Your project is a small medical scheduling platform with two services:

- `doctor-service`
- `appointment-service`

They are separate because they own different business areas:

- doctor-service owns doctor data
- appointment-service owns appointment data

The services communicate using **gRPC**.

That means:

- the client does not send normal REST requests like `POST /doctors`
- the client calls named RPC methods like `DoctorService.CreateDoctor`
- request and response data is described in `.proto` files

## 2. What Is gRPC?

gRPC is a framework for communication between programs.

You can think about it like this:

- REST says: "send HTTP request to URL with JSON body"
- gRPC says: "call a remote function with a typed request object"

Example idea:

- REST style:
  - `POST /appointments`
  - JSON body with `title`, `description`, `doctor_id`

- gRPC style:
  - call `AppointmentService.CreateAppointment`
  - pass `CreateAppointmentRequest`
  - receive `AppointmentResponse`

So gRPC feels closer to calling a function, even though the function is running on another service.

## 3. What Is Protocol Buffers?

Protocol Buffers, or **protobuf**, is the format gRPC uses to define data and services.

You write a `.proto` file. In that file, you define:

- service names
- RPC method names
- request message structure
- response message structure

Example:

```proto
service DoctorService {
  rpc CreateDoctor(CreateDoctorRequest) returns (DoctorResponse);
}
```

This means:

- there is a service called `DoctorService`
- it has a remote method called `CreateDoctor`
- it accepts `CreateDoctorRequest`
- it returns `DoctorResponse`

Then `protoc` generates Go code from that contract.

## 4. Why Do We Need `.proto` Files?

Because gRPC needs a contract.

The contract answers:

- what methods exist?
- what is the request shape?
- what is the response shape?
- what are the exact field names and field types?

Without a `.proto` file, the client and server would not agree on how to communicate.

In your project:

- [internal/doctor/proto/doctor.proto](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/proto/doctor.proto)
- [internal/appointment/proto/appointment.proto](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/proto/appointment.proto)

## 5. What Files Are Generated From `.proto`?

For each `.proto`, Go code is generated:

- `doctor.pb.go`
- `doctor_grpc.pb.go`
- `appointment.pb.go`
- `appointment_grpc.pb.go`

These files are generated, not handwritten.

In your project:

- [internal/doctor/proto/doctor.pb.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/proto/doctor.pb.go)
- [internal/doctor/proto/doctor_grpc.pb.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/proto/doctor_grpc.pb.go)
- [internal/appointment/proto/appointment.pb.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/proto/appointment.pb.go)
- [internal/appointment/proto/appointment_grpc.pb.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/proto/appointment_grpc.pb.go)

What they contain:

- Go structs for requests and responses
- server interfaces
- client stubs
- registration helpers for gRPC server

You normally do not edit these files manually.

## 6. What Is a Client Stub?

A client stub is generated Go code that lets one service call another service.

For example:

- appointment-service needs to ask doctor-service whether a doctor exists
- instead of building raw network packets, it uses the generated gRPC client

In your project this happens in:

- [internal/appointment/client/doctor_service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/client/doctor_service.go)

That file creates:

- a gRPC connection
- a `DoctorServiceClient`

Then it calls:

```go
c.client.GetDoctor(...)
```

That is the remote call to doctor-service.

## 7. What Is the Server Side in gRPC?

On the server side, you implement the methods described in `.proto`.

For example:

- `.proto` says Doctor Service has `CreateDoctor`
- your Go code must implement that method

In your project:

- [internal/doctor/transport/grpc/server.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/transport/grpc/server.go)
- [internal/appointment/transport/grpc/server.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/transport/grpc/server.go)

These files are your **delivery layer** for gRPC.

Their job is:

- receive gRPC request
- map proto request into use case input
- call use case
- map domain result into proto response
- convert errors into gRPC status codes

Important:

- no business logic should live here

## 8. What Is Clean Architecture in This Project?

Clean Architecture means the project is split into layers with clear responsibilities.

Your project mainly has these layers:

### Domain / model

Files:

- [internal/doctor/model/doctor.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/model/doctor.go)
- [internal/appointment/model/appointment.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/model/appointment.go)

These are the core business entities.

Examples:

- `Doctor`
- `Appointment`
- `Status`

These files should not know about:

- gRPC
- protobuf
- MongoDB
- framework-specific details

### Use case

Files:

- [internal/doctor/usecase/service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/usecase/service.go)
- [internal/appointment/usecase/service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/usecase/service.go)

This is where business rules live.

Examples:

- doctor `full_name` is required
- doctor `email` is required and must be unique
- appointment `title` is required
- appointment `doctor_id` is required
- doctor must exist before appointment is created
- status must be `new`, `in_progress`, or `done`
- `done -> new` is forbidden

### Repository

Files:

- [internal/doctor/repository/mongo.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/repository/mongo.go)
- [internal/doctor/repository/memory.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/repository/memory.go)
- [internal/appointment/repository/mongo.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/repository/mongo.go)
- [internal/appointment/repository/memory.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/repository/memory.go)

This layer talks to storage.

Its job:

- save data
- read data
- update data

It should not contain business rules like "done cannot go back to new". That rule belongs in use case.

### Transport / delivery

Files:

- [internal/doctor/transport/grpc/server.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/transport/grpc/server.go)
- [internal/appointment/transport/grpc/server.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/transport/grpc/server.go)

This layer talks to the outside world.

Its job:

- accept gRPC requests
- call use cases
- return gRPC responses

### Client

File:

- [internal/appointment/client/doctor_service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/client/doctor_service.go)

This is the outbound communication layer for appointment-service.

It calls doctor-service over gRPC.

### App / wiring

Files:

- [internal/doctor/app/app.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/app/app.go)
- [internal/appointment/app/app.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/app/app.go)

This layer wires everything together:

- repository
- use case
- gRPC server
- outbound client

### Entry points

Files:

- [main.go](/Users/myrzanizimbetov/Desktop/med-go/main.go)
- [cmd/doctor-service/main.go](/Users/myrzanizimbetov/Desktop/med-go/cmd/doctor-service/main.go)
- [cmd/appointment-service/main.go](/Users/myrzanizimbetov/Desktop/med-go/cmd/appointment-service/main.go)

These files start the services.

## 9. The Most Important Dependency Rule

Outer layers can depend on inner layers.

But inner layers should not depend on outer layers.

Good:

- gRPC server depends on use case
- use case depends on repository interface

Bad:

- use case imports protobuf-generated types
- domain model imports gRPC package
- use case directly creates gRPC client connection

In your project, this is kept clean:

- use cases do not use protobuf types
- models do not use protobuf types
- gRPC mapping happens in the transport layer

## 10. What Is the Full Request Flow?

Let us walk through a real example.

### Example A: Create doctor

Client calls:

- `doctor.DoctorService/CreateDoctor`

Step by step:

1. Client sends `CreateDoctorRequest`.
2. gRPC server in [internal/doctor/transport/grpc/server.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/transport/grpc/server.go) receives it.
3. That server builds `usecase.CreateDoctorInput`.
4. It calls doctor use case.
5. Use case validates:
   - `full_name` required
   - valid email
   - email uniqueness
6. Use case asks repository to save the doctor.
7. Repository stores it in MongoDB.
8. Use case returns domain `Doctor`.
9. gRPC server maps `Doctor` to `DoctorResponse`.
10. Client receives the response.

### Example B: Create appointment

Client calls:

- `appointment.AppointmentService/CreateAppointment`

Step by step:

1. Client sends `CreateAppointmentRequest`.
2. Appointment gRPC server receives it.
3. It maps request into `usecase.CreateAppointmentInput`.
4. Appointment use case validates:
   - `title` required
   - `doctor_id` required
5. Appointment use case calls `DoctorLookup.Exists(...)`.
6. The implementation of `DoctorLookup` is the gRPC client in [internal/appointment/client/doctor_service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/client/doctor_service.go).
7. That client calls doctor-service remotely with `GetDoctor`.
8. If doctor exists, control returns to appointment use case.
9. Appointment use case creates the appointment and stores it.
10. Appointment gRPC server maps the result into `AppointmentResponse`.
11. Client receives the response.

## 11. Why Does Appointment Service Call Doctor Service?

Because of data ownership.

doctor-service owns doctors.

appointment-service does **not** directly read doctor Mongo collection.

That is important because:

- service boundaries stay real
- appointment-service does not know doctor-service storage details
- doctor-service remains the source of truth for doctors

This is one of the core microservice ideas.

## 12. Why Not Use One Shared Database Query?

Because that would weaken the service boundary.

If appointment-service directly queried the doctors collection:

- it would depend on doctor-service internals
- changing doctor storage would break appointment-service
- the system would start looking like a distributed monolith

Better:

- appointment-service asks doctor-service through a clear API

## 13. What Is Stored in `.proto` vs What Is Stored in Domain Model?

This is very important for defense.

### Proto messages

Proto messages are transport contracts.

They define:

- how data travels over the network
- field names in gRPC requests/responses

Example:

- `CreateDoctorRequest`
- `DoctorResponse`
- `CreateAppointmentRequest`

### Domain models

Domain models represent business entities in your application.

Examples:

- `model.Doctor`
- `model.Appointment`

They are internal application objects.

Important rule:

- proto types stay in transport
- domain types stay in use case and repository

## 14. Why Do We Map Between Proto And Domain?

Because transport is not business logic.

The gRPC server receives:

- `doctorpb.CreateDoctorRequest`

But use case should work with:

- `usecase.CreateDoctorInput`

That conversion happens in transport.

This keeps business logic independent from gRPC.

## 15. What Is Inside `doctor.proto`?

File:

- [internal/doctor/proto/doctor.proto](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/proto/doctor.proto)

It defines:

- service: `DoctorService`
- RPCs:
  - `CreateDoctor`
  - `GetDoctor`
  - `ListDoctors`

Messages:

- `CreateDoctorRequest`
- `GetDoctorRequest`
- `ListDoctorsRequest`
- `DoctorResponse`
- `ListDoctorsResponse`

This contract says exactly what doctor-service exposes over gRPC.

## 16. What Is Inside `appointment.proto`?

File:

- [internal/appointment/proto/appointment.proto](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/proto/appointment.proto)

It defines:

- service: `AppointmentService`
- RPCs:
  - `CreateAppointment`
  - `GetAppointment`
  - `ListAppointments`
  - `UpdateAppointmentStatus`

Messages:

- `CreateAppointmentRequest`
- `GetAppointmentRequest`
- `ListAppointmentsRequest`
- `UpdateStatusRequest`
- `AppointmentResponse`
- `ListAppointmentsResponse`

## 17. What Is `go_package` In `.proto`?

Example:

```proto
option go_package = "med-go/internal/doctor/proto;doctorpb";
```

This tells the generator:

- where the generated Go package belongs
- what Go package name to use

For doctor:

- Go package name becomes `doctorpb`

For appointment:

- Go package name becomes `appointmentpb`

That is why your code imports:

```go
doctorpb "med-go/internal/doctor/proto"
appointmentpb "med-go/internal/appointment/proto"
```

## 18. How Are Stubs Generated?

You use `protoc`.

In your project there is a helper script:

- [scripts/generate_proto.sh](/Users/myrzanizimbetov/Desktop/med-go/scripts/generate_proto.sh)

It runs `protoc` and generates:

- `*.pb.go`
- `*_grpc.pb.go`

Command:

```bash
bash scripts/generate_proto.sh
```

## 19. What Happens in `internal/doctor/app/app.go`?

This file wires doctor-service together.

In simple words, it does:

1. create repository
2. create use case service
3. create gRPC server
4. register DoctorService implementation on that server

So this file is not business logic.

It is assembly / wiring.

## 20. What Happens in `internal/appointment/app/app.go`?

This file wires appointment-service together.

It does:

1. create appointment repository
2. create Doctor Service gRPC client
3. create appointment use case
4. create gRPC server
5. register AppointmentService implementation

This file is also wiring, not business logic.

## 21. What Is the Difference Between `main.go` And `cmd/.../main.go`?

### Root `main.go`

File:

- [main.go](/Users/myrzanizimbetov/Desktop/med-go/main.go)

This starts both services together in one process.

This exists because the assignment requires:

- `go run .`

### `cmd/doctor-service/main.go`

Starts only doctor-service.

### `cmd/appointment-service/main.go`

Starts only appointment-service.

These are useful for separate runs.

## 22. What Ports Are Used?

By default:

- doctor-service -> `:8081`
- appointment-service -> `:8082`

These are now gRPC ports, not REST ports.

## 23. What Environment Variables Exist?

See:

- [.env.example](/Users/myrzanizimbetov/Desktop/med-go/.env.example)

Main values:

- `MONGODB_URI`
- `MONGODB_DATABASE`
- `DOCTOR_SERVICE_ADDR`
- `APPOINTMENT_SERVICE_ADDR`
- `DOCTOR_SERVICE_GRPC_TARGET`

`DOCTOR_SERVICE_GRPC_TARGET` is used by appointment-service to call doctor-service.

Default:

- `127.0.0.1:8081`

## 24. What Are gRPC Status Codes?

In REST, you return HTTP statuses like:

- `400`
- `404`
- `503`

In gRPC, you return gRPC status codes like:

- `codes.InvalidArgument`
- `codes.NotFound`
- `codes.AlreadyExists`
- `codes.Unavailable`
- `codes.FailedPrecondition`

These come from:

- `google.golang.org/grpc/codes`

## 25. How Does Error Mapping Work In Your Code?

Your use case returns normal Go errors.

Then the gRPC delivery layer maps them to gRPC status codes.

Example in doctor-service:

- invalid input -> `codes.InvalidArgument`
- duplicate email -> `codes.AlreadyExists`
- doctor not found -> `codes.NotFound`

Example in appointment-service:

- invalid input -> `codes.InvalidArgument`
- doctor not found during remote validation -> `codes.FailedPrecondition`
- doctor-service unavailable -> `codes.Unavailable`
- appointment not found -> `codes.NotFound`
- invalid status transition -> `codes.InvalidArgument`

This is the correct place for mapping because:

- use case should not know transport details

## 26. Why Is Doctor Not Found `FailedPrecondition` In Appointment Create?

Because when creating an appointment, the main resource is the appointment.

The appointment cannot be created because a prerequisite is not satisfied:

- doctor must exist first

That is why this is modeled as:

- `codes.FailedPrecondition`

## 27. Why Is Doctor Service Unreachable `Unavailable`?

Because the problem is not invalid user data.

The problem is:

- downstream service could not be reached
- network or remote service is unavailable

So the correct gRPC code is:

- `codes.Unavailable`

## 28. What Is the 3-Second Timeout?

In [internal/appointment/client/doctor_service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/client/doctor_service.go), the outbound call uses:

```go
context.WithTimeout(ctx, 3*time.Second)
```

That means:

- appointment-service will not wait forever for doctor-service
- after 3 seconds, the call fails

Why this is useful:

- avoids hanging forever
- gives predictable failure behavior

## 29. Why Is Logging Important?

When doctor-service is unavailable, the client should get a useful gRPC error.

But the server also needs internal logs for debugging.

That is why appointment use case logs lookup failures.

This helps answer:

- what failed?
- when did it fail?
- for which `doctor_id`?

## 30. What Business Rules Exist?

### Doctor rules

- `full_name` required
- `email` required
- `email` must be valid
- `email` must be unique
- `specialization` is optional

### Appointment rules

- `title` required
- `doctor_id` required
- doctor must exist in doctor-service
- status must be:
  - `new`
  - `in_progress`
  - `done`
- `done -> new` is forbidden

## 31. Where Exactly Do Business Rules Live?

Doctor rules:

- [internal/doctor/usecase/service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/doctor/usecase/service.go)

Appointment rules:

- [internal/appointment/usecase/service.go](/Users/myrzanizimbetov/Desktop/med-go/internal/appointment/usecase/service.go)

Important defense point:

- handlers/transport do not implement business logic
- use case layer implements business logic

## 32. What Is the Repository Interface Doing?

Use cases do not depend directly on MongoDB code.

Instead, they depend on interfaces like:

- create doctor
- get doctor
- list doctors
- exists by email

This is useful because:

- use case does not care if data comes from Mongo or memory
- code is easier to test
- dependency inversion is preserved

## 33. Why Do We Have Memory Repositories?

Memory repositories are useful for:

- tests
- simple development experiments
- proving interface-based design

Mongo repositories are used in real app wiring.

## 34. What Is Dependency Inversion Here?

Dependency inversion means:

- use case depends on abstraction, not concrete implementation

Examples:

- doctor use case depends on repository interface
- appointment use case depends on repository interface
- appointment use case depends on `DoctorLookup` interface

This is good because:

- use case is not tightly coupled to MongoDB
- use case is not tightly coupled to a concrete gRPC client

## 35. Why Is `DoctorLookup` Important?

In appointment use case:

- doctor existence must be checked

But use case should not directly create a gRPC connection.

So it depends on an interface:

- `DoctorLookup`

Then app wiring injects a concrete implementation:

- the gRPC doctor client

This keeps the use case clean.

## 36. What Is The End-To-End Flow Between Services?

This is one of the best things to explain in defense.

### CreateAppointment end-to-end

1. external client calls `AppointmentService.CreateAppointment`
2. appointment gRPC server receives request
3. server maps proto -> use case input
4. use case validates title and doctor_id
5. use case calls `DoctorLookup.Exists`
6. appointment gRPC client calls `DoctorService.GetDoctor`
7. doctor-service gRPC server receives request
8. doctor-service use case gets doctor from repository
9. doctor-service returns either:
   - doctor data
   - `NotFound`
10. appointment gRPC client interprets the result
11. appointment use case either:
   - continues
   - returns doctor-not-found error
   - returns doctor-service-unavailable error
12. appointment gRPC server maps that error to gRPC status code
13. client receives final result

## 37. How Do You Test The Services?

You use `grpcurl`.

Guide file:

- [grpcurl_commands.md](/Users/myrzanizimbetov/Desktop/med-go/grpcurl_commands.md)

Why `grpcurl` is useful:

- easy manual testing
- good for defense/demo
- shows exact request and response behavior

## 38. How To Run The Project?

### Start MongoDB

Example:

```bash
docker run --name med-go-mongo -p 27017:27017 -d mongo:8
```

### Start both services

```bash
go run .
```

Or start separately:

```bash
go run ./cmd/doctor-service
go run ./cmd/appointment-service
```

## 39. How To Regenerate Proto Code?

Run:

```bash
bash scripts/generate_proto.sh
```

This is important because on defense you may be asked:

- "How were the protobuf files generated?"

Short answer:

- `.proto` defines the contract
- `protoc` generates Go stubs
- `scripts/generate_proto.sh` runs that generation

## 40. What Are The Advantages Of gRPC Over REST Here?

You should know at least a few.

### Advantage 1: Strong contract

REST often relies on informal JSON agreements.

gRPC gives:

- exact `.proto` contract
- generated server code
- generated client code

### Advantage 2: Better for service-to-service communication

gRPC is very suitable for internal microservice communication because:

- compact binary format
- strongly typed
- generated clients reduce manual coding

### Advantage 3: Standardized error model

gRPC gives standard codes like:

- `InvalidArgument`
- `NotFound`
- `Unavailable`

That makes service-to-service error propagation clearer.

## 41. What Are The Advantages Of REST Over gRPC?

Also know the trade-offs.

REST is easier when:

- humans want to test quickly in browser/Postman
- public APIs need to be very accessible
- JSON readability matters more than strict contracts

So a good defense answer is:

- REST is simpler for public-facing APIs and debugging
- gRPC is stronger for internal service-to-service communication

## 42. What Would Be Bad Design Here?

Bad examples:

- using protobuf types directly in use case
- putting business rules in gRPC handler
- appointment use case directly creating `grpc.NewClient(...)`
- appointment-service reading doctor Mongo collection directly
- returning plain Go errors from handler without gRPC status mapping

These are important because they are exactly the kinds of mistakes instructors look for.

## 43. What Should You Say If Asked "How Does gRPC Work?"

Use this short answer:

> gRPC works through a contract-first approach. I define services and messages in `.proto` files, then generate Go code from them using `protoc`. The generated code gives me server interfaces and client stubs. My server implements the generated service interface, and my client calls the generated stub methods. In this project, the transport layer maps proto requests into use case inputs, the use case runs business logic, and the result is mapped back into proto responses.

## 44. What Should You Say If Asked "Where Is Clean Architecture Preserved?"

Use this short answer:

> Clean Architecture is preserved because only the transport layer changed from REST to gRPC. Domain models and use cases still do not depend on transport details. The gRPC server is a thin delivery layer, repositories still handle persistence, and the appointment use case depends on an interface for doctor lookup instead of directly depending on the concrete gRPC client.

## 45. What Should You Say If Asked "How Does Appointment Service Validate Doctors?"

Use this short answer:

> Before creating an appointment, the appointment use case calls a `DoctorLookup` interface. The concrete implementation is a gRPC client that calls `DoctorService.GetDoctor`. If doctor-service returns not found, appointment-service rejects creation with `FailedPrecondition`. If doctor-service is unreachable, appointment-service returns `Unavailable`.

## 46. What Should You Say If Asked "Why Is This Not a Distributed Monolith?"

Use this short answer:

> Because the services have explicit boundaries and separate ownership. Appointment-service does not read doctor-service storage directly. It communicates through a well-defined gRPC contract. Each service owns its own business rules and repository implementation.

## 47. What Should You Say If Asked "What Changed From Assignment 1?"

Use this short answer:

> The transport changed from REST/Gin to gRPC/Protocol Buffers. The domain models, use cases, repositories, and business rules stayed the same. The appointment service no longer calls doctor-service over REST; it now uses a gRPC client stub.

## 48. Quick Memory Map Of The Project

If you forget where things are, remember this:

- contract -> `internal/*/proto/*.proto`
- generated code -> `internal/*/proto/*.pb.go`
- inbound gRPC server -> `internal/*/transport/grpc/server.go`
- outbound doctor client -> `internal/appointment/client/doctor_service.go`
- business rules -> `internal/*/usecase/service.go`
- database access -> `internal/*/repository/*.go`
- server wiring -> `internal/*/app/app.go`
- startup -> `main.go`, `cmd/.../main.go`

## 49. Defense Checklist

Before defense, make sure you can explain:

- what gRPC is
- what `.proto` is
- why stubs are generated
- what `go_package` means
- difference between proto messages and domain models
- why use case does not import protobuf types
- how appointment-service calls doctor-service
- why `FailedPrecondition` is used for missing doctor on appointment create
- why `Unavailable` is used when doctor-service is down
- how to run `go run .`
- how to regenerate stubs
- how to demo with `grpcurl`

## 50. Final One-Minute Explanation

If you need one compact explanation for the whole project, use this:

> This project is a two-service medical scheduling platform migrated from REST to gRPC. The service contracts are defined in `.proto` files, and Go server/client stubs are generated with `protoc`. The gRPC server layer is thin: it maps proto requests to use case inputs, calls business logic, and maps results back to proto responses with proper gRPC status codes. The domain and use case layers remain transport-agnostic. Appointment-service validates doctor existence by calling doctor-service through a generated gRPC client stub hidden behind an interface, which preserves Clean Architecture and explicit microservice boundaries.
