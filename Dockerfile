FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION
ARG REVISION
ARG BUILD
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "\
        -X main.version=${VERSION} \
        -X main.revision=${REVISION} \
        -X main.build=${BUILD} \
	-s -w" -o datadog-sql-metrics .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/datadog-sql-metrics .
COPY --from=builder /app/config.yaml .
ENTRYPOINT ["./datadog-sql-metrics"]
