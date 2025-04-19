# Stage 1: Build the Go binary
FROM golang:1.23 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary (statically linked)
RUN CGO_ENABLED=0 go build -o bin/backup cmd/backup/main.go
RUN CGO_ENABLED=0 go build -o bin/mgr cmd/mgr/main.go
RUN CGO_ENABLED=0 go build -o bin/proxy cmd/proxy/main.go

# Stage 2: Final image using distroless
FROM gcr.io/distroless/cc-debian12

WORKDIR /app

# Copy only the binary
COPY --from=builder /app/bin/* /app/
COPY migration /app/migration
