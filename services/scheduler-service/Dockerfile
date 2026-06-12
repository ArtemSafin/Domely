FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY services/scheduler-service/go.mod services/scheduler-service/go.sum ./
RUN go mod download

COPY services/scheduler-service/ .

RUN go build -o scheduler ./cmd/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/scheduler .

CMD ["./scheduler"]