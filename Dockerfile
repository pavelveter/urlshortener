# Stage 1: Build
FROM golang:1.22.6-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o app main.go

# Stage 2: Run
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/app /app/
COPY config.ini urls.txt ./
EXPOSE 8081
CMD ["./app"]