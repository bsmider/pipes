package factory

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// GenerateDockerfile generates a Dockerfile to compile all RPC methods and the orchestrator
func GenerateDockerfile(methods []MethodInfo, config CodeGenConfig) error {
	var buf bytes.Buffer

	buf.WriteString("FROM golang:1.24-alpine AS builder\n\n")
	buf.WriteString("WORKDIR /app\n\n")

	// Copy go.mod and go.sum first for caching
	buf.WriteString("# Copy the entire core module to resolve dependencies\n")
	buf.WriteString("COPY core/go.mod core/go.sum ./core/\n")
	buf.WriteString("RUN cd core && go mod download\n\n")

	// Copy the source code
	buf.WriteString("COPY core/ ./core/\n\n")

	// Build each RPC method
	for _, method := range methods {
		// Calculate the directory to build in
		// Assumes the generated files are in core/generated/, so inside the container
		// they are at /app/core/generated/
		// method.RelativePath is like "example/book_service/get_book/main.go"

		// valid relative path for `cd`
		dirPath := filepath.Dir(method.RelativePath)
		buildDir := filepath.Join("core/generated", dirPath)

		buf.WriteString(fmt.Sprintf("# Build %s\n", method.MethodName))
		// We use the ShortID for the binary identification
		buf.WriteString(fmt.Sprintf("RUN cd %s && go build -o /%s main.go\n\n", buildDir, method.ShortID))
	}

	// Build Orchestrator
	buf.WriteString("# Build orchestrator\n")
	buf.WriteString("RUN cd core/generated/orchestrator && go build -o /orchestrator main.go\n\n")

	// Final Stage
	buf.WriteString("FROM alpine:latest\n\n")
	buf.WriteString("WORKDIR /app\n\n")
	buf.WriteString("RUN apk add --no-cache libc6-compat\n\n")

	// Copy binaries
	buf.WriteString("# Copy binaries\n")
	for _, method := range methods {
		buf.WriteString(fmt.Sprintf("COPY --from=builder /%s .\n", method.ShortID))
	}
	buf.WriteString("COPY --from=builder /orchestrator .\n\n")

	// Run orchestrator
	buf.WriteString("# Run orchestrator\n")
	buf.WriteString("CMD [\"./orchestrator\"]\n")

	// Write Dockerfile to OutputDir
	outputPath := filepath.Join(config.OutputDir, "Dockerfile")
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	return nil
}
