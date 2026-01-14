package factory

// MethodInfo contains information about a generated RPC method
// needed for Dockerfile and Orchestrator generation
type MethodInfo struct {
	MethodName   string // e.g. "GetBook"
	MethodID     string // Unique ID e.g. "github.com...GetBook"
	ShortID      string // Short unique ID for binaries e.g. "get_book"
	FullDirPath  string // Absolute path to the directory containing main.go
	RelativePath string // Path generated relative to the CodeGenConfig.OutputDir
}
