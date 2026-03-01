package controller

import (
	"context"
	"time"

	"connectrpc.com/connect"
	gamesv1 "github.com/saifkazi/trove/backend/gen/trove/v1/games"
	"github.com/saifkazi/trove/backend/gen/trove/v1/games/gamesv1connect"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
)

type gameHandler struct {
	svc *service.GameService
}

var _ gamesv1connect.GameServiceHandler = (*gameHandler)(nil)

func (h *gameHandler) ListGames(
	ctx context.Context,
	req *connect.Request[gamesv1.ListGamesRequest],
) (*connect.Response[gamesv1.ListGamesResponse], error) {
	msg := req.Msg
	f := dao.GameFilter{Search: msg.GetSearch()}
	if msg.YearPlayed != nil {
		f.YearPlayed = msg.GetYearPlayed()
	}

	games, err := h.svc.ListGames(ctx, f)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamesv1.ListGamesResponse{
		Games: mapGames(games),
	}), nil
}

func (h *gameHandler) GetGame(
	ctx context.Context,
	req *connect.Request[gamesv1.GetGameRequest],
) (*connect.Response[gamesv1.GetGameResponse], error) {
	game, err := h.svc.GetGame(ctx, req.Msg.Id)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamesv1.GetGameResponse{Game: mapGame(game)}), nil
}

func (h *gameHandler) CreateGame(
	ctx context.Context,
	req *connect.Request[gamesv1.CreateGameRequest],
) (*connect.Response[gamesv1.CreateGameResponse], error) {
	msg := req.Msg
	if msg.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	p := dao.CreateGameParams{
		Title:       msg.Title,
		Studio:      msg.Studio,
		Rating:      int16PtrFromInt32(msg.Rating),
		Review:      msg.Review,
		CoverImage:  msg.CoverImage,
		YearsPlayed: int16SliceFromInt32(msg.YearsPlayed),
	}

	game, err := h.svc.CreateGame(ctx, p)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamesv1.CreateGameResponse{Game: mapGame(game)}), nil
}

func (h *gameHandler) UpdateGame(
	ctx context.Context,
	req *connect.Request[gamesv1.UpdateGameRequest],
) (*connect.Response[gamesv1.UpdateGameResponse], error) {
	msg := req.Msg
	if msg.Id == "" || msg.Title == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	p := dao.UpdateGameParams{
		ID:          uid,
		Title:       msg.Title,
		Studio:      msg.Studio,
		Rating:      int16PtrFromInt32(msg.Rating),
		Review:      msg.Review,
		CoverImage:  msg.CoverImage,
		YearsPlayed: int16SliceFromInt32(msg.YearsPlayed),
	}

	game, err := h.svc.UpdateGame(ctx, p)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamesv1.UpdateGameResponse{Game: mapGame(game)}), nil
}

func (h *gameHandler) DeleteGame(
	ctx context.Context,
	req *connect.Request[gamesv1.DeleteGameRequest],
) (*connect.Response[gamesv1.DeleteGameResponse], error) {
	if err := h.svc.DeleteGame(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamesv1.DeleteGameResponse{}), nil
}

// --- mapping helpers ---

func mapGames(games []model.VideoGame) []*gamesv1.VideoGame {
	result := make([]*gamesv1.VideoGame, len(games))
	for i := range games {
		result[i] = mapGame(&games[i])
	}
	return result
}

func mapGame(g *model.VideoGame) *gamesv1.VideoGame {
	pb := &gamesv1.VideoGame{
		Id:          g.ID.String(),
		Title:       g.Title,
		YearsPlayed: int32SliceFromInt16(g.YearsPlayed),
		CreatedAt:   g.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   g.UpdatedAt.Format(time.RFC3339),
	}
	if g.Studio != nil {
		pb.Studio = g.Studio
	}
	if g.Rating != nil {
		v := int32(*g.Rating)
		pb.Rating = &v
	}
	if g.Review != nil {
		pb.Review = g.Review
	}
	if g.CoverImage != nil {
		pb.CoverImage = g.CoverImage
	}
	return pb
}
