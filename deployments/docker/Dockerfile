FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o bot ./cmd/main.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/bot .
CMD ["./bot"]
