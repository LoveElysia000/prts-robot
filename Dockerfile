FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

FROM python:3.12-alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bot .
RUN mkdir -p /app/data

EXPOSE 8080
ENTRYPOINT ["./bot"]
