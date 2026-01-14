FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy metadata to resolve dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . ./

# Build GetBook
RUN cd example/generated/example/book_service/get_book && go build -o /f5bcc3da3077_GetBook main.go

# Build GetAuthorNameFromBookId
RUN cd example/generated/example/book_service/get_author_name_from_book_id && go build -o /743aee161164_GetAuthorNameFromBookId main.go

# Build GetAuthor
RUN cd example/generated/example/book_service/get_author && go build -o /3dbb7c569bfe_GetAuthor main.go

# Build orchestrator
RUN cd example/generated/orchestrator && go build -o /orchestrator main.go

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache libc6-compat

# Copy binaries
COPY --from=builder /f5bcc3da3077_GetBook .
COPY --from=builder /743aee161164_GetAuthorNameFromBookId .
COPY --from=builder /3dbb7c569bfe_GetAuthor .
COPY --from=builder /orchestrator .

# Run orchestrator
CMD ["./orchestrator"]
