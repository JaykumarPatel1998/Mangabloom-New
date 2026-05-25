#Stage 1 : Build the binary
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-service ./cmd/api

#Stage 2 : Create the final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/api-service .
#if your api loads queries/templates, copy them here
# COPY --from=builder /app/migrations ./migrations

EXPOSE 3000
CMD ["./api-service"]