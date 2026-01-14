FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the entire core module to resolve dependencies
COPY core/go.mod core/go.sum ./core/
RUN cd core && go mod download

COPY core/ ./core/

# Build get_author
RUN cd core/factory/example/get_author && go build -o /get_author get_author.go

# Build get_book
RUN cd core/factory/example/get_book && go build -o /get_book get_book.go

# Build orchestrator
RUN cd core/factory/example/orchestrator && go build -o /orchestrator orchestrator.go

FROM alpine:latest

WORKDIR /app

# Install dependencies for running go binaries if needed (usually not needed for static builds, but alpine might need some libs)
RUN apk add --no-cache libc6-compat

# Copy binaries
COPY --from=builder /get_book .
COPY --from=builder /get_author .
COPY --from=builder /orchestrator .

# Run orchestrator
CMD ["./orchestrator"]
