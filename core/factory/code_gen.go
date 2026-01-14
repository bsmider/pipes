package factory

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsmider/vibe/core/factory/utils"
)

// CodeGenConfig contains configuration for code generation
type CodeGenConfig struct {
	OutputDir         string // Directory where generated files will be written
	ProcessesImport   string // Import path for the processes package
	ProtoImportPath   string // Import path for proto messages (e.g., "github.com/bsmider/vibe/core/factory/build/example")
	ProtoPackageAlias string // Package alias for proto imports (e.g., "example")
}

// DefaultCodeGenConfig returns the default configuration for code generation
func DefaultCodeGenConfig() CodeGenConfig {
	return CodeGenConfig{
		OutputDir:         "./generated",
		ProcessesImport:   "github.com/bsmider/vibe/core/factory/processes",
		ProtoPackageAlias: "example",
	}
}

// GenerateFromServiceFile generates code files for all RPC methods in a service file
func GenerateFromServiceFile(servicePath string, config CodeGenConfig) ([]MethodInfo, error) {
	// Parse the service file
	parsed, err := utils.ParseServiceFile(servicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service file: %w", err)
	}

	// Use discovered proto import path if not specified
	if config.ProtoImportPath == "" {
		config.ProtoImportPath = parsed.ProtoImportPath
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Collect all method names for RPC call detection
	methodNames := make([]string, len(parsed.Methods))
	for i, m := range parsed.Methods {
		methodNames[i] = m.Name
	}

	var methods []MethodInfo

	// Generate a file for each method
	for _, method := range parsed.Methods {
		info, err := generateMethodFile(servicePath, method, parsed, methodNames, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate file for method %s: %w", method.Name, err)
		}
		methods = append(methods, *info)
	}

	return methods, nil
}

// generateMethodFile generates a single Go file for an RPC method
func generateMethodFile(servicePath string, method utils.ServiceMethod, parsed *utils.ParsedServiceFile, methodNames []string, config CodeGenConfig) (*MethodInfo, error) {
	// Read the original source file
	sourceBytes, err := os.ReadFile(servicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}
	source := string(sourceBytes)

	// Extract and transform the function body
	transformedBody, err := transformMethodBody(servicePath, source, method, parsed, methodNames, config)
	if err != nil {
		return nil, fmt.Errorf("failed to transform method body: %w", err)
	}

	// Generate the output file content
	output := generateFileContent(method, parsed, transformedBody, config)

	// Generate unique directory path based on proto package, service, and method
	// This ensures uniqueness even with same method names across different services
	methodDir := utils.GenerateDirPath(config.ProtoImportPath, parsed.ServiceName, method.Name)
	outputDir := filepath.Join(config.OutputDir, methodDir)

	// Create the method-specific directory (including parent directories)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create method directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, "main.go")

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}

	// Construct relative path for Dockerfile (relative from core/)
	// Assumes OutputDir is something like "core/generated"
	// We want the relative path from the config.OutputDir's parent or relevant root.
	// For now, let's store the path relative to the OutputDir and let the caller adjust.
	relPath := filepath.Join(methodDir, "main.go")

	// Generate the unique MethodID
	methodID := utils.GenerateMethodID(config.ProtoImportPath, parsed.ServiceName, method.Name)
	shortID := utils.GenerateShortMethodID(config.ProtoImportPath, parsed.ServiceName, method.Name)

	return &MethodInfo{
		MethodName:   method.Name,
		MethodID:     methodID,
		ShortID:      shortID,
		FullDirPath:  outputDir,
		RelativePath: relPath, // e.g. "example/book_service/get_book/main.go"
	}, nil
}

