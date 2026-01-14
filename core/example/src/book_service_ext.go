package example

import (
	"context"

	"github.com/bsmider/pipes/core/example/build/example"
)

func (s *BookService) GetAuthor(ctx context.Context, req *example.GetAuthorRequest) (*example.GetAuthorResponse, error) {
	return nil, nil
}
