FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o burner-pod ./cmd/web

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/burner-pod .
COPY ui ./ui

EXPOSE 8080

CMD ["./burner-pod"]