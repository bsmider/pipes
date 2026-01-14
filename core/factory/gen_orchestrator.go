package factory

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// GenerateOrchestrator generates the main.go for the orchestrator
// which spawns all the generated RPC worker processes.
func GenerateOrchestrator(methods []MethodInfo, config CodeGenConfig) error {
	var buf bytes.Buffer

	// Package declaration
	buf.WriteString("package main\n\n")

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"github.com/bsmider/pipes/core/factory/orchestrator\"\n")
	buf.WriteString("\t\"log\"\n")
	buf.WriteString(")\n\n")

	// Main function
	buf.WriteString("func main() {\n")
	buf.WriteString("\torch := orchestrator.NewOrchestrator()\n")
	buf.WriteString("\n")

	for _, method := range methods {
		// Use the ShortID for the binary name to match what we'll generate in the Dockerfile
		// Binary path is relative to the working directory in the container (WORKDIR is /app)
		binaryPath := fmt.Sprintf("./%s", method.ShortID)

		buf.WriteString(fmt.Sprintf("\tif err := orch.Spawn(\"%s\", \"%s\", 1); err != nil {\n", method.MethodID, binaryPath))
		buf.WriteString(fmt.Sprintf("\t\tlog.Fatalf(\"Failed to spawn worker for %%s: %%v\", \"%s\", err)\n", method.MethodName))
		buf.WriteString("\t}\n")
	}

	buf.WriteString("\n")
	buf.WriteString("\t// Block forever to keep the orchestrator running\n")
	buf.WriteString("\tselect {}\n")
	buf.WriteString("}\n")

	// Output directory for orchestrator
	orchDir := filepath.Join(config.OutputDir, "orchestrator")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		return fmt.Errorf("failed to create orchestrator directory: %w", err)
	}

	outputPath := filepath.Join(orchDir, "main.go")
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write orchestrator main.go: %w", err)
	}

	return nil
}
