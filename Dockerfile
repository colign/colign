# Build stage
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

# Install buf and proto plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest && \
    go install github.com/bufbuild/buf/cmd/buf@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate proto (Go only)
RUN cd proto && buf generate --template buf.gen.go.yaml

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/mcp ./cmd/mcp

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 1001 appuser

COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/mcp /bin/mcp
COPY --from=builder /app/migrations /migrations

USER appuser

EXPOSE 8080

CMD ["/bin/api"]
