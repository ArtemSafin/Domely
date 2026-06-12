FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY services/task-service/go.mod services/task-service/go.sum ./
RUN go mod download

COPY services/task-service/ .

RUN go build -o server ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY migrations/ ./migrations/

EXPOSE 8080

CMD ["./server"]