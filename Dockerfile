FROM golang:1.22.6-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum main.go ./
RUN go mod download
COPY . .
RUN go build -o app main.go
EXPOSE 8081
CMD ["./app"]
