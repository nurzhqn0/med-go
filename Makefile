DOCTOR_DB_URL ?= postgres://postgres:postgres@localhost:5433/doctor_service?sslmode=disable
APPOINTMENT_DB_URL ?= postgres://postgres:postgres@localhost:5433/appointment_service?sslmode=disable
NATS_URL ?= nats://localhost:4222
REDIS_URL ?= redis://localhost:6379
GATEWAY_URL ?= http://localhost:8080
CACHE_TTL_SECONDS ?= 60
RATE_LIMIT_RPM ?= 100
WORKER_POOL_SIZE ?= 3
GATEWAY_PORT ?= 8080
JOB_MAX_RETRIES ?= 3
JOB_BACKOFF_SECONDS ?= 1,2,4
DOCTOR_ADDR ?= :8081
APPOINTMENT_ADDR ?= :8082
DOCTOR_TARGET ?= 127.0.0.1:8081
GOCACHE_DIR ?= /private/tmp/med-go-gocache

.PHONY: run test infra-up stack-up stack-down infra-down doctor appointment notification gateway migrate-up migrate-down verify docker-config

run:
	go run .

test:
	GOCACHE=$(GOCACHE_DIR) go test ./...

infra-up:
	docker compose up -d postgres nats redis

stack-up:
	docker compose up --build

stack-down:
	docker compose down

infra-down:
	docker compose down

docker-config:
	docker compose config

doctor:
	cd doctor-service && DATABASE_URL="$(DOCTOR_DB_URL)" NATS_URL="$(NATS_URL)" REDIS_URL="$(REDIS_URL)" CACHE_TTL_SECONDS="$(CACHE_TTL_SECONDS)" RATE_LIMIT_RPM="$(RATE_LIMIT_RPM)" DOCTOR_SERVICE_ADDR="$(DOCTOR_ADDR)" go run .

appointment:
	cd appointment-service && DATABASE_URL="$(APPOINTMENT_DB_URL)" NATS_URL="$(NATS_URL)" REDIS_URL="$(REDIS_URL)" CACHE_TTL_SECONDS="$(CACHE_TTL_SECONDS)" RATE_LIMIT_RPM="$(RATE_LIMIT_RPM)" APPOINTMENT_SERVICE_ADDR="$(APPOINTMENT_ADDR)" DOCTOR_SERVICE_GRPC_TARGET="$(DOCTOR_TARGET)" go run .

notification:
	cd notification-service && NATS_URL="$(NATS_URL)" REDIS_URL="$(REDIS_URL)" GATEWAY_URL="$(GATEWAY_URL)" WORKER_POOL_SIZE="$(WORKER_POOL_SIZE)" JOB_MAX_RETRIES="$(JOB_MAX_RETRIES)" JOB_BACKOFF_SECONDS="$(JOB_BACKOFF_SECONDS)" go run .

gateway:
	cd mock-gateway && GATEWAY_PORT="$(GATEWAY_PORT)" go run .

migrate-up:
	migrate -path doctor-service/migrations -database "$(DOCTOR_DB_URL)" up
	migrate -path appointment-service/migrations -database "$(APPOINTMENT_DB_URL)" up

migrate-down:
	migrate -path doctor-service/migrations -database "$(DOCTOR_DB_URL)" down 1
	migrate -path appointment-service/migrations -database "$(APPOINTMENT_DB_URL)" down 1

verify:
	bash scripts/verify_assignment3.sh
