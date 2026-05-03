FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/med-go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/doctor-service ./doctor-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/appointment-service ./appointment-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/notification-service ./notification-service

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/med-go /app/bin/med-go
COPY --from=builder /out/doctor-service /app/bin/doctor-service
COPY --from=builder /out/appointment-service /app/bin/appointment-service
COPY --from=builder /out/notification-service /app/bin/notification-service
COPY doctor-service/migrations /app/doctor-service/migrations
COPY appointment-service/migrations /app/appointment-service/migrations

EXPOSE 8081 8082

CMD ["/app/bin/med-go"]
