package main

import (
	"context"
	"flag"
	"log"

	"github.com/bsmider/vibe/core/factory/build/example"
	"github.com/bsmider/vibe/core/factory/processes"
)

func GetAuthorNameFromBookId(context context.Context, req *example.GetAuthorNameFromBookIdRequest) (*example.GetAuthorNameFromBookIdResponse, error) {
	return &example.GetAuthorNameFromBookIdResponse{
		AuthorName: "BREVIN SMIDER",
	}, nil
}

func main() {
	log.Println("GetAuthorNameFromBookId worker binary starting...")
	nodeID := flag.String("id", "default-worker", "The unique ID for this worker instance")
	flag.Parse()
	node := processes.GetIONode(*nodeID)
	node.Listen()
	log.Println("GetAuthorNameFromBookId worker binary listening...")
	processes.Handle(GetAuthorNameFromBookId)
	select {}
}
