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

	// Get relative path to output directory for build context references
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get cwd: %w", err)
	}
	relOutputDir, err := filepath.Rel(cwd, config.OutputDir)
	if err != nil {
		return fmt.Errorf("failed to get relative output dir: %w", err)
	}

	buf.WriteString("FROM golang:1.24-alpine AS builder\n\n")
	buf.WriteString("WORKDIR /app\n\n")

	// Copy go.mod and go.sum first for caching
	buf.WriteString("# Copy metadata to resolve dependencies\n")
	buf.WriteString("COPY go.mod go.sum ./\n")
	buf.WriteString("RUN go mod download\n\n")

	// Copy the source code
	buf.WriteString("COPY . ./\n\n")

	// Build each RPC method
	for _, method := range methods {
		// Calculate the directory to build in
		// method.RelativePath is like "example/book_service/get_book/main.go"

		// valid relative path for `cd`
		dirPath := filepath.Dir(method.RelativePath)
		buildDir := filepath.Join(relOutputDir, dirPath)

		buf.WriteString(fmt.Sprintf("# Build %s\n", method.MethodName))
		// We use the ShortID for the binary identification
		buf.WriteString(fmt.Sprintf("RUN cd %s && go build -o /%s main.go\n\n", buildDir, method.ShortID))
	}

	// Build Orchestrator
	buf.WriteString("# Build orchestrator\n")
	orchestratorDir := filepath.Join(relOutputDir, "orchestrator")
	buf.WriteString(fmt.Sprintf("RUN cd %s && go build -o /orchestrator main.go\n\n", orchestratorDir))

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
