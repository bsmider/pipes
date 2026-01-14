package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/bsmider/pipes/core/factory"
)

func main() {
	// Use relative paths for portability (works in Docker and local)
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current working directory: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Join(cwd, "example/generated")
	serviceDir := filepath.Join(cwd, "example/src")

	config := factory.CodeGenConfig{
		OutputDir: outputDir,
		SrcDir:    serviceDir,
	}

	if err := factory.Build(config); err != nil {
		fmt.Printf("Build failed: %v\n", err)
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

	// Spawn the Docker container
	fmt.Println("\nBuilding Docker image...")
	dockerfile := filepath.Join(outputDir, "Dockerfile")
	buildCmd := exec.Command("docker", "build", "-t", "pipes-generated-example", "-f", dockerfile, ".")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Printf("Failed to build Docker image: %v\n", err)
		return
	}

	containerName := "pipes-generated-example-run"
	fmt.Printf("Running Docker container: %s...\n", containerName)
	// cleanup existing container if any
	exec.Command("docker", "rm", "-f", containerName).Run()

	runCmd := exec.Command("docker", "run", "--rm", "--name", containerName, "pipes-generated-example")
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr
	if err := runCmd.Run(); err != nil {
		fmt.Printf("Failed to run Docker container: %v\n", err)
		return
	}
}
