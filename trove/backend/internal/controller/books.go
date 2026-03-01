package controller

import (
	"context"
	"time"

	"connectrpc.com/connect"
	booksv1 "github.com/saifkazi/trove/backend/gen/trove/v1/books"
	"github.com/saifkazi/trove/backend/gen/trove/v1/books/booksv1connect"
	trovev1 "github.com/saifkazi/trove/backend/gen/trove/v1"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
)

type bookHandler struct {
	svc *service.BookService
}

var _ booksv1connect.BookServiceHandler = (*bookHandler)(nil)

func (h *bookHandler) ListBooks(
	ctx context.Context,
	req *connect.Request[booksv1.ListBooksRequest],
) (*connect.Response[booksv1.ListBooksResponse], error) {
	msg := req.Msg
	f := dao.BookFilter{
		Search: msg.GetSearch(),
		TagID:  msg.GetTagId(),
	}
	if msg.YearRead != nil {
		f.YearRead = msg.GetYearRead()
	}

	books, err := h.svc.ListBooks(ctx, f)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.ListBooksResponse{
		Books: mapBooks(books),
	}), nil
}

func (h *bookHandler) GetBook(
	ctx context.Context,
	req *connect.Request[booksv1.GetBookRequest],
) (*connect.Response[booksv1.GetBookResponse], error) {
	book, err := h.svc.GetBook(ctx, req.Msg.Id)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.GetBookResponse{Book: mapBook(book)}), nil
}

func (h *bookHandler) CreateBook(
	ctx context.Context,
	req *connect.Request[booksv1.CreateBookRequest],
) (*connect.Response[booksv1.CreateBookResponse], error) {
	msg := req.Msg
	if msg.Title == "" || msg.Author == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	p := dao.CreateBookParams{
		Title:      msg.Title,
		Author:     msg.Author,
		Rating:     int16PtrFromInt32(msg.Rating),
		Review:     msg.Review,
		CoverImage: msg.CoverImage,
		YearsRead:  int16SliceFromInt32(msg.YearsRead),
	}

	tagIDs, err := parseUUIDs(msg.TagIds)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	p.TagIDs = tagIDs

	book, err := h.svc.CreateBookWithTagIDs(ctx, p)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.CreateBookResponse{Book: mapBook(book)}), nil
}

func (h *bookHandler) UpdateBook(
	ctx context.Context,
	req *connect.Request[booksv1.UpdateBookRequest],
) (*connect.Response[booksv1.UpdateBookResponse], error) {
	msg := req.Msg
	if msg.Id == "" || msg.Title == "" || msg.Author == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	tagIDs, err := parseUUIDs(msg.TagIds)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	p := dao.UpdateBookParams{
		ID:         uid,
		Title:      msg.Title,
		Author:     msg.Author,
		Rating:     int16PtrFromInt32(msg.Rating),
		Review:     msg.Review,
		CoverImage: msg.CoverImage,
		YearsRead:  int16SliceFromInt32(msg.YearsRead),
		TagIDs:     tagIDs,
	}

	book, err := h.svc.UpdateBook(ctx, p)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.UpdateBookResponse{Book: mapBook(book)}), nil
}

func (h *bookHandler) DeleteBook(
	ctx context.Context,
	req *connect.Request[booksv1.DeleteBookRequest],
) (*connect.Response[booksv1.DeleteBookResponse], error) {
	if err := h.svc.DeleteBook(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.DeleteBookResponse{}), nil
}

func (h *bookHandler) ListTags(
	ctx context.Context,
	req *connect.Request[booksv1.ListTagsRequest],
) (*connect.Response[booksv1.ListTagsResponse], error) {
	tags, err := h.svc.ListTags(ctx)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.ListTagsResponse{Tags: mapTags(tags)}), nil
}

func (h *bookHandler) CreateTag(
	ctx context.Context,
	req *connect.Request[booksv1.CreateTagRequest],
) (*connect.Response[booksv1.CreateTagResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	tag, err := h.svc.CreateTag(ctx, req.Msg.Name)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.CreateTagResponse{
		Tag: &trovev1.Tag{Id: tag.ID.String(), Name: tag.Name},
	}), nil
}

func (h *bookHandler) ListCollections(
	ctx context.Context,
	req *connect.Request[booksv1.ListCollectionsRequest],
) (*connect.Response[booksv1.ListCollectionsResponse], error) {
	cols, err := h.svc.ListCollections(ctx)
	if err != nil {
		return nil, connectErr(err)
	}
	pbCols := make([]*booksv1.Collection, len(cols))
	for i, c := range cols {
		pbCols[i] = &booksv1.Collection{Id: c.ID.String(), Name: c.Name}
	}
	return connect.NewResponse(&booksv1.ListCollectionsResponse{Collections: pbCols}), nil
}

func (h *bookHandler) CreateCollection(
	ctx context.Context,
	req *connect.Request[booksv1.CreateCollectionRequest],
) (*connect.Response[booksv1.CreateCollectionResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	col, err := h.svc.CreateCollection(ctx, req.Msg.Name)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.CreateCollectionResponse{
		Collection: &booksv1.Collection{Id: col.ID.String(), Name: col.Name},
	}), nil
}

func (h *bookHandler) AddBookToCollection(
	ctx context.Context,
	req *connect.Request[booksv1.AddBookToCollectionRequest],
) (*connect.Response[booksv1.AddBookToCollectionResponse], error) {
	msg := req.Msg
	if err := h.svc.AddBookToCollection(ctx, msg.CollectionId, msg.BookId); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.AddBookToCollectionResponse{}), nil
}

func (h *bookHandler) RemoveBookFromCollection(
	ctx context.Context,
	req *connect.Request[booksv1.RemoveBookFromCollectionRequest],
) (*connect.Response[booksv1.RemoveBookFromCollectionResponse], error) {
	msg := req.Msg
	if err := h.svc.RemoveBookFromCollection(ctx, msg.CollectionId, msg.BookId); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&booksv1.RemoveBookFromCollectionResponse{}), nil
}

// --- mapping helpers ---

func mapBooks(books []model.Book) []*booksv1.Book {
	result := make([]*booksv1.Book, len(books))
	for i := range books {
		result[i] = mapBook(&books[i])
	}
	return result
}

func mapBook(b *model.Book) *booksv1.Book {
	pb := &booksv1.Book{
		Id:        b.ID.String(),
		Title:     b.Title,
		Author:    b.Author,
		YearsRead: int32SliceFromInt16(b.YearsRead),
		Tags:      mapTags(b.Tags),
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.Format(time.RFC3339),
	}
	if b.Rating != nil {
		v := int32(*b.Rating)
		pb.Rating = &v
	}
	if b.Review != nil {
		pb.Review = b.Review
	}
	if b.CoverImage != nil {
		pb.CoverImage = b.CoverImage
	}
	return pb
}

func mapTags(tags []model.Tag) []*trovev1.Tag {
	result := make([]*trovev1.Tag, len(tags))
	for i, t := range tags {
		result[i] = &trovev1.Tag{Id: t.ID.String(), Name: t.Name}
	}
	return result
}
