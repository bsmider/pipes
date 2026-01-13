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
	orch.Spawn("GetBook", "./get_book", 1)
	orch.Spawn("GetAuthor", "./get_author", 1)

	// 1. Create the specific domain message
	bookReq := &example.GetBookRequest{
		BookId: "123-abc",
	}
	log.Printf("Created GetBookRequest for ID: %s\n", bookReq.BookId)

	ctx := (&factory.Context{
		TraceId: "trace-uuid-" + uuid.NewString()[:8], // Example trace ID
	})
	ctx.AddHop("orchestrator-main") // Record where this started

	requestPacket, err := factory.CreateRequestPacket("GetBook", ctx, bookReq, nil)
	if err != nil {
		log.Fatal("error creating request packet")
	}

	resp, err := orch.RouteRequest(requestPacket)
	if err != nil {
		log.Fatal("Failed to route request:", err)
	}
	log.Printf("Received response message ID: %s\n", resp.Id)
	resp.PrintDetails()
}
