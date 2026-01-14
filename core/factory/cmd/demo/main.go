package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bsmider/vibe/core/factory"
)

func main() {
	outputDir := "/Users/brevin/abc/projects/pipes/core/generated"

	config := factory.CodeGenConfig{
		OutputDir:         outputDir,
		ProcessesImport:   "github.com/bsmider/vibe/core/factory/processes",
		ProtoImportPath:   "github.com/bsmider/vibe/core/factory/build/example",
		ProtoPackageAlias: "example",
	}

	// Create output directory
	os.MkdirAll(config.OutputDir, 0755)

	servicePath := "/Users/brevin/abc/projects/pipes/core/example/book_service.go"

	// Get service methods first
	methods, err := factory.GetServiceMethods(servicePath)
	if err != nil {
		fmt.Printf("Error getting methods: %v\n", err)
		return
	}
	fmt.Printf("Found %d RPC service methods:\n", len(methods))
	for _, m := range methods {
		fmt.Printf("  - %s\n", m)
	}

	generatedMethods, err := factory.GenerateFromServiceFile(servicePath, config)
	if err != nil {
		fmt.Printf("Error generating methods: %v\n", err)
		return
	}

	// Generate Orchestrator
	if err := factory.GenerateOrchestrator(generatedMethods, config); err != nil {
		fmt.Printf("Error generating orchestrator: %v\n", err)
		return
	}

	// Generate Dockerfile
	if err := factory.GenerateDockerfile(generatedMethods, config); err != nil {
		fmt.Printf("Error generating Dockerfile: %v\n", err)
		return
	}

	// List generated files
	fmt.Println("\nGenerated files:")
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fmt.Printf("  - %s\n", info.Name())
		}
		return nil
	})

	fmt.Println("\nGeneration complete!")
}
