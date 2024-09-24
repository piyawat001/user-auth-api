# Build stage
FROM golang:1.22.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Run stage
FROM alpine:3.14

WORKDIR /app

COPY --from=builder /app/main .
COPY .env .

EXPOSE 3000

CMD ["./main"]