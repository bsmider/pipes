package factory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFromServiceFile(t *testing.T) {
	// Get the path to the example service file
	servicePath := "../example/book_service.go"

	// Create a temporary output directory
	tempDir, err := os.MkdirTemp("", "codegen_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := CodeGenConfig{
		OutputDir: tempDir,
	}

	_, err = GenerateFromServiceFile(servicePath, config)
	if err != nil {
		t.Fatalf("GenerateFromServiceFile failed: %v", err)
	}

	// Check that the file was created in the new directory structure:
	// {proto_package}/{service_name}/{method_name}/main.go
	getBookPath := filepath.Join(tempDir, "example", "book_service", "get_book", "main.go")
	if _, err := os.Stat(getBookPath); os.IsNotExist(err) {
		t.Errorf("Expected main.go at %s to be created", getBookPath)
	}

	// Read and verify the content
	content, err := os.ReadFile(getBookPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify package is main
	if !strings.Contains(contentStr, "package main") {
		t.Error("Generated file should have package main")
	}

	// Verify processes.Call is used with full unique method ID
	if !strings.Contains(contentStr, "processes.Call[") {
		t.Error("Generated file should contain processes.Call")
	}

	// Verify the unique method ID is included in the processes.Call
	if !strings.Contains(contentStr, "github.com/bsmider/pipes/core/factory/build/example.BookService.") {
		t.Error("Generated file should use full unique method ID in processes.Call")
	}

	// Verify the function signature is standalone (no receiver)
	if strings.Contains(contentStr, "func (s *BookService)") {
		t.Error("Generated file should not have receiver")
	}

	// Verify main function exists
	if !strings.Contains(contentStr, "func main()") {
		t.Error("Generated file should have main function")
	}

	// Verify processes.Handle is called
	if !strings.Contains(contentStr, "processes.Handle(GetBook)") {
		t.Error("Generated file should call processes.Handle(GetBook)")
	}

	// Verify the other method was also generated
	getAuthorPath := filepath.Join(tempDir, "example", "book_service", "get_author_name_from_book_id", "main.go")
	if _, err := os.Stat(getAuthorPath); os.IsNotExist(err) {
		t.Errorf("Expected main.go at %s to be created", getAuthorPath)
	}
}

func TestParsesServiceFile(t *testing.T) {
	servicePath := "../example/book_service.go"

	methods, err := GetServiceMethods(servicePath)
	if err != nil {
		t.Fatalf("GetServiceMethods failed: %v", err)
	}

	if len(methods) == 0 {
		t.Error("Expected at least one method")
	}

	// Check that GetBook is in the methods
	found := false
	for _, m := range methods {
		if m == "GetBook" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected GetBook method to be found")
	}
}

func TestValidateServiceFile(t *testing.T) {
	servicePath := "../example/book_service.go"

	err := ValidateServiceFile(servicePath)
	if err != nil {
		t.Errorf("ValidateServiceFile should pass for valid service: %v", err)
	}
}
