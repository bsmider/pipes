package main

import (
	"github.com/bsmider/pipes/core/factory/orchestrator"
	"log"
)

func main() {
	orch := orchestrator.NewOrchestrator()

	if err := orch.Spawn("github.com/bsmider/pipes/core/example/build/example.BookService.GetBook", "./f5bcc3da3077_GetBook", 1); err != nil {
		log.Fatalf("Failed to spawn worker for %s: %v", "GetBook", err)
	}
	if err := orch.Spawn("github.com/bsmider/pipes/core/example/build/example.BookService.GetAuthorNameFromBookId", "./743aee161164_GetAuthorNameFromBookId", 1); err != nil {
		log.Fatalf("Failed to spawn worker for %s: %v", "GetAuthorNameFromBookId", err)
	}
	if err := orch.Spawn("github.com/bsmider/pipes/core/example/build/example.BookService.GetAuthor", "./3dbb7c569bfe_GetAuthor", 1); err != nil {
		log.Fatalf("Failed to spawn worker for %s: %v", "GetAuthor", err)
	}

	// Block forever to keep the orchestrator running
	select {}
}
