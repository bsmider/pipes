package example

import (
	"context"
	"log"

	"github.com/bsmider/pipes/core/example/build/example"
)

func (s *BookService) GetAuthor(ctx context.Context, req *example.GetAuthorRequest) (*example.GetAuthorResponse, error) {
	log.Printf("GetAuthor")
	return nil, nil
}
