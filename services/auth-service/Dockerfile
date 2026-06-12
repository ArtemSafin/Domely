FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY services/auth-service/go.mod services/auth-service/go.sum ./
RUN go mod download

COPY services/auth-service/ .

RUN go build -o auth ./cmd/main.go

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/auth .

CMD ["./auth"]