package main

import (
	"log"

	"github.com/bsmider/pipes/core/factory"
	"github.com/bsmider/pipes/core/factory/build/example"
	"github.com/bsmider/pipes/core/factory/orchestrator"
	"github.com/google/uuid"
)

func main() {
	orch := orchestrator.NewOrchestrator()
	orch.Spawn("GetBook", "./get_book", 1)
	orch.Spawn("GetAuthor", "./get_author", 1)

	// 1. Create the specific domain message
	bookReq := &example.GetBookRequest{
		BookId: "123-abc",
	}

	ctx := factory.NewContext(nil, "trace-uuid-"+uuid.NewString()[:8], []*factory.Hop{})
	ctx.AddHop("orchestrator") // Record where this started

	requestPacket, err := factory.CreateRequestPacket("GetBook", ctx, bookReq, nil)
	if err != nil {
		log.Fatal("error creating request packet")
	}

	resp, err := orch.RouteRequest(requestPacket)
	if err != nil {
		log.Fatal("Failed to route request:", err)
	}

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
}
