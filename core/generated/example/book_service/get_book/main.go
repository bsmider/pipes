package main

import (
	"context"
	"flag"
	"log"

	"github.com/bsmider/vibe/core/factory/build/example"
	"github.com/bsmider/vibe/core/factory/processes"
)

func GetBook(ctx context.Context, req *example.GetBookRequest) (*example.GetBookResponse, error) {
	authorRequest := &example.GetAuthorNameFromBookIdRequest{
		BookId: req.BookId,
	}
	authorResponse, err := processes.Call[*example.GetAuthorNameFromBookIdRequest, *example.GetAuthorNameFromBookIdResponse]("github.com/bsmider/vibe/core/factory/build/example.BookService.GetAuthorNameFromBookId", ctx, authorRequest)
	if err != nil {
		return nil, err
	}

	book := example.Book{
		BookId:   req.BookId,
		AuthorId: authorResponse.AuthorName,
		Title:    "this is my book",
	}

	bookResponse := example.GetBookResponse{
		Book: &book,
	}

	log.Printf("GetBook responding with: %v\n", &bookResponse)

	return &bookResponse, nil
}

func main() {
	nodeID := flag.String("id", "default-worker", "The unique ID for this worker instance")
	flag.Parse()
	node := processes.GetIONode(*nodeID)
	node.Listen()
	processes.Handle(GetBook)
	select {}
}
