package utils

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ServiceMethod represents a parsed RPC service method
type ServiceMethod struct {
	Name         string // Method name (e.g., "GetBook")
	ReceiverType string // Receiver type (e.g., "BookService")
	ReceiverName string // Receiver variable name (e.g., "s")
	CtxName      string // Context parameter name
	ReqName      string // Request parameter name
	ReqType      string // Request type (e.g., "*example.GetBookRequest")
	RespType     string // Response type (e.g., "*example.GetBookResponse")
	BodyStart    int    // Starting position of function body (after '{')
	BodyEnd      int    // Ending position of function body (before '}')
}

// RPCCall represents a call to another RPC method that needs to be transformed
type RPCCall struct {
	MethodName   string // The method being called (e.g., "GetAuthorNameFromBookId")
	CallStart    int    // Start position in source
	CallEnd      int    // End position in source (including the closing paren)
	CtxArg       string // Context argument passed
	ReqArg       string // Request argument passed
	ReqType      string // Request type inferred from the method
	RespType     string // Response type inferred from the method
	FullCallExpr string // The full call expression text
}

// ImportInfo represents an imported package
type ImportInfo struct {
	Name string // Package alias (or empty if default)
	Path string // Full import path
}

// ParsedServiceFile contains all extracted information from a service file
type ParsedServiceFile struct {
	PackageName     string
	Imports         []ImportInfo
	ServiceName     string
	Methods         []ServiceMethod
	ProtoImportPath string // The proto import path (e.g., "github.com/bsmider/pipes/core/factory/build/example")
}

// ParseServiceFile parses a Go service file and extracts RPC method information
func ParseServiceFile(filePath string) (*ParsedServiceFile, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	result := &ParsedServiceFile{
		PackageName: file.Name.Name,
		Methods:     []ServiceMethod{},
		Imports:     []ImportInfo{},
	}

	// Extract imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}

		result.Imports = append(result.Imports, ImportInfo{
			Name: name,
			Path: importPath,
		})
	}

	// Find all method declarations with receivers
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			return true
		}

		// Extract receiver info
		if len(funcDecl.Recv.List) == 0 {
			return true
		}

		recv := funcDecl.Recv.List[0]
		recvName := ""
		if len(recv.Names) > 0 {
			recvName = recv.Names[0].Name
		}

		recvType := extractTypeName(recv.Type)
		if result.ServiceName == "" && strings.HasSuffix(recvType, "Service") {
			result.ServiceName = recvType
		}

		// Extract method signature
		method := ServiceMethod{
			Name:         funcDecl.Name.Name,
			ReceiverType: recvType,
			ReceiverName: recvName,
		}

		// Extract parameters
		if funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) >= 2 {
			// First param should be context
			if len(funcDecl.Type.Params.List[0].Names) > 0 {
				method.CtxName = funcDecl.Type.Params.List[0].Names[0].Name
			}
			// Second param should be request
			if len(funcDecl.Type.Params.List[1].Names) > 0 {
				method.ReqName = funcDecl.Type.Params.List[1].Names[0].Name
			}
			method.ReqType = exprToString(funcDecl.Type.Params.List[1].Type, fset)
		}

		// Extract return types
		if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) >= 1 {
			method.RespType = exprToString(funcDecl.Type.Results.List[0].Type, fset)
		}

		// Extract body positions
		if funcDecl.Body != nil {
			method.BodyStart = int(funcDecl.Body.Lbrace)
			method.BodyEnd = int(funcDecl.Body.Rbrace)
		}

		result.Methods = append(result.Methods, method)
		return true
	})

	// Resolve ProtoImportPath by inspecting the Request Type of the first available method
	// We assume all methods in the service use types from the same proto package.
	for _, method := range result.Methods {
		if method.ReqType == "" {
			continue
		}

		// Request type is typically like "*example.GetBookRequest"
		// We want to extract "example"
		parts := strings.Split(strings.TrimPrefix(method.ReqType, "*"), ".")
		if len(parts) < 2 {
			continue
		}

		pkgAlias := parts[0]

		// Look up the import path for this alias
		for _, imp := range result.Imports {
			// Check explicit alias
			if imp.Name == pkgAlias {
				result.ProtoImportPath = imp.Path
				break
			}

			// Check default package name (last part of path)
			if imp.Name == "" {
				pathParts := strings.Split(imp.Path, "/")
				if len(pathParts) > 0 && pathParts[len(pathParts)-1] == pkgAlias {
					result.ProtoImportPath = imp.Path
					break
				}
			}
		}

		if result.ProtoImportPath != "" {
			break
		}
	}

	return result, nil
}

// GetMethodByName finds a ServiceMethod by name from a list
func GetMethodByName(methods []ServiceMethod, name string) *ServiceMethod {
	for _, m := range methods {
		if m.Name == name {
			return &m
		}
	}
	return nil
}

// extractTypeName extracts the type name from an expression, handling pointers
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		return extractTypeName(t.X) + "." + t.Sel.Name
	default:
		return ""
	}
}

// exprToString converts an AST expression to its string representation
func exprToString(expr ast.Expr, fset *token.FileSet) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X, fset)
	case *ast.SelectorExpr:
		return exprToString(t.X, fset) + "." + t.Sel.Name
	default:
		return ""
	}
}

// FindRPCCalls finds all receiver method calls in a function body that match the pattern s.MethodName(ctx, req)
// These are the calls that need to be transformed to processes.Call
func FindRPCCalls(filePath string, method ServiceMethod, serviceMethodNames []string) ([]RPCCall, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	// Build a set of known RPC method names for quick lookup
	rpcMethodSet := make(map[string]bool)
	for _, name := range serviceMethodNames {
		rpcMethodSet[name] = true
	}

	var calls []RPCCall

	// Find the function declaration for this method
	ast.Inspect(file, func(n ast.Node) bool {
		funcDecl, ok := n.(*ast.FuncDecl)
		if !ok || funcDecl.Name.Name != method.Name {
			return true
		}

		// Walk the function body to find call expressions
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
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

			call := RPCCall{
				MethodName: methodName,
				CallStart:  int(callExpr.Pos()),
				CallEnd:    int(callExpr.End()),
				CtxArg:     exprToString(callExpr.Args[0], fset),
				ReqArg:     exprToString(callExpr.Args[1], fset),
			}

			calls = append(calls, call)
			return true
		})

		return false // Found our method, stop searching
	})

	return calls, nil
}