// transformMethodBody transforms the method body, replacing RPC calls with processes.Call
func transformMethodBody(servicePath string, source string, method utils.ServiceMethod, parsed *utils.ParsedServiceFile, methodNames []string, config CodeGenConfig) (string, error) {
	// Parse the file to get accurate positions
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, servicePath, source, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	// Find the function declaration
	var funcDecl *ast.FuncDecl
	ast.Inspect(file, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if ok && fd.Name.Name == method.Name && fd.Recv != nil {
			funcDecl = fd
			return false
		}
		return true
	})

	if funcDecl == nil || funcDecl.Body == nil {
		return "", fmt.Errorf("function %s not found", method.Name)
	}

	// Get the body content (between braces)
	bodyStart := fset.Position(funcDecl.Body.Lbrace).Offset + 1
	bodyEnd := fset.Position(funcDecl.Body.Rbrace).Offset
	body := source[bodyStart:bodyEnd]

	// Find and replace RPC calls
	rpcCalls, err := findRPCCallsInBody(funcDecl.Body, method, methodNames, fset)
	if err != nil {
		return "", err
	}

	// Replace calls in reverse order to preserve positions
	for i := len(rpcCalls) - 1; i >= 0; i-- {
		call := rpcCalls[i]

		// Find the target method to get request/response types
		targetMethod := utils.GetMethodByName(parsed.Methods, call.MethodName)
		if targetMethod == nil {
			continue
		}

		// Build the replacement call
		replacement := buildProcessesCall(call, targetMethod, parsed, config)

		// Calculate positions relative to body start
		callStartInBody := call.CallStart - bodyStart
		callEndInBody := call.CallEnd - bodyStart

		// Ensure positions are within bounds
		if callStartInBody < 0 || callEndInBody > len(body) {
			continue
		}

		body = body[:callStartInBody] + replacement + body[callEndInBody:]
	}

	return body, nil
}

// findRPCCallsInBody finds all RPC method calls in a function body
func findRPCCallsInBody(body *ast.BlockStmt, method utils.ServiceMethod, methodNames []string, fset *token.FileSet) ([]utils.RPCCall, error) {
	// Build a set of known RPC method names
	rpcMethodSet := make(map[string]bool)
	for _, name := range methodNames {
		rpcMethodSet[name] = true
	}

	var calls []utils.RPCCall

	ast.Inspect(body, func(n ast.Node) bool {
		callExpr, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if this is a selector expression (e.g., s.MethodName)
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// Check if the receiver matches our service receiver
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok || ident.Name != method.ReceiverName {
			return true
		}

		methodName := selExpr.Sel.Name

		// Check if this is an RPC method (not the current one)
		if !rpcMethodSet[methodName] || methodName == method.Name {
			return true
		}

		// Extract call arguments
		if len(callExpr.Args) < 2 {
			return true
		}

		call := utils.RPCCall{
			MethodName: methodName,
			CallStart:  fset.Position(callExpr.Pos()).Offset,
			CallEnd:    fset.Position(callExpr.End()).Offset,
			CtxArg:     astExprToString(callExpr.Args[0], fset),
			ReqArg:     astExprToString(callExpr.Args[1], fset),
		}

		calls = append(calls, call)
		return true
	})

	return calls, nil
}

// astExprToString converts an AST expression to its source string representation
func astExprToString(expr ast.Expr, fset *token.FileSet) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)
	return buf.String()
}

// buildProcessesCall builds the processes.Call replacement string
// Uses the unique method ID to ensure correct routing even with same method names across services
func buildProcessesCall(call utils.RPCCall, targetMethod *utils.ServiceMethod, parsed *utils.ParsedServiceFile, config CodeGenConfig) string {
	// Extract the type names without the pointer prefix for the generic params
	reqType := targetMethod.ReqType
	respType := targetMethod.RespType

	// Generate unique method ID based on full package path, service, and method
	methodID := utils.GenerateMethodID(config.ProtoImportPath, parsed.ServiceName, targetMethod.Name)

	return fmt.Sprintf(`processes.Call[%s, %s]("%s", %s, %s)`,
		reqType,
		respType,
		methodID,
		call.CtxArg,
		call.ReqArg,
	)
}

