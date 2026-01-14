package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"unicode"
)

// PascalToSnake converts a PascalCase or camelCase string to snake_case.
// Example: "GetBook" -> "get_book", "GetAuthorNameFromBookId" -> "get_author_name_from_book_id"
func PascalToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ExtractMethodName extracts the method name from a receiver call like "s.GetAuthorNameFromBookId".
// Returns the method name (e.g., "GetAuthorNameFromBookId") and true if it's a receiver call,
// or empty string and false otherwise.
func ExtractMethodName(call string) (string, bool) {
	parts := strings.Split(call, ".")
	if len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
		return parts[1], true
	}
	return "", false
}

// GenerateMethodID creates a unique identifier for an RPC method based on the full package path,
// service name, and method name. This ensures uniqueness even when different services have
// methods with the same name (e.g., A.GetBook vs B.GetBook).
// Format: "{proto_package_path}.{service_name}.{method_name}"
// Example: "github.com/bsmider/pipes/core/factory/build/example.BookService.GetBook"
func GenerateMethodID(protoImportPath, serviceName, methodName string) string {
	return protoImportPath + "." + serviceName + "." + methodName
}

// GenerateShortMethodID creates a shorter but still unique identifier using a hash.
// This is useful when the full ID would be too long for practical use.
// Returns first 12 characters of SHA256 hash + method name for readability.
// Example: "a1b2c3d4e5f6_GetBook"
func GenerateShortMethodID(protoImportPath, serviceName, methodName string) string {
	fullID := GenerateMethodID(protoImportPath, serviceName, methodName)
	hash := sha256.Sum256([]byte(fullID))
	shortHash := hex.EncodeToString(hash[:])[:12]
	return shortHash + "_" + methodName
}

// GenerateDirPath creates a unique directory path for a method based on the proto package
// and method hierarchy. This creates nested directories that reflect the package structure.
// Example: "example/book_service/get_book"
func GenerateDirPath(protoImportPath, serviceName, methodName string) string {
	// Extract the proto package name from the import path (last component)
	protoPackage := filepath.Base(protoImportPath)

	// Convert service and method names to snake_case
	serviceDir := PascalToSnake(serviceName)
	methodDir := PascalToSnake(methodName)

	return filepath.Join(protoPackage, serviceDir, methodDir)
}

// SafePathFromID converts a full method ID into a safe directory path
// by replacing invalid characters with underscores.
// This provides an alternative flat directory structure using the full unique ID.
func SafePathFromID(methodID string) string {
	// Replace characters that are problematic in paths
	safe := strings.ReplaceAll(methodID, "/", "_")
	safe = strings.ReplaceAll(safe, ".", "_")
	return safe
}
