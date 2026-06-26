FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bot ./cmd/bot

FROM python:3.12-alpine
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /build/bot .
COPY prompts/ ./prompts/
COPY tools/ ./tools/
RUN pip install --no-cache-dir beautifulsoup4 lxml

EXPOSE 8080
ENTRYPOINT ["./bot"]
