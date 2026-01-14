FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the entire core module to resolve dependencies
COPY core/go.mod core/go.sum ./core/
RUN cd core && go mod download

COPY core/ ./core/

# Build GetBook
RUN cd core/generated/example/book_service/get_book && go build -o /49923a45240c_GetBook main.go

# Build GetAuthorNameFromBookId
RUN cd core/generated/example/book_service/get_author_name_from_book_id && go build -o /b10ac789a6d1_GetAuthorNameFromBookId main.go

# Build orchestrator
RUN cd core/generated/orchestrator && go build -o /orchestrator main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache libc6-compat

# Copy binaries
COPY --from=builder /49923a45240c_GetBook .
COPY --from=builder /b10ac789a6d1_GetAuthorNameFromBookId .
COPY --from=builder /orchestrator .

# Run orchestrator
CMD ["./orchestrator"]
