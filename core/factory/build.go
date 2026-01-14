package factory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Build runs the code generation process for all service files in a directory
func Build(config CodeGenConfig) error {
	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	files, err := os.ReadDir(config.SrcDir)
	if err != nil {
		return fmt.Errorf("error reading service directory: %w", err)
	}

	var allGeneratedMethods []MethodInfo

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".go" || strings.HasSuffix(file.Name(), "_test.go") {
			continue
		}

		servicePath := filepath.Join(config.SrcDir, file.Name())

		// Get service methods first
		methods, err := GetServiceMethods(servicePath)
		if err != nil {
			fmt.Printf("Skipping file %s: %v\n", file.Name(), err)
			continue
		}
		fmt.Printf("Found %d RPC service methods in %s:\n", len(methods), file.Name())
		for _, m := range methods {
			fmt.Printf("  - %s\n", m)
		}

		generatedMethods, err := GenerateFromServiceFile(servicePath, config)
		if err != nil {
			fmt.Printf("Error generating methods for %s: %v\n", file.Name(), err)
			continue
		}
		allGeneratedMethods = append(allGeneratedMethods, generatedMethods...)
	}

	// Generate Orchestrator
	if err := GenerateOrchestrator(allGeneratedMethods, config); err != nil {
		return fmt.Errorf("error generating orchestrator: %w", err)
	}

	// Generate Dockerfile
	if err := GenerateDockerfile(allGeneratedMethods, config); err != nil {
		return fmt.Errorf("error generating Dockerfile: %w", err)
	}

	return nil
}
