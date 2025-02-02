FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o datadog-sql-metics .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/datadog-sql-metics .
COPY --from=builder /app/config.yaml .
ENTRYPOINT ["./datadog-sql-metics"]
