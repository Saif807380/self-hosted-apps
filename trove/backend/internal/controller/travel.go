package controller

import (
	"context"
	"time"

	"connectrpc.com/connect"
	travelv1 "github.com/saifkazi/trove/backend/gen/trove/v1/travel"
	"github.com/saifkazi/trove/backend/gen/trove/v1/travel/travelv1connect"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
)

type travelHandler struct {
	svc *service.TravelService
}

var _ travelv1connect.TravelServiceHandler = (*travelHandler)(nil)

func (h *travelHandler) ListLocations(
	ctx context.Context,
	req *connect.Request[travelv1.ListLocationsRequest],
) (*connect.Response[travelv1.ListLocationsResponse], error) {
	locs, err := h.svc.ListLocations(ctx, req.Msg.GetSearch())
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.ListLocationsResponse{
		Locations: mapLocations(locs),
	}), nil
}

func (h *travelHandler) GetLocation(
	ctx context.Context,
	req *connect.Request[travelv1.GetLocationRequest],
) (*connect.Response[travelv1.GetLocationResponse], error) {
	loc, err := h.svc.GetLocation(ctx, req.Msg.Id)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.GetLocationResponse{Location: mapLocation(loc)}), nil
}

func (h *travelHandler) CreateLocation(
	ctx context.Context,
	req *connect.Request[travelv1.CreateLocationRequest],
) (*connect.Response[travelv1.CreateLocationResponse], error) {
	msg := req.Msg
	if msg.City == "" || msg.Country == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	visitedFrom, err := parseDatePtr(msg.VisitedFrom)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	visitedTo, err := parseDatePtr(msg.VisitedTo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	loc, err := h.svc.CreateLocation(ctx, dao.CreateLocationParams{
		City:               msg.City,
		Country:            msg.Country,
		VisitedFrom:        visitedFrom,
		VisitedTo:          visitedTo,
		PhotoCollectionURL: msg.PhotoCollectionUrl,
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.CreateLocationResponse{Location: mapLocation(loc)}), nil
}

func (h *travelHandler) UpdateLocation(
	ctx context.Context,
	req *connect.Request[travelv1.UpdateLocationRequest],
) (*connect.Response[travelv1.UpdateLocationResponse], error) {
	msg := req.Msg
	if msg.Id == "" || msg.City == "" || msg.Country == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	visitedFrom, err := parseDatePtr(msg.VisitedFrom)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	visitedTo, err := parseDatePtr(msg.VisitedTo)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	loc, err := h.svc.UpdateLocation(ctx, dao.UpdateLocationParams{
		ID:                 uid,
		City:               msg.City,
		Country:            msg.Country,
		VisitedFrom:        visitedFrom,
		VisitedTo:          visitedTo,
		PhotoCollectionURL: msg.PhotoCollectionUrl,
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.UpdateLocationResponse{Location: mapLocation(loc)}), nil
}

func (h *travelHandler) DeleteLocation(
	ctx context.Context,
	req *connect.Request[travelv1.DeleteLocationRequest],
) (*connect.Response[travelv1.DeleteLocationResponse], error) {
	if err := h.svc.DeleteLocation(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.DeleteLocationResponse{}), nil
}

func (h *travelHandler) CreateSpot(
	ctx context.Context,
	req *connect.Request[travelv1.CreateSpotRequest],
) (*connect.Response[travelv1.CreateSpotResponse], error) {
	msg := req.Msg
	if msg.LocationId == "" || msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.LocationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	spot, err := h.svc.CreateSpot(ctx, dao.CreateSpotParams{
		LocationID:  uid,
		Name:        msg.Name,
		Description: msg.Description,
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.CreateSpotResponse{Spot: mapSpot(spot)}), nil
}

func (h *travelHandler) UpdateSpot(
	ctx context.Context,
	req *connect.Request[travelv1.UpdateSpotRequest],
) (*connect.Response[travelv1.UpdateSpotResponse], error) {
	msg := req.Msg
	if msg.Id == "" || msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	uid, err := parseUUID(msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	spot, err := h.svc.UpdateSpot(ctx, dao.UpdateSpotParams{
		ID:          uid,
		Name:        msg.Name,
		Description: msg.Description,
	})
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.UpdateSpotResponse{Spot: mapSpot(spot)}), nil
}

func (h *travelHandler) DeleteSpot(
	ctx context.Context,
	req *connect.Request[travelv1.DeleteSpotRequest],
) (*connect.Response[travelv1.DeleteSpotResponse], error) {
	if err := h.svc.DeleteSpot(ctx, req.Msg.Id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&travelv1.DeleteSpotResponse{}), nil
}

// --- mapping helpers ---

func mapLocations(locs []model.TravelLocation) []*travelv1.TravelLocation {
	result := make([]*travelv1.TravelLocation, len(locs))
	for i := range locs {
		result[i] = mapLocation(&locs[i])
	}
	return result
}

func mapLocation(l *model.TravelLocation) *travelv1.TravelLocation {
	pb := &travelv1.TravelLocation{
		Id:          l.ID.String(),
		City:        l.City,
		Country:     l.Country,
		TouristSpots: mapSpots(l.TouristSpots),
		CreatedAt:   l.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   l.UpdatedAt.Format(time.RFC3339),
	}
	if l.VisitedFrom != nil {
		s := l.VisitedFrom.Format("2006-01-02")
		pb.VisitedFrom = &s
	}
	if l.VisitedTo != nil {
		s := l.VisitedTo.Format("2006-01-02")
		pb.VisitedTo = &s
	}
	if l.PhotoCollectionURL != nil {
		pb.PhotoCollectionUrl = l.PhotoCollectionURL
	}
	return pb
}

func mapSpots(spots []model.TouristSpot) []*travelv1.TouristSpot {
	result := make([]*travelv1.TouristSpot, len(spots))
	for i := range spots {
		result[i] = mapSpot(&spots[i])
	}
	return result
}

func mapSpot(s *model.TouristSpot) *travelv1.TouristSpot {
	pb := &travelv1.TouristSpot{
		Id:         s.ID.String(),
		LocationId: s.LocationID.String(),
		Name:       s.Name,
	}
	if s.Description != nil {
		pb.Description = s.Description
	}
	return pb
}

func parseDatePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
