package main

import (
	"context"
	"flag"

	"log"
	"github.com/bsmider/pipes/core/example/build/example"
	"github.com/bsmider/pipes/core/factory/processes"
)

func GetAuthorNameFromBookId(context context.Context, req *example.GetAuthorNameFromBookIdRequest) (*example.GetAuthorNameFromBookIdResponse, error) {
	log.Printf("GetAuthorNameFromBookId")
	return &example.GetAuthorNameFromBookIdResponse{
		AuthorName: "BREVIN SMIDER",
	}, nil
}

func main() {
	nodeID := flag.String("id", "default-worker", "The unique ID for this worker instance")
	flag.Parse()
	node := processes.GetIONode(*nodeID)
	node.Listen()
	processes.Handle(GetAuthorNameFromBookId)
	select {}
}
