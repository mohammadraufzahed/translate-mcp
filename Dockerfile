# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o translate-mcp ./cmd/translate-mcp

FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=builder /app/translate-mcp /app/translate-mcp

EXPOSE 8787

USER nonroot:nonroot

ENTRYPOINT ["/app/translate-mcp"]
