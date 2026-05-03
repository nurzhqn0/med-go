#!/usr/bin/env bash
set -euo pipefail

echo "Checking for active MongoDB references..."
if rg 'go\.mongodb|MONGODB|NewMongoRepository|bson\.NewObjectID' \
  --glob '!GRPC_STUDY_GUIDE.md' \
  --glob '!go.sum' \
  --glob '!scripts/verify_assignment3.sh' \
  .; then
  echo "Unexpected MongoDB reference found"
  exit 1
fi

echo "Checking DDL exists only in migration files..."
if rg 'CREATE TABLE|ALTER TABLE|DROP TABLE' \
  --glob '!doctor-service/migrations/*.sql' \
  --glob '!appointment-service/migrations/*.sql' \
  --glob '!scripts/verify_assignment3.sh' \
  .; then
  echo "DDL found outside migration files"
  exit 1
fi

echo "Checking required migration files..."
test -f doctor-service/migrations/000001_create_doctors.up.sql
test -f doctor-service/migrations/000001_create_doctors.down.sql
test -f appointment-service/migrations/000001_create_appointments.up.sql
test -f appointment-service/migrations/000001_create_appointments.down.sql

echo "Checking required runnable service entrypoints..."
test -f doctor-service/main.go
test -f appointment-service/main.go
test -f notification-service/main.go

echo "Checking expected event subjects..."
rg 'doctors\.created' internal/doctor internal/notification README.md grpcurl_commands.md >/dev/null
rg 'appointments\.created' internal/appointment internal/notification README.md grpcurl_commands.md >/dev/null
rg 'appointments\.status_updated' internal/appointment internal/notification README.md grpcurl_commands.md >/dev/null

echo "Running Go tests..."
GOCACHE="${GOCACHE:-/private/tmp/med-go-gocache}" go test ./...

echo "Checking Docker Compose config..."
docker compose config >/dev/null

echo "Assignment 3 static verification passed."
