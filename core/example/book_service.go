package example

import (
	"context"
	"log"

	"github.com/bsmider/vibe/core/factory/build/example"
)

type BookService struct {
	example.UnimplementedBookServiceServer
}

func (s *BookService) GetBook(ctx context.Context, req *example.GetBookRequest) (*example.GetBookResponse, error) {
	authorRequest := &example.GetAuthorNameFromBookIdRequest{
		BookId: req.BookId,
	}
	authorResponse, err := s.GetAuthorNameFromBookId(ctx, authorRequest)
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

func (s *BookService) GetAuthorNameFromBookId(context context.Context, req *example.GetAuthorNameFromBookIdRequest) (*example.GetAuthorNameFromBookIdResponse, error) {
	return &example.GetAuthorNameFromBookIdResponse{
		AuthorName: "BREVIN SMIDER",
	}, nil
}