// generateFileContent generates the complete Go file content for a method
func generateFileContent(method utils.ServiceMethod, parsed *utils.ParsedServiceFile, transformedBody string, config CodeGenConfig) string {
	var buf bytes.Buffer

	// Package declaration
	buf.WriteString("package main\n\n")

	// Imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n")
	buf.WriteString("\t\"flag\"\n")
	buf.WriteString("\t\"log\"\n")
	buf.WriteString("\n")
	buf.WriteString(fmt.Sprintf("\t\"%s\"\n", config.ProtoImportPath))
	buf.WriteString(fmt.Sprintf("\t\"%s\"\n", config.ProcessesImport))
	buf.WriteString(")\n\n")

	// Function signature (without receiver)
	buf.WriteString(fmt.Sprintf("func %s(%s context.Context, %s %s) (%s, error) {",
		method.Name,
		method.CtxName,
		method.ReqName,
		method.ReqType,
		method.RespType,
	))

	// Function body (already transformed)
	buf.WriteString(transformedBody)
	buf.WriteString("}\n\n")

	// Main function
	buf.WriteString("func main() {\n")
	buf.WriteString("\tnodeID := flag.String(\"id\", \"default-worker\", \"The unique ID for this worker instance\")\n")
	buf.WriteString("\tflag.Parse()\n")
	buf.WriteString("\tnode := processes.GetIONode(*nodeID)\n")
	buf.WriteString("\tnode.Listen()\n")
	buf.WriteString(fmt.Sprintf("\tprocesses.Handle(%s)\n", method.Name))
	buf.WriteString("\tselect {}\n")
	buf.WriteString("}\n")

	return buf.String()
}

// QuickGenerate is a convenience function that generates code from a service file
// using sensible defaults and writes to a directory named after the method
func QuickGenerate(servicePath string, outputBaseDir string, protoImportPath string) error {
	config := DefaultCodeGenConfig()
	config.OutputDir = outputBaseDir
	config.ProtoImportPath = protoImportPath
	_, err := GenerateFromServiceFile(servicePath, config)
	return err
}

// GenerateSingleMethod generates code for a single method by name
func GenerateSingleMethod(servicePath string, methodName string, config CodeGenConfig) error {
	// Parse the service file
	parsed, err := utils.ParseServiceFile(servicePath)
	if err != nil {
		return fmt.Errorf("failed to parse service file: %w", err)
	}

	// Use discovered proto import path if not specified
	if config.ProtoImportPath == "" {
		config.ProtoImportPath = parsed.ProtoImportPath
	}

	// Find the method
	var targetMethod *utils.ServiceMethod
	for _, m := range parsed.Methods {
		if m.Name == methodName {
			targetMethod = &m
			break
		}
	}

	if targetMethod == nil {
		return fmt.Errorf("method %s not found in service file", methodName)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Collect all method names for RPC call detection
	methodNames := make([]string, len(parsed.Methods))
	for i, m := range parsed.Methods {
		methodNames[i] = m.Name
	}

	_, err = generateMethodFile(servicePath, *targetMethod, parsed, methodNames, config)
	return err
}

// GetServiceMethods returns all method names from a service file
func GetServiceMethods(servicePath string) ([]string, error) {
	parsed, err := utils.ParseServiceFile(servicePath)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(parsed.Methods))
	for i, m := range parsed.Methods {
		names[i] = m.Name
	}
	return names, nil
}

// ValidateServiceFile checks if a file is a valid gRPC service implementation
func ValidateServiceFile(servicePath string) error {
	parsed, err := utils.ParseServiceFile(servicePath)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	if len(parsed.Methods) == 0 {
		return fmt.Errorf("no service methods found in file")
	}

	if parsed.ServiceName == "" {
		return fmt.Errorf("could not determine service name")
	}

	return nil
}

// FilterImports removes unnecessary imports from the generated code
// Currently a placeholder for future optimization
func FilterImports(imports []string, body string) []string {
	var filtered []string
	for _, imp := range imports {
		// Simple heuristic: check if the package name appears in the body
		parts := strings.Split(imp, "/")
		pkgName := parts[len(parts)-1]
		if strings.Contains(body, pkgName+".") {
			filtered = append(filtered, imp)
		}
	}
	return filtered
}
