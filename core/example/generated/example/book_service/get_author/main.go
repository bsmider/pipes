package main

import (
	"context"
	"flag"

	"github.com/bsmider/pipes/core/example/build/example"
	"github.com/bsmider/pipes/core/factory/processes"
)

func GetAuthor(ctx context.Context, req *example.GetAuthorRequest) (*example.GetAuthorResponse, error) {
	return nil, nil
}

func main() {
	nodeID := flag.String("id", "default-worker", "The unique ID for this worker instance")
	flag.Parse()
	node := processes.GetIONode(*nodeID)
	node.Listen()
	processes.Handle(GetAuthor)
	select {}
}
