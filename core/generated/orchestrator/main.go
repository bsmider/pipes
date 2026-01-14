package main

import (
	"log"

	"github.com/bsmider/vibe/core/factory"
	"github.com/bsmider/vibe/core/factory/build/example"
	"github.com/bsmider/vibe/core/factory/orchestrator"
	"github.com/google/uuid"
)

func main() {
	orch := orchestrator.NewOrchestrator()

	if err := orch.Spawn("github.com/bsmider/vibe/core/factory/build/example.BookService.GetBook", "./49923a45240c_GetBook", 1); err != nil {
		log.Fatalf("Failed to spawn worker for %s: %v", "GetBook", err)
	}
	if err := orch.Spawn("github.com/bsmider/vibe/core/factory/build/example.BookService.GetAuthorNameFromBookId", "./b10ac789a6d1_GetAuthorNameFromBookId", 1); err != nil {
		log.Fatalf("Failed to spawn worker for %s: %v", "GetAuthorNameFromBookId", err)
	}
	// 1. Create the specific domain message
	bookReq := &example.GetBookRequest{
		BookId: "123-abc",
	}

	ctx := factory.NewContext(nil, "trace-uuid-"+uuid.NewString()[:8], []*factory.Hop{})
	ctx.AddHop("orchestrator") // Record where this started

	requestPacket, err := factory.CreateRequestPacket("github.com/bsmider/vibe/core/factory/build/example.BookService.GetBook", ctx, bookReq, nil)
	if err != nil {
		log.Fatal("error creating request packet")
	}

	resp, err := orch.RouteRequest(requestPacket)
	if err != nil {
		log.Fatal("Failed to route request:", err)
	}

	resp.PrintDetails()

	// Deserialize the payload into a GetBookResponse
	bookResp, err := factory.DeserializePacket[*example.GetBookResponse](resp)
	if err != nil {
		log.Fatal("Failed to deserialize response:", err)
	}

	if bookResp.Book != nil {
		log.Printf("Book Title: %s\n", bookResp.Book.Title)
		log.Printf("Book ID: %s\n", bookResp.Book.BookId)
		log.Printf("Author ID: %s\n", bookResp.Book.AuthorId)
	} else {
		log.Println("Response received but Book is nil")
	}

	// Block forever to keep the orchestrator running
	select {}
}
